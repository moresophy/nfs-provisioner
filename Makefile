# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

CMDS=nfs-provisioner
all: build

include release-tools/build.make

 BUILD_PLATFORMS=linux amd64 amd64; linux arm arm -arm; linux arm64 arm64 -arm64; linux ppc64le ppc64le -ppc64le; linux s390x s390x -s390x

# Release: bump version, commit, and tag for CI to build and push the Docker image.
# Usage: make release VERSION=4.0.4
.PHONY: release
release:
	@test -n "$(VERSION)" || (echo "Usage: make release VERSION=x.y.z" && exit 1)
	@bash scripts/bump-version.sh $(VERSION)
	git add VERSION charts/nfs-provisioner/Chart.yaml charts/nfs-provisioner/values.yaml deploy/deployment.yaml
	git commit -m "chore: bump version to $(VERSION)"
	git tag gh-v$(VERSION)
	git push
	git push origin gh-v$(VERSION)
