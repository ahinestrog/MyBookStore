FROM golang:1.24-alpine AS build
RUN apk add --no-cache git build-base sqlite-dev
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY proto/gen ./proto/gen
COPY Backend/src/payment/ ./Backend/src/payment/

WORKDIR /app/Backend/src/payment/src
ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN go build -o /bin/payment-service .

FROM alpine:3.20
WORKDIR /srv
RUN apk add --no-cache ca-certificates sqlite-libs \
 && adduser -D -H -u 10001 app \
 && mkdir -p /data && chown -R app:app /data

USER app:app
COPY --from=build /bin/payment-service /srv/payment-service

ENV PAYMENT_GRPC_PORT=50053 \
    PAYMENT_SQLITE_PATH=/data/payment.db \
    PAYMENT_LOG_LEVEL=info \
    PAYMENT_REQUEST_QUEUE=payment.charge.requested \
    PAYMENT_CONSUMER_TAG=payment-service \
    PAYMENT_PREFETCH_COUNT=10 \
    RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/ \
    EVENTS_EXCHANGE=mybookstore.events

VOLUME ["/data"]
EXPOSE 50053
ENTRYPOINT ["/srv/payment-service"]
