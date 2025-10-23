# Build
FROM golang:1.22 AS builder
WORKDIR /app
ENV CGO_ENABLED=1 GOOS=linux

# Go deps (desde la raíz del repo, para importar proto/gen/..)
COPY go.mod go.sum ./
RUN go mod download

COPY . .
WORKDIR /app/Backend/src/catalog/src
RUN go build -o /catalog-server

# Runtime
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot

COPY --from=builder /catalog-server /srv/catalog-server
COPY --from=builder /app/Backend/src/catalog/src/db.sql /srv/src/db.sql
COPY --from=builder /app/Backend/src/catalog/src/seed.sql /srv/src/seed.sql

ENV GRPC_PORT=50051
ENV CATALOG_DB_PATH=./data/catalog.db
EXPOSE 50051
ENTRYPOINT ["/srv/catalog-server"]
