FROM golang:1.24 AS builder
WORKDIR /app
ENV CGO_ENABLED=1 GOOS=linux

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Backend/src/catalog/ ./Backend/src/catalog/

WORKDIR /app/Backend/src/catalog/src
RUN go build -o /catalog-server

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot

COPY --from=builder /catalog-server /srv/catalog-server
COPY --from=builder /app/Backend/src/catalog/src/db/db.sql /srv/src/db/db.sql
COPY --from=builder /app/Backend/src/catalog/src/db/seed.sql /srv/src/db/seed.sql

ENV GRPC_PORT=50051
ENV CATALOG_DB_PATH=./data/catalog.db
# Persistencia del servicio
VOLUME ["/data"]
EXPOSE 50051
ENTRYPOINT ["/srv/catalog-server"]
