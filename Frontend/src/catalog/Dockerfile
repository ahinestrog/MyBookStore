FROM golang:1.24-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Frontend/src/catalog/ ./Frontend/src/catalog/

WORKDIR /app/Frontend/src/catalog
RUN go build -o /bin/catalog-frontend .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /srv

COPY --from=build /bin/catalog-frontend /srv/catalog-frontend
COPY Frontend/src/catalog/templates /srv/templates
COPY Frontend/src/catalog/static /srv/static

RUN chown -R appuser:appgroup /srv/templates /srv/static
USER appuser:appgroup

ENV FRONTEND_CATALOG_ADDR=:8081 \
    CATALOG_GRPC_ADDR=catalog:50051
EXPOSE 8081
ENTRYPOINT ["/srv/catalog-frontend"]
