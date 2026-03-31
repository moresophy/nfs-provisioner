/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	storage "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	storagehelpers "k8s.io/component-helpers/storage/volume"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller"
)

const (
	provisionerNameKey = "PROVISIONER_NAME"
)

type nfsProvisioner struct {
	client      kubernetes.Interface
	server      string
	path        string
	defaultMode os.FileMode
	defaultUid  int
	defaultGid  int
}

type pvcMetadata struct {
	data        map[string]string
	labels      map[string]string
	annotations map[string]string
	pvData      map[string]string
}

var pattern = regexp.MustCompile(`\${\.PVC\.((labels|annotations)\.(.*?)|.*?)}`)
var pvPattern = regexp.MustCompile(`\${\.PV\.(.*?)}`)

func (meta *pvcMetadata) stringParser(str string) string {
	result := pattern.FindAllStringSubmatch(str, -1)
	for _, r := range result {
		switch r[2] {
		case "labels":
			str = strings.ReplaceAll(str, r[0], meta.labels[r[3]])
		case "annotations":
			str = strings.ReplaceAll(str, r[0], meta.annotations[r[3]])
		default:
			str = strings.ReplaceAll(str, r[0], meta.data[r[1]])
		}
	}

	pvResult := pvPattern.FindAllStringSubmatch(str, -1)
	for _, r := range pvResult {
		str = strings.ReplaceAll(str, r[0], meta.pvData[r[1]])
	}

	return str
}

const (
	mountPath        = "/persistentvolumes"
	annotationPrefix = "k8s-sigs.io"
)

var _ controller.Provisioner = &nfsProvisioner{}

// fsExec runs a filesystem operation in a separate goroutine and returns an
// error if the context is cancelled or times out before the operation finishes.
// This prevents the provisioner from hanging indefinitely on a stalled NFS mount.
// The goroutine itself may remain blocked until the mount recovers — Go cannot
// interrupt a kernel-level blocking syscall — but the provisioner stays responsive.
func fsExec(ctx context.Context, op func() error) error {
	errc := make(chan error, 1)
	go func() { errc <- op() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return fmt.Errorf("filesystem operation did not complete: %w", ctx.Err())
	}
}

