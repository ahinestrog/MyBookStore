FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download
COPY proto/gen ./proto/gen
COPY Backend/src/cart/ ./Backend/src/cart/

WORKDIR /app/Backend/src/cart/src
RUN CGO_ENABLED=1 go build -o /bin/cart .

FROM alpine:3.18
WORKDIR /srv

RUN addgroup -S appgroup && adduser -S appuser -G appgroup && \
    mkdir -p /data && \
    chown -R appuser:appgroup /data /srv

COPY --from=build /bin/cart /srv/cart
COPY Backend/src/cart/src/sql /srv/sql
EXPOSE 50050
USER appuser:appgroup
ENTRYPOINT ["/srv/cart"]
