# Contributing to NFS-PROVISIONER

Thank you for your interest in contributing to NFS-PROVISIONER, maintained by [Moresophy GmbH](https://github.com/moresophy).

## How to Contribute

1. **Fork** the repository at https://github.com/moresophy/nfs-provisioner
2. **Create a branch** for your change: `git checkout -b fix/my-fix`
3. **Make your changes** and ensure the code compiles: `make build`
4. **Run checks**: `make test`
5. **Submit a pull request** against the `master` branch

## Code Style

- Follow standard Go formatting: `gofmt -w .`
- Run `go vet ./...` before submitting
- Keep changes focused — one PR per fix/feature

## Reporting Issues

Please use the [GitHub issue tracker](https://github.com/moresophy/nfs-provisioner/issues).

Include:
- Kubernetes version
- NFS server type and version
- Provisioner logs (`kubectl logs -n <namespace> deployment/nfs-provisioner`)
- Steps to reproduce

## Maintainers

| Name | GitHub |
|---|---|
| Sebastian Broers | [@natorus87](https://github.com/natorus87) |
| Moresophy GmbH | [@moresophy](https://github.com/moresophy) |

## Upstream

This project is a fork of [kubernetes-sigs/nfs-subdir-external-provisioner](https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner). Bug fixes that are applicable upstream are encouraged to be submitted there as well.

## License

By contributing, you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).
