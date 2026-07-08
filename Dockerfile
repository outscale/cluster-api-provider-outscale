# syntax=docker/dockerfile:experimental@sha256:600e5c62eedff338b3f7a0850beb7c05866e0ef27b2d2e8c02aa468e78496ff5

# Distroless debug is used to get a busybox shell
ARG RUNTIME_IMAGE_TAG=debug-dca9008b864a381b5ce97196a4d8399ac3c2fa65@sha256:ea6a51495f94a482dc431cd247bbace8f9a096ed6397005995245520ce5afcfe

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static-debian12:${RUNTIME_IMAGE_TAG}
ARG TARGETPLATFORM
WORKDIR /
COPY $TARGETPLATFORM/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
