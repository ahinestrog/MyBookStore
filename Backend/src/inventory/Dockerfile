FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Backend/src/inventory/src ./Backend/src/inventory/src

WORKDIR /app/Backend/src/inventory/src
RUN CGO_ENABLED=1 go build -o /bin/inventory .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /data && \
    chown -R appuser:appgroup /data /srv

COPY --from=build /bin/inventory /srv/inventory
USER appuser:appgroup

VOLUME ["/data"]

ENV INVENTORY_DB_PATH=/data/inventory.db
ENV INVENTORY_GRPC_ADDR=:50053
ENV RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
ENV INVENTORY_EVENTS_EXCHANGE=mybookstore.events

EXPOSE 50053
ENTRYPOINT ["/srv/inventory"]
