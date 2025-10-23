# Frontend/src/order/Dockerfile
FROM golang:1.22 AS build

WORKDIR /app
# módulos del repo raíz
COPY go.mod go.sum ./
RUN go mod download

# Código del frontend y stubs de proto
COPY Frontend/src/order /app/Frontend/src/order
COPY proto/gen /app/proto/gen

WORKDIR /app/Frontend/src/order
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /order-frontend server.go

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
ENV FRONTEND_ADDR=:8083 \
    ORDER_SVC_ADDR=order:50053
EXPOSE 8083
COPY --from=build /order-frontend /usr/local/bin/order-frontend
ENTRYPOINT ["/usr/local/bin/order-frontend"]
