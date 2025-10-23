FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache build-base

# Dependencias 
COPY go.mod go.sum ./
RUN go mod download

# Copiamos todo el repo para que el import de proto/gen/* funcione
COPY . .

# Compilamos el servidor
WORKDIR /app/Frontend/src/payment
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/payment-frontend server.go

FROM alpine:3.20
WORKDIR /srv
ENV FRONTEND_PAYMENT_PORT=8084 \
    PAYMENT_GRPC_ADDR=payment:50053
EXPOSE 8084
# Binario
COPY --from=build /bin/payment-frontend /usr/local/bin/payment-frontend
# Templates y estáticos (desde esta carpeta)
COPY Frontend/src/payment/templates /srv/templates
COPY Frontend/src/payment/static /srv/static
ENTRYPOINT ["/usr/local/bin/payment-frontend"]

