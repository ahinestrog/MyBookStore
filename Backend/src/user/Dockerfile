FROM golang:1.24 AS build
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY Backend/src/user/ ./Backend/src/user/
COPY proto/gen ./proto/gen

WORKDIR /app/Backend/src/user/src
RUN go build -o /bin/user-service .


FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot

COPY --from=build /bin/user-service /srv/user-service
VOLUME ["/data"]

ENV USER_GRPC_PORT=50055 \
    USER_DB_PATH=/data/user.db \
    USER_LOG_LEVEL=info \
    RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/ \
    EVENTS_EXCHANGE=mybookstore.events

EXPOSE 50055
ENTRYPOINT ["/srv/user-service"]
