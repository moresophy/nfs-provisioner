FROM gcr.io/distroless/static:latest
LABEL maintainers="moresophy"
LABEL description="NFS-PROVISIONER — dynamic NFS subdirectory provisioner for Kubernetes"
ARG binary=./bin/nfs-provisioner

COPY ${binary} /nfs-provisioner
ENTRYPOINT ["/nfs-provisioner"]