func (p *nfsProvisioner) Provision(ctx context.Context, options controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
	if options.PVC.Spec.Selector != nil {
		return nil, controller.ProvisioningFinished, fmt.Errorf("claim Selector is not supported")
	}
	klog.V(4).Infof("nfs provisioner: VolumeOptions %v", options)

	pvcNamespace := options.PVC.Namespace
	pvcName := options.PVC.Name

	pvName := strings.Join([]string{pvcNamespace, pvcName, options.PVName}, "-")

	metadata := &pvcMetadata{
		data: map[string]string{
			"name":      pvcName,
			"namespace": pvcNamespace,
		},
		labels:      options.PVC.Labels,
		annotations: options.PVC.Annotations,
		pvData: map[string]string{
			"name": options.PVName,
		},
	}

	fullPath := filepath.Join(mountPath, pvName)
	path := filepath.Join(p.path, pvName)

	pathPattern, exists := options.StorageClass.Parameters["pathPattern"]
	if exists {
		customPath := metadata.stringParser(pathPattern)
		if customPath != "" {
			path = filepath.Join(p.path, customPath)
			fullPath = filepath.Join(mountPath, customPath)
		}
	}

	// Check if the PVC has an annotation requesting a specific mode. Fallback to defaults if not.
	mode := p.defaultMode
	pvcMode := metadata.annotations[annotationPrefix+"/nfs-directory-mode"]
	if pvcMode != "" {
		var err error
		mode, err = getModeFromString(pvcMode)
		if err != nil {
			return nil, controller.ProvisioningFinished, fmt.Errorf("invalid directoryMode %s: %v", pvcMode, err)
		}
	}
	klog.V(4).Infof("creating path %s", fullPath)
	if err := fsExec(ctx, func() error { return os.MkdirAll(fullPath, mode) }); err != nil {
		return nil, controller.ProvisioningFinished, errors.New("unable to create directory to provision new pv: " + err.Error())
	}
	if err := fsExec(ctx, func() error { return os.Chmod(fullPath, mode) }); err != nil {
		return nil, controller.ProvisioningFinished, err
	}

	// Check if the PVC has an annotation requesting a specific UID and GID. Again, fallback to defaults if not.
	uid := p.defaultUid
	pvcUid := metadata.annotations[annotationPrefix+"/nfs-directory-uid"]
	if pvcUid != "" {
		var err error
		uid, err = getIdFromString(pvcUid)
		if err != nil {
			// No real point in returning an error here as the dir will have already been created as root:root
			// log the error and continue with the default uid
			klog.Errorf("invalid directoryUid %s: %v", pvcUid, err)
			uid = p.defaultUid
		}
	}
	gid := p.defaultGid
	pvcGid := metadata.annotations[annotationPrefix+"/nfs-directory-gid"]
	if pvcGid != "" {
		var err error
		gid, err = getIdFromString(pvcGid)
		if err != nil {
			// No real point in returning an error here as the dir will have already been created as root:root
			// log the error and continue with the default gid
			klog.Errorf("invalid directoryGid %s: %v", pvcGid, err)
			gid = p.defaultGid
		}
	}
	if err := fsExec(ctx, func() error { return os.Chown(fullPath, uid, gid) }); err != nil {
		return nil, controller.ProvisioningFinished, err
	}
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: *options.StorageClass.ReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			MountOptions:                  options.StorageClass.MountOptions,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				NFS: &v1.NFSVolumeSource{
					Server:   p.server,
					Path:     path,
					ReadOnly: false,
				},
			},
		},
	}
	return pv, controller.ProvisioningFinished, nil
}

func (p *nfsProvisioner) Delete(ctx context.Context, volume *v1.PersistentVolume) error {
	path := volume.Spec.PersistentVolumeSource.NFS.Path
	basePath := filepath.Base(path)
	oldPath := strings.Replace(path, p.path, mountPath, 1)

	if err := fsExec(ctx, func() error {
		_, err := os.Stat(oldPath)
		return err
	}); os.IsNotExist(err) {
		klog.Warningf("path %s does not exist, deletion skipped", oldPath)
		return nil
	} else if err != nil {
		return err
	}

	// Get the storage class for this volume.
	storageClass, err := p.getClassForVolume(ctx, volume)
	if err != nil {
		return err
	}

	// Determine if the "onDelete" parameter exists.
	// If it exists and has a `delete` value, delete the directory.
	// If it exists and has a `retain` value, safe the directory.
	onDelete := storageClass.Parameters["onDelete"]
	switch onDelete {
	case "delete":
		return fsExec(ctx, func() error { return os.RemoveAll(oldPath) })
	case "retain":
		return nil
	}

	// Determine if the "archiveOnDelete" parameter exists.
	// If it exists and has a false value, delete the directory.
	// Otherwise, archive it.
	archiveOnDelete, exists := storageClass.Parameters["archiveOnDelete"]
	if exists {
		archiveBool, err := strconv.ParseBool(archiveOnDelete)
		if err != nil {
			return err
		}
		if !archiveBool {
			return fsExec(ctx, func() error { return os.RemoveAll(oldPath) })
		}
	}

	archivePath := filepath.Join(mountPath, "archived-"+basePath)
	klog.V(4).Infof("archiving path %s to %s", oldPath, archivePath)
	return fsExec(ctx, func() error { return os.Rename(oldPath, archivePath) })
}

