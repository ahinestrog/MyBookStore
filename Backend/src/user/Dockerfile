# --------- build stage ----------
FROM golang:1.22 AS build
WORKDIR /app
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download

# Copiamos solo el servicio user y los protos generados
COPY Backend/src/user ./Backend/src/user
COPY proto/gen ./proto/gen

WORKDIR /app/Backend/src/user
RUN go build -o /usr/local/bin/user_service .

# --------- runtime stage ----------
FROM gcr.io/distroless/base-debian12
WORKDIR /
# directorio de datos para sqlite
VOLUME ["/data"]
ENV USER_GRPC_PORT=50055 USER_DB_PATH=/data/user.db RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
COPY --from=build /usr/local/bin/user_service /usr/local/bin/user_service
EXPOSE 50055
ENTRYPOINT ["/usr/local/bin/user_service"]
