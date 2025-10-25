FROM golang:1.24 as build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY Backend/src/order/ ./Backend/src/order/
COPY proto/gen ./proto/gen

WORKDIR /app/Backend/src/order/src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /order-service

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot
COPY --from=build /order-service /usr/local/bin/order-service
VOLUME ["/data"]
ENV ORDER_GRPC_ADDR=:50053 \
    ORDER_DB_PATH=/data/order.db \
    RABBIT_URL=amqp://guest:guest@rabbitmq:5672/ \
    RABBIT_EXCHANGE=domain_events \
    CART_GRPC_ADDR=cart:50051
EXPOSE 50054
ENTRYPOINT ["/srv/order-service"]
