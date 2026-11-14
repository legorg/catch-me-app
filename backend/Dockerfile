FROM golang:1.26-alpine AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY api ./api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM registry.opencode.de/oci-community/images/zendis/static:1.0.0-minimal@sha256:b9d32313fd20a8f6aa9811e087ef033cf6d36fd3f5f3c390d135ecee39936cf5 AS runtime
WORKDIR /app
COPY --from=source /out/server ./server
ENTRYPOINT ["/app/server"]
CMD ["-db", "/data/catch-me.sqlite"]
