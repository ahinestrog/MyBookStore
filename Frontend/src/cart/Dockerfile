FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Copiamos los protos generados y el front
COPY proto/ ./proto/
COPY Frontend/src/cart/ ./Frontend/src/cart/
WORKDIR /app/Frontend/src/cart
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cart-gateway

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
COPY --from=build /bin/cart-gateway /bin/cart-gateway
COPY Frontend/src/cart/templates ./templates
COPY Frontend/src/cart/static ./static
ENV GATEWAY_HTTP_ADDR=:8080
ENV CART_GRPC_TARGET=cart:50051
EXPOSE 8080
ENTRYPOINT ["/bin/cart-gateway"]
