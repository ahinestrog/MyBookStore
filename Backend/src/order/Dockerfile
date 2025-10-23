FROM golang:1.22 as build

WORKDIR /app
# Copia go.mod/sum del repo raíz para resolver módulos
COPY go.mod go.sum ./
RUN go mod download

# Copia sólo el microservicio de order y los protos generados
COPY Backend/src/order/src ./Backend/src/order/src
COPY proto/gen ./proto/gen

WORKDIR /app/Backend/src/order/src
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /order-service

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
ENV ORDER_GRPC_ADDR=:50053 \
    ORDER_DB_PATH=/data/order.db \
    RABBIT_URL=amqp://guest:guest@rabbitmq:5672/ \
    RABBIT_EXCHANGE=domain_events \
    CART_GRPC_ADDR=cart:50051
VOLUME ["/data"]
EXPOSE 50053
COPY --from=build /order-service /usr/local/bin/order-service
ENTRYPOINT ["/usr/local/bin/order-service"]
