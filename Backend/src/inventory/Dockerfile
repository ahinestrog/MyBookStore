# Etapa build
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git build-base
COPY go.mod go.sum ./
RUN go mod download

# Copiamos solo lo necesario para compilar inventory
COPY Backend/src/inventory/src ./Backend/src/inventory/src
COPY proto ./proto

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /inventory \
  ./Backend/src/inventory/src

# Etapa runtime
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /inventory /usr/local/bin/inventory

# Crea carpeta data por defecto para el .db
RUN mkdir -p /data
ENV INVENTORY_DB_PATH=/data/inventory.db
ENV INVENTORY_GRPC_ADDR=:50053
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/

EXPOSE 50053
ENTRYPOINT ["/usr/local/bin/inventory"]
