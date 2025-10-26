FROM golang:1.24-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Frontend/src/user ./Frontend/src/user

WORKDIR /app/Frontend/src/user
RUN go build -o /bin/user-frontend .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /srv

COPY --from=build /bin/user-frontend /srv/user-frontend
COPY Frontend/src/user/templates /srv/templates
COPY Frontend/src/user/static /srv/static

RUN chown -R appuser:appgroup /srv/templates /srv/static
USER appuser:appgroup
ENV HTTP_ADDR=:8080 USER_GRPC_ADDR=user:50055
EXPOSE 8080
ENTRYPOINT ["/srv/user-frontend"]
