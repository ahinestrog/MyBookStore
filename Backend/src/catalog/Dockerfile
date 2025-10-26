FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Backend/src/catalog ./Backend/src/catalog

WORKDIR /app/Backend/src/catalog/src
RUN CGO_ENABLED=1 go build -o /bin/catalog-server .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /data && \
    chown -R appuser:appgroup /data /srv

COPY --from=build /bin/catalog-server /srv/catalog-server
COPY --from=build /app/Backend/src/catalog/src/db/db.sql /srv/db/db.sql
COPY --from=build /app/Backend/src/catalog/src/db/seed.sql /srv/db/seed.sql

USER appuser:appgroup

ENV GRPC_PORT=50051
ENV CATALOG_DB_PATH=/data/catalog.db
# Persistencia del servicio
VOLUME ["/data"]
EXPOSE 50051
ENTRYPOINT ["/srv/catalog-server"]
