FROM golang:1.16 as builder

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
COPY config/assets/ out/config/assets/
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
