FROM golang:1.16 as builder

# milvus-operator use https://github.com/milvus-io/milvus-helm's charts & values as its built dependencies
ARG MILVUS_HELM_VERSION=milvus-3.0.16

WORKDIR /workspace
# ENV GOPROXY https://goproxy.cn
# Copy the Go Modules manifests
# # cache deps before building and copying source so that we don't need to re-download as much
# # and so that source changes don't invalidate our downloaded layer
#
# # Copy the go source
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
#
# # Build
COPY main.go main.go
COPY apis/ apis/
COPY pkg/ pkg/
COPY tool/ tool/
COPY config/assets/templates out/config/assets/templates
RUN wget https://github.com/milvus-io/milvus-helm/raw/${MILVUS_HELM_VERSION}/charts/milvus/charts/etcd-6.3.3.tgz -O ./etcd.tgz
RUN wget https://github.com/milvus-io/milvus-helm/raw/${MILVUS_HELM_VERSION}/charts/milvus/charts/minio-8.0.11.tgz -O ./minio.tgz
RUN wget https://github.com/milvus-io/milvus-helm/raw/${MILVUS_HELM_VERSION}/charts/milvus/charts/pulsar-2.7.8.tgz -O ./pulsar.tgz
RUN mkdir -p ./out/config/assets/charts/
RUN wget https://github.com/milvus-io/milvus-helm/raw/${MILVUS_HELM_VERSION}/charts/milvus/values.yaml -O ./out/config/assets/charts/values.yaml
RUN tar -xf ./etcd.tgz -C ./out/config/assets/charts/
RUN tar -xf ./minio.tgz -C ./out/config/assets/charts/
RUN tar -xf ./pulsar.tgz -C ./out/config/assets/charts/
COPY scripts/run.sh out/run.sh

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o out/manager main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -a -o out/checker ./tool/checker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -a -o out/merge ./tool/merge
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -a -o out/cp ./tool/cp
#
# # Use distroless as minimal base image to package the manager binary
# # Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/out/ /

USER 65532:65532

ENTRYPOINT ["/manager"]
