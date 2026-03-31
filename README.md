# NFS-PROVISIONER

**NFS-PROVISIONER** is a dynamic Kubernetes storage provisioner maintained by [Moresophy GmbH](https://github.com/moresophy). It automatically creates subdirectories on an existing NFS server for every PersistentVolumeClaim and cleans them up on deletion.

> **Fork notice:** This project is a maintained fork of [kubernetes-sigs/nfs-subdir-external-provisioner](https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner). It ships with critical bug fixes for modern Kubernetes (1.21+) that have not been merged upstream, including a fix for the broken leader election on client-go ≥ v0.29, removal of the deprecated PodSecurityPolicy, and NFS soft-mount options to prevent node hangs.

## Fixes over upstream

| # | Fix |
|---|-----|
| Leader election crash (k8s 1.29+) | `endpoints` lock removed in client-go v0.35 — patched to use `coordination.k8s.io/leases` |
| Node hang on NFS failure | All filesystem operations now respect context timeout via `fsExec(ctx, ...)` |
| RBAC wrong API group | Role updated from `""/endpoints` → `coordination.k8s.io/leases` |
| PodSecurityPolicy | Removed — PSP was deleted in Kubernetes 1.25 |
| NFS soft-mount | `soft,timeo=30,retrans=3` set by default on the provisioner's own NFS mount |
| `${.PV.name}` in pathPattern | PV name now usable in custom path templates |
| Logging | Replaced deprecated `glog` with `klog/v2`; `stderrthreshold` flag now works |
| PersistentVolume namespace | Removed invalid `namespace:` field from PV (PVs are cluster-scoped) |

## Quick Start

### With Helm

```console
helm repo add nfs-provisioner https://moresophy.github.io/nfs-provisioner/
helm install nfs-provisioner nfs-provisioner/nfs-provisioner \
    --set nfs.server=<YOUR_NFS_SERVER> \
    --set nfs.path=/exported/path
```

### With Kustomize

**Step 1:** Create a `kustomization.yaml`:

```yaml
namespace: nfs-provisioner
resources:
  - https://github.com/moresophy/nfs-provisioner//deploy
  - namespace.yaml
patches:
  - patch_nfs_details.yaml
```

**Step 2:** Create a namespace:

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nfs-provisioner
```

**Step 3:** Patch the deployment with your NFS server details:

```yaml
# patch_nfs_details.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-provisioner
spec:
  template:
    spec:
      containers:
        - name: nfs-provisioner
          env:
            - name: NFS_SERVER
              value: <YOUR_NFS_SERVER_IP>
            - name: NFS_PATH
              value: <YOUR_NFS_SHARE_PATH>
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: nfs-provisioner-root
spec:
  nfs:
    server: <YOUR_NFS_SERVER_IP>
    path: <YOUR_NFS_SHARE_PATH>
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-provisioner-root
  namespace: nfs-provisioner
```

**Step 4:** Deploy:

```sh
kubectl apply -k .
```

### Manually

**Step 1:** Set your namespace:

```sh
NAMESPACE=default  # or your target namespace
sed -i "s/namespace:.*/namespace: $NAMESPACE/g" deploy/rbac.yaml deploy/deployment.yaml deploy/provisioner-nfs-pv.yaml
```

**Step 2:** Edit `deploy/provisioner-nfs-pv.yaml` and `deploy/deployment.yaml` — replace `10.3.243.101` and `/ifs/kubernetes` with your NFS server IP and export path.

**Step 3:** Apply:

```sh
kubectl apply -f deploy/rbac.yaml
kubectl apply -f deploy/provisioner-nfs-pv.yaml
kubectl apply -f deploy/deployment.yaml
kubectl apply -f deploy/class.yaml
```

**Step 4:** Test:

```sh
kubectl apply -f deploy/test-claim.yaml -f deploy/test-pod.yaml
# Check your NFS server for a SUCCESS file in the new directory
kubectl delete -f deploy/test-pod.yaml -f deploy/test-claim.yaml
```

**Step 5:** Deploy your own PVCs:

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: my-pvc
spec:
  storageClassName: nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 5Gi
```

## StorageClass Parameters

| Parameter | Description | Default |
|---|---|---|
| `onDelete` | `delete` — remove directory; `retain` — keep directory | archive |
| `archiveOnDelete` | `false` — delete; `true` — rename to `archived-<name>`. Ignored if `onDelete` is set | `true` |
| `pathPattern` | Template for the subdirectory name. Supports `${.PVC.namespace}`, `${.PVC.name}`, `${.PVC.labels.*}`, `${.PVC.annotations.*}`, `${.PV.name}` | `<namespace>-<pvcName>-<pvName>` |

**StorageClass example:**

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs
provisioner: moresophy/nfs-provisioner
parameters:
  pathPattern: "${.PVC.namespace}/${.PVC.name}"
  onDelete: delete
```

## Permissions and Ownership

Control directory permissions via environment variables or per-PVC annotations:

| Environment Variable | Default | Description |
|---|---|---|
| `NFS_DEFAULT_MODE` | `777` | Octal directory permissions |
| `NFS_DEFAULT_UID` | `0` | Default owner UID |
| `NFS_DEFAULT_GID` | `0` | Default owner GID |

Per-PVC annotations (override defaults):

- `k8s-sigs.io/nfs-directory-mode` — e.g. `"750"`
- `k8s-sigs.io/nfs-directory-uid` — e.g. `"1000"`
- `k8s-sigs.io/nfs-directory-gid` — e.g. `"1000"`

## Building Your Own Image

Multi-arch (amd64, arm64, arm/v7):

```sh
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  --file Dockerfile.multiarch \
  --build-arg APP_FOLDER=/go/src/github.com/moresophy/nfs-provisioner \
  --tag moresophy/nfs-provisioner:v4.0.3 \
  --push .
```

Single-arch (current platform):

```sh
make build
# Binary → ./bin/nfs-provisioner
```

## Automated Releases via GitHub Actions

Push a tag matching `gh-v{major}.{minor}.{patch}` to trigger the release workflow, which builds and pushes a multi-arch image to `docker.io/moresophy/nfs-provisioner` with tags `latest`, `{major}`, `{major}.{minor}`, `{major}.{minor}.{patch}`.

Required repository secrets: `REGISTRY_USERNAME`, `REGISTRY_TOKEN`, `DOCKER_IMAGE`.

## Known Limitations

- Storage capacity is not enforced — the application can write beyond the requested PVC size.
- Storage resize is not supported.
- Requires an already-configured NFS server with an exported share.

## Maintainers

| Name | GitHub | Email |
|---|---|---|
| Sebastian Broers | [@natorus87](https://github.com/natorus87) | sebastian.broers@moresophy.com |
| Moresophy GmbH | [@moresophy](https://github.com/moresophy) | — |

## License

Apache 2.0 — see [LICENSE](LICENSE).

Upstream project: [kubernetes-sigs/nfs-subdir-external-provisioner](https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner) (Apache 2.0)
