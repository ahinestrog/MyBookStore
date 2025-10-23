# Backend/src/payment/src/Dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /app

# Dependencias del sistema para SQLite
RUN apk add --no-cache build-base sqlite-dev

# Copia del repo
COPY go.mod go.sum ./
RUN go mod download

# Copiamos todo el monorepo para que el import de proto/gen/* funcione
COPY . .

WORKDIR /app/Backend/src/payment/src
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /bin/payment-service .

FROM alpine:3.20
WORKDIR /srv
RUN apk add --no-cache ca-certificates sqlite-libs
ENV PAYMENT_GRPC_PORT=50053 \
    PAYMENT_SQLITE_PATH=/data/payment.db \
    EVENTS_EXCHANGE=mybookstore.events \
    PAYMENT_REQUEST_QUEUE=payment.charge.requested
EXPOSE 50053
VOLUME ["/data"]
COPY --from=build /bin/payment-service /usr/local/bin/payment-service
ENTRYPOINT ["/usr/local/bin/payment-service"]
