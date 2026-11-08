FROM golang:1.26-alpine AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY api ./api

FROM source AS test
RUN go test ./...
RUN go vet ./...

FROM test AS build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.23 AS runtime
RUN apk add --no-cache ca-certificates
RUN addgroup -S app && adduser -S -G app app
RUN mkdir /data && chown app:app /data
WORKDIR /app
COPY --from=build /out/server ./server
USER app:app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["/app/server"]
CMD ["-db", "/data/catch-me.sqlite"]
