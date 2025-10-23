FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY Frontend/src/user ./Frontend/src/user
COPY proto/gen ./proto/gen

WORKDIR /app/Frontend/src/user
RUN go build -o /usr/local/bin/frontend_user .

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /usr/local/bin/frontend_user /usr/local/bin/frontend_user
COPY --from=build /app/Frontend/src/user/templates ./templates
COPY --from=build /app/Frontend/src/user/static ./static
ENV USER_GRPC_ADDR=user:50055 HTTP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/frontend_user"]
