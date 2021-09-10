FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
# # cache deps before building and copying source so that we don't need to re-download as much
# # and so that source changes don't invalidate our downloaded layer
#
# # Copy the go source
COPY go.mod go.mod
COPY go.sum go.sum
#
# # Build
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY config/assets/ config/assets/

RUN go mod tidy

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
#
# # Use distroless as minimal base image to package the manager binary
# # Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
#
ENTRYPOINT ["/manager"]