// getClassForVolume returns StorageClass.
func (p *nfsProvisioner) getClassForVolume(ctx context.Context, pv *v1.PersistentVolume) (*storage.StorageClass, error) {
	if p.client == nil {
		return nil, fmt.Errorf("cannot get kube client")
	}
	className := storagehelpers.GetPersistentVolumeClass(pv)
	if className == "" {
		return nil, fmt.Errorf("volume has no storage class")
	}
	class, err := p.client.StorageV1().StorageClasses().Get(ctx, className, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return class, nil
}

func getModeFromString(mode string) (os.FileMode, error) {
	if mode == "" {
		return os.FileMode(0o777), nil // Default to 0777, per current behavior
	}
	var modeInt int64
	var err error
	modeInt, err = strconv.ParseInt(mode, 8, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid mode %s: %v", mode, err)
	}
	if modeInt < 0 || modeInt > 0o777 {
		return 0, fmt.Errorf("mode must be between 0 and 0777, got %s", mode)
	}
	return os.FileMode(modeInt), nil
}

func getIdFromString(id string) (int, error) {
	if id == "" {
		return 0, nil // Default to 0 aka root, per current behavior
	}
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("invalid id %s: %v", id, err)
	}
	if idInt < 0 || idInt > 65535 {
		return 0, fmt.Errorf("id must be between 0 and 65535, got %s", id)
	}
	return idInt, nil
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	server := os.Getenv("NFS_SERVER")
	if server == "" {
		klog.Fatal("NFS_SERVER not set")
	}
	path := os.Getenv("NFS_PATH")
	if path == "" {
		klog.Fatal("NFS_PATH not set")
	}
	provisionerName := os.Getenv(provisionerNameKey)
	if provisionerName == "" {
		klog.Fatalf("environment variable %s is not set! Please set it.", provisionerNameKey)
	}
	// Get the default mode, uid, and gid from environment variables
	mode, err := getModeFromString(os.Getenv("NFS_DEFAULT_MODE"))
	if err != nil {
		klog.Fatalf("Failed to parse NFS_DEFAULT_MODE: %v", err)
	}
	uid, err := getIdFromString(os.Getenv("NFS_DEFAULT_UID"))
	if err != nil {
		klog.Fatalf("Failed to parse NFS_DEFAULT_UID: %v", err)
	}
	gid, err := getIdFromString(os.Getenv("NFS_DEFAULT_GID"))
	if err != nil {
		klog.Fatalf("Failed to parse NFS_DEFAULT_GID: %v", err)
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	var config *rest.Config
	if kubeconfig != "" {
		// Create an OutOfClusterConfig and use it to create a client for the controller
		// to use to communicate with Kubernetes
		var err error
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			klog.Fatalf("Failed to create kubeconfig: %v", err)
		}
	} else {
		// Create an InClusterConfig and use it to create a client for the controller
		// to use to communicate with Kubernetes
		var err error
		config, err = rest.InClusterConfig()
		if err != nil {
			klog.Fatalf("Failed to create config: %v", err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create client: %v", err)
	}

	// The controller needs to know what the server version is because out-of-tree
	// provisioners aren't officially supported until 1.5
	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		klog.Fatalf("Error getting server version: %v", err)
	}

	leaderElection := true
	leaderElectionEnv := os.Getenv("ENABLE_LEADER_ELECTION")
	if leaderElectionEnv != "" {
		leaderElection, err = strconv.ParseBool(leaderElectionEnv)
		if err != nil {
			klog.Fatalf("Unable to parse ENABLE_LEADER_ELECTION env var: %v", err)
		}
	}

	clientNFSProvisioner := &nfsProvisioner{
		client:      clientset,
		server:      server,
		path:        path,
		defaultMode: mode,
		defaultUid:  uid,
		defaultGid:  gid,
	}
	// Start the provision controller which will dynamically provision efs NFS
	// PVs
	pc := controller.NewProvisionController(clientset,
		provisionerName,
		clientNFSProvisioner,
		serverVersion.GitVersion,
		controller.LeaderElection(leaderElection),
	)
	// Never stops.
	pc.Run(context.Background())
}
