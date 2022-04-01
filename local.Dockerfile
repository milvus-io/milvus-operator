# build with local go to speed up 
FROM golang:1.16 as builder

# milvus-operator use https://github.com/milvus-io/milvus-helm's charts & values as its built dependencies
ARG MILVUS_HELM_VERSION=milvus-3.0.16

WORKDIR /workspace

COPY out/ out/
#
# # Use distroless as minimal base image to package the manager binary
# # Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/out/ /

USER 65532:65532

ENTRYPOINT ["/manager"]
