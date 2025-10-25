FROM golang:1.24 AS builder
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
      git build-essential ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download

COPY Backend/src/inventory/src ./Backend/src/inventory/src
COPY proto/gen ./proto/gen

WORKDIR /app/Backend/src/inventory/src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/inventory

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot
COPY --from=builder /bin/inventory /srv/inventory

VOLUME ["/data"]

ENV INVENTORY_DB_PATH=/data/inventory.db
ENV INVENTORY_GRPC_ADDR=:50053
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
ENV INVENTORY_EVENTS_EXCHANGE=mybookstore.events

EXPOSE 50053
ENTRYPOINT ["/srv/inventory"]
