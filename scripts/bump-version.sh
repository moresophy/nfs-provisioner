#!/usr/bin/env bash
# Usage: scripts/bump-version.sh <new-version>
# Example: scripts/bump-version.sh 4.0.4
set -euo pipefail

NEW_VERSION="${1:-}"
if [[ -z "$NEW_VERSION" ]]; then
  echo "Usage: $0 <new-version>" >&2
  exit 1
fi

# Strip leading 'v' if present
NEW_VERSION="${NEW_VERSION#v}"

OLD_VERSION="$(cat VERSION)"

if [[ "$NEW_VERSION" == "$OLD_VERSION" ]]; then
  echo "Already at version $NEW_VERSION, nothing to do." >&2
  exit 0
fi

echo "Bumping $OLD_VERSION → $NEW_VERSION"

# VERSION file
echo "$NEW_VERSION" > VERSION

# Chart.yaml: version and appVersion
sed -i "s/^version: .*/version: ${NEW_VERSION}/" charts/nfs-provisioner/Chart.yaml
sed -i "s/^appVersion: .*/appVersion: ${NEW_VERSION}/" charts/nfs-provisioner/Chart.yaml

# values.yaml: image.tag
sed -i "s/^\(\s*tag:\s*\)v.*/\1v${NEW_VERSION}/" charts/nfs-provisioner/values.yaml

# deploy/deployment.yaml: container image tag
sed -i "s|\(image: moresophy/nfs-provisioner:\)v[0-9.]*|\1v${NEW_VERSION}|" deploy/deployment.yaml

echo "Done. Files updated:"
echo "  VERSION"
echo "  charts/nfs-provisioner/Chart.yaml"
echo "  charts/nfs-provisioner/values.yaml"
echo "  deploy/deployment.yaml"
echo ""
echo "Next steps:"
echo "  git add VERSION charts/nfs-provisioner/Chart.yaml charts/nfs-provisioner/values.yaml deploy/deployment.yaml"
echo "  git commit -m \"chore: bump version to ${NEW_VERSION}\""
echo "  git tag gh-v${NEW_VERSION}"
echo "  git push && git push origin gh-v${NEW_VERSION}"
