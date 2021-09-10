# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY /bin/milvus-operator .
USER nonroot:nonroot

ENTRYPOINT ["/milvus-operator"]