FROM golang:1.24 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY proto/gen ./proto/gen
COPY Frontend/src/cart/ ./Frontend/src/cart/
WORKDIR /app/Frontend/src/cart
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cart-gateway

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot

COPY --from=build /bin/cart-gateway /bin/cart-gateway
COPY --from=build /app/Frontend/src/cart/templates /srv/templates
COPY --from=build /app/Frontend/src/cart/static    /srv/static
ENV GATEWAY_HTTP_ADDR=:8085 \
    CART_GRPC_TARGET=cart:50050 \
    ORDER_GRPC_TARGET=order:50054 \
    INVENTORY_GRPC_TARGET=inventory:50052
EXPOSE 8085
ENTRYPOINT ["/bin/cart-gateway"]
