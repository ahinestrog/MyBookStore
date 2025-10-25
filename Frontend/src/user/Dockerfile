FROM golang:1.24 AS build
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
COPY go.mod go.sum ./
RUN go mod download

COPY Frontend/src/user ./Frontend/src/user
COPY proto/gen ./proto/gen

WORKDIR /app/Frontend/src/user
RUN go build -o /bin/user-frontend .

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
USER nonroot:nonroot
COPY --from=build /bin/user-frontend /bin/user-frontend
COPY --from=build /app/Frontend/src/user/templates /srv/templates
COPY --from=build /app/Frontend/src/user/static    /srv/static
ENV HTTP_ADDR=:8080 USER_GRPC_ADDR=user:50055
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/frontend_user"]
