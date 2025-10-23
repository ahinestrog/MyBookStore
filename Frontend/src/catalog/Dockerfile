# Build
FROM golang:1.22 AS builder
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .
WORKDIR /app/Frontend/src/catalog
RUN go build -o /catalog-frontend

# Runtime
FROM gcr.io/distroless/static-debian12
WORKDIR /srv
COPY --from=builder /catalog-frontend /srv/catalog-frontend
# Los assets embebidos (embed.FS) ya están dentro del binario
ENV PORT=8080
ENV CATALOG_GRPC_ADDR=catalog:50051
EXPOSE 8080
ENTRYPOINT ["/srv/catalog-frontend"]
