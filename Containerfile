# Foundry supplies the approved, digest-pinned Go builder image.
# Foundry must approve this exact builder identity before materialization.
ARG GO_BUILDER_IMAGE=golang:1.27.1-bookworm@sha256:8d48e12ec56735e9358640898b9d9b9fcca110612ed8a5567438c0a1baa24e66
FROM ${GO_BUILDER_IMAGE} AS build

WORKDIR /src
COPY . ./

RUN GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go test -count=1 ./... \
    && GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go vet ./...

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
    go build -trimpath -ldflags="-s -w" -o /out/bce .

FROM scratch
COPY --from=build /out/bce /bce
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ARG SOURCE_REVISION=unknown
LABEL org.opencontainers.image.title="Bitmagnet Concept Explorer" \
      org.opencontainers.image.version="0.0.1" \
      org.opencontainers.image.revision="${SOURCE_REVISION}" \
      org.opencontainers.image.source="https://github.com/mark-e-deyoung/bitmagnet-concept-explorer"
USER 65532:65532
EXPOSE 3086
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/bce", "-healthcheck"]
ENTRYPOINT ["/bce"]
