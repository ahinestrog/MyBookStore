FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Backend/src/user/ ./Backend/src/user/

WORKDIR /app/Backend/src/user/src
RUN CGO_ENABLED=1 go build -o /bin/user-service .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /data && \
    chown -R appuser:appgroup /data /srv

COPY --from=build /bin/user-service /srv/user-service
USER appuser:appgroup
VOLUME ["/data"]

ENV USER_GRPC_PORT=50055 \
    USER_DB_PATH=/data/user.db \
    USER_LOG_LEVEL=info \
    RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/ \
    EVENTS_EXCHANGE=mybookstore.events

EXPOSE 50055
ENTRYPOINT ["/srv/user-service"]
