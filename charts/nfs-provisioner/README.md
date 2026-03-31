# NFS-PROVISIONER Helm Chart

Helm chart for [NFS-PROVISIONER](https://github.com/moresophy/nfs-provisioner) — a dynamic Kubernetes storage provisioner by [Moresophy GmbH](https://github.com/moresophy) that automatically provisions subdirectories on an existing NFS server.

## TL;DR

```console
helm repo add nfs-provisioner https://moresophy.github.io/nfs-provisioner/
helm install nfs-provisioner nfs-provisioner/nfs-provisioner \
    --set nfs.server=x.x.x.x \
    --set nfs.path=/exported/path
```

## Prerequisites

- Kubernetes ≥ 1.21
- Existing NFS server with an exported share

## Installing the Chart

```console
helm install my-release nfs-provisioner/nfs-provisioner \
    --set nfs.server=x.x.x.x \
    --set nfs.path=/exported/path
```

## Uninstalling the Chart

```console
helm delete my-release
```

## Configuration

| Parameter | Description | Default |
|---|---|---|
| `replicaCount` | Number of provisioner replicas | `1` |
| `strategyType` | Deployment update strategy | `Recreate` |
| `image.repository` | Container image | `moresophy/nfs-provisioner` |
| `image.tag` | Image tag | `v4.0.3` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `nfs.server` | NFS server hostname/IP (required) | — |
| `nfs.path` | NFS export path | `/nfs-storage` |
| `nfs.mountOptions` | Mount options for the provisioner's NFS mount | `[soft, timeo=30, retrans=3]` |
| `nfs.volumeName` | Internal volume name | `nfs-provisioner-root` |
| `nfs.reclaimPolicy` | Reclaim policy for the provisioner's NFS PV | `Retain` |
| `storageClass.create` | Create a StorageClass | `true` |
| `storageClass.name` | StorageClass name | `nfs` |
| `storageClass.provisionerName` | Override provisioner name | `moresophy/nfs-provisioner` |
| `storageClass.defaultClass` | Set as default StorageClass | `false` |
| `storageClass.allowVolumeExpansion` | Allow volume expansion | `true` |
| `storageClass.reclaimPolicy` | PV reclaim policy | `Delete` |
| `storageClass.archiveOnDelete` | Archive directory on PVC deletion | `true` |
| `storageClass.onDelete` | `delete` or `retain` — overrides `archiveOnDelete` | — |
| `storageClass.pathPattern` | Directory name template (supports `${.PVC.*}`, `${.PV.name}`) | — |
| `storageClass.accessModes` | PV access mode | `ReadWriteOnce` |
| `storageClass.volumeBindingMode` | Volume binding mode | `Immediate` |
| `storageClass.annotations` | Extra StorageClass annotations | `{}` |
| `leaderElection.enabled` | Enable leader election | `true` |
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create ServiceAccount | `true` |
| `serviceAccount.name` | ServiceAccount name | — |
| `serviceAccount.annotations` | ServiceAccount annotations | `{}` |
| `resources` | CPU/memory resource requests/limits | `{}` |
| `nodeSelector` | Node labels for pod assignment | `{}` |
| `affinity` | Pod affinity rules | `{}` |
| `tolerations` | Node taints to tolerate | `[]` |
| `podAnnotations` | Extra pod annotations | `{}` |
| `podSecurityContext` | Pod security context | `{}` |
| `securityContext` | Container security context | `{}` |
| `priorityClassName` | Pod priority class | — |
| `labels` | Extra labels for all resources | `{}` |
| `podDisruptionBudget.enabled` | Create a PodDisruptionBudget | `false` |
| `podDisruptionBudget.maxUnavailable` | Max unavailable pods | `1` |
| `livenessProbe` | Liveness probe config | `exec: ls /persistentvolumes` |
| `readinessProbe` | Readiness probe config | `exec: ls /persistentvolumes` |

## Multiple Provisioners

To use multiple NFS servers or exports, install multiple releases with different provisioner names:

```console
helm install nfs-provisioner-2 nfs-provisioner/nfs-provisioner \
    --set nfs.server=y.y.y.y \
    --set nfs.path=/other/export \
    --set storageClass.name=nfs-2 \
    --set storageClass.provisionerName=moresophy/nfs-provisioner-2
```

## Source

- Chart and provisioner: https://github.com/moresophy/nfs-provisioner
- Upstream: https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner
