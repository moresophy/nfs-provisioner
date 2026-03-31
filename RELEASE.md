# Release Process

## Versioning

NFS-PROVISIONER uses the version format `v{major}.{minor}.{patch}-moresophy` for fork-specific releases, e.g. `v4.0.3-moresophy`.

## Docker Image

Images are published to `docker.io/moresophy/nfs-provisioner` with tags:
- `latest`
- `{major}` (e.g. `4`)
- `{major}.{minor}` (e.g. `4.0`)
- `{major}.{minor}.{patch}` (e.g. `4.0.3`)

## Automated Release (GitHub Actions)

1. Update `CHANGELOG.md` and `charts/nfs-provisioner/Chart.yaml` (version + appVersion)
2. Commit: `git commit -m "chore: release vX.Y.Z"`
3. Tag: `git tag gh-vX.Y.Z`
4. Push: `git push origin master --tags`

The [release workflow](.github/workflows/release.yml) will automatically build and push the multi-arch image.

Required repository secrets:
- `REGISTRY_USERNAME` — Docker Hub username (`moresophy`)
- `REGISTRY_TOKEN` — Docker Hub access token
- `DOCKER_IMAGE` — Image name (`moresophy/nfs-provisioner`)

## Manual Release

```sh
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/arm/v7 \
  --file Dockerfile.multiarch \
  --build-arg APP_FOLDER=/go/src/github.com/moresophy/nfs-provisioner \
  --tag moresophy/nfs-provisioner:vX.Y.Z \
  --tag moresophy/nfs-provisioner:latest \
  --push .
```

## Maintainers

| Name | GitHub |
|---|---|
| Sebastian Broers | [@natorus87](https://github.com/natorus87) |
| Moresophy GmbH | [@moresophy](https://github.com/moresophy) |
