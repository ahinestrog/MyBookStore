FROM golang:1.24-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Frontend/src/inventory/ ./Frontend/src/inventory/

WORKDIR /app/Frontend/src/inventory
RUN go build -o /bin/inventory-frontend .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /srv

COPY --from=build /bin/inventory-frontend /srv/inventory-frontend
COPY Frontend/src/inventory/templates /srv/templates
COPY Frontend/src/inventory/static /srv/static

RUN chown -R appuser:appgroup /srv/templates /srv/static
USER appuser:appgroup

ENV FRONTEND_INVENTORY_ADDR=:8082 \
    INVENTORY_GRPC_ADDR=inventory:50052
EXPOSE 8082
ENTRYPOINT ["/srv/inventory-frontend"]
