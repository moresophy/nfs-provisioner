# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Does

NFS-PROVISIONER is a Kubernetes dynamic storage provisioner maintained by moresophy. It creates subdirectories on an existing NFS share when PersistentVolumeClaims are created, and cleans them up (delete, retain, or archive) on deletion. It is a fork of the original `kubernetes-sigs/nfs-subdir-external-provisioner`, with bug fixes for modern Kubernetes versions (1.21+).

## Commands

### Build

```bash
make build                  # Build binary for current platform → ./bin/nfs-provisioner
make container              # Build Docker image (single-arch, Dockerfile)
```

Multi-arch image (amd64, arm64, arm/v7) via Docker Buildx:
```bash
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  --file Dockerfile.multiarch \
  --build-arg APP_FOLDER=/go/src/github.com/moresophy/nfs-provisioner \
  --tag moresophy/nfs-provisioner:v4.0.3 \
  --push .
```

### Test

```bash
make test           # All checks (go test, vet, fmt, vendor, boilerplate)
make test-go        # Unit tests only
make test-vet       # go vet
make test-fmt       # gofmt check
make test-vendor    # Verify vendor directory consistency
make test V=1       # Verbose
```

Single test:
```bash
go test -v ./cmd/nfs-provisioner/... -run TestName
```

### Lint

```bash
make test-fmt
make test-vet
make test-boilerplate
make test-shellcheck    # requires Docker
make test-spelling
```

## Architecture

### Core Logic

All application logic lives in [cmd/nfs-provisioner/provisioner.go](cmd/nfs-provisioner/provisioner.go).

The `nfsProvisioner` struct implements `sigs.k8s.io/sig-storage-lib-external-provisioner/v6/controller.Provisioner` with two methods:

- **`Provision(ctx, options)`** — Creates a subdirectory on the NFS mount at `/persistentvolumes`. Path can be templated via PVC metadata: `${.PVC.namespace}`, `${.PVC.name}`, `${.PVC.labels.*}`, `${.PVC.annotations.*}`, `${.PV.name}`. Sets UID/GID/mode. Returns a PV pointing to the subdirectory.

- **`Delete(ctx, volume)`** — Deletes, retains, or renames to `archived-<name>` based on `onDelete`/`archiveOnDelete` StorageClass parameters.

All filesystem operations (`os.MkdirAll`, `os.Chmod`, `os.Chown`, `os.Stat`, `os.RemoveAll`, `os.Rename`) run inside `fsExec(ctx, ...)` which returns immediately when the context times out, preventing the provisioner from hanging indefinitely on a stalled NFS mount.

### Runtime Configuration (Environment Variables)

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `PROVISIONER_NAME` | yes | — | Must match the StorageClass `provisioner:` field (`moresophy/nfs-provisioner`) |
| `NFS_SERVER` | yes | — | NFS server hostname/IP |
| `NFS_PATH` | yes | — | Base export path on the NFS server |
| `NFS_DEFAULT_MODE` | no | `0777` | Default octal directory permissions |
| `NFS_DEFAULT_UID` | no | `0` | Default owner UID |
| `NFS_DEFAULT_GID` | no | `0` | Default owner GID |
| `ENABLE_LEADER_ELECTION` | no | `true` | Enable leader election for HA |

Per-PVC annotation overrides: `k8s-sigs.io/nfs-directory-mode`, `nfs-directory-uid`, `nfs-directory-gid`.

### Deployment

The provisioner runs as a Kubernetes Deployment. The NFS share is mounted at `/persistentvolumes` via a PVC (with `soft,timeo=30,retrans=3` mount options) backed by a static PV — this is required because `mountOptions` can only be set on PersistentVolume specs, not Pod volume specs.

RBAC grants permissions on `coordination.k8s.io/leases` (leader election) and cluster-scoped PV management.

Raw manifests: [deploy/](deploy/) (+ Kustomize)  
Helm chart: [charts/nfs-provisioner/](charts/nfs-provisioner/)

### Key Compatibility Notes

- **Minimum Kubernetes: 1.21** — uses `coordination.k8s.io/leases` for leader election (endpoints lock was removed in client-go v0.35+)
- The vendored `sig-storage-lib-external-provisioner/v6/controller/controller.go` has been patched to use `resourcelock.LeasesResourceLock` instead of `"endpoints"`
- PodSecurityPolicy support has been removed (PSP was removed in Kubernetes 1.25)

### Build System

`Makefile` sets `CMDS=nfs-provisioner` and delegates to `release-tools/build.make` (a git subtree from the Kubernetes SIG Storage build framework). Do not edit `release-tools/` directly — update via subtree.

The `Dockerfile.multiarch` uses `go mod download` + `-mod=mod` instead of the vendor directory, because the vendor directory may be incomplete after dependency updates.

### CI

- `.github/workflows/release.yml` — builds and pushes to `docker.io/moresophy/nfs-provisioner` on `gh-v*.*.*` tags
- `.github/workflows/helm-chart-lint.yml` — runs `ct lint` on PRs
