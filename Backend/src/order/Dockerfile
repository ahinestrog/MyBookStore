FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Backend/src/order/ ./Backend/src/order/

WORKDIR /app/Backend/src/order/src
RUN CGO_ENABLED=1 go build -o /bin/order-service .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /data && \
    chown -R appuser:appgroup /data /srv

COPY --from=build /bin/order-service /srv/order-service
USER appuser:appgroup
VOLUME ["/data"]
ENV ORDER_GRPC_ADDR=:50053 \
    ORDER_DB_PATH=/data/order.db \
    RABBIT_URL=amqp://guest:guest@rabbitmq:5672/ \
    RABBIT_EXCHANGE=domain_events \
    CART_GRPC_ADDR=cart:50051
EXPOSE 50054
ENTRYPOINT ["/srv/order-service"]
