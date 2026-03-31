# Changelog

## v4.0.3-moresophy (2026-03-31)

Maintained fork by [Moresophy GmbH](https://github.com/moresophy). Based on upstream v4.0.3.

### Bug Fixes

- **Fix leader election crash on Kubernetes 1.29+** — The upstream provisioner used the `endpoints` resource lock for leader election, which was removed in `client-go` v0.35 (Kubernetes 1.29+). Patched the vendored `sig-storage-lib-external-provisioner` controller to use `coordination.k8s.io/leases` instead. Updated RBAC accordingly.
- **Fix node hang on NFS server failure** — All filesystem operations (`os.MkdirAll`, `os.Chmod`, `os.Chown`, `os.Stat`, `os.RemoveAll`, `os.Rename`) now run inside `fsExec(ctx, ...)` which respects the context deadline, preventing the provisioner from blocking indefinitely on a stalled NFS mount (a known issue on Talos and similar minimal Linux distributions with hard NFS mounts).
- **Fix invalid `namespace:` on PersistentVolume** — PVs are cluster-scoped resources. The Helm chart template incorrectly set `namespace: {{ .Release.Namespace }}` which was rejected by strict API validators. Removed.
- **Fix NFS soft-mount not configured** — The provisioner's own NFS mount now uses `soft,timeo=30,retrans=3` by default. This requires a PV+PVC for the provisioner's mount (mount options can only be set in PV specs, not in Pod volume specs directly). Raw manifests now include `deploy/provisioner-nfs-pv.yaml`.

### Improvements

- **Add `${.PV.name}` support in `pathPattern`** — The PV name can now be used in custom path patterns alongside existing `${.PVC.*}` variables.
- **Replace deprecated `glog` with `klog/v2`** — Switched logging from the unmaintained `github.com/golang/glog` to `k8s.io/klog/v2`. The `stderrthreshold` and `-v` flags now work correctly.
- **Add liveness/readiness probes** — Both raw manifests and the Helm chart now include `exec: ls /persistentvolumes` probes to detect a hung provisioner.
- **Remove PodSecurityPolicy** — PSP was removed in Kubernetes 1.25. Templates, RBAC rules, and values have been cleaned up.
- **Update minimum Kubernetes version** — Chart `kubeVersion` updated from `>=1.9.0-0` to `>=1.21.0-0`.

### Rebranding

- Project renamed from `nfs-subdir-external-provisioner` to `NFS-PROVISIONER`
- Go module: `github.com/moresophy/nfs-provisioner`
- Docker image: `docker.io/moresophy/nfs-provisioner`
- StorageClass provisioner name: `moresophy/nfs-provisioner`
- Helm chart: `nfs-provisioner` (chart version `4.0.3`)

---

## Upstream history (kubernetes-sigs/nfs-subdir-external-provisioner)

### v4.0.3

- Prevent mounting of root directory on empty customPath
- Upgrade k8s client to v1.23.4
- Add error handling to chmod on volume creation
- Import GetPersistentVolumeClass from component-helpers
- Resolve CVE-2022-27191 in golang.org/x/crypto
- Fix onDelete option for subdirectories
- Resolve all trivy vulnerabilities up to 2024-01-25

### v4.0.2
- Add arm7 (32bit) support

### v4.0.1
- Preserve name of the PV directory name during archiving

### v4.0.0
- Switch to `kubernetes-sigs/sig-storage-lib-external-provisioner`
- Fill in rbac.yaml with ServiceAccount manifest
- Update Deployment apiVersion to `apps/v1`
- Support running controller outside of cluster
- Add leader election disable flag
- Enable mountOptions from StorageClass to PersistentVolume

### v3.x and earlier

See the [upstream changelog](https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner/blob/master/CHANGELOG.md).
