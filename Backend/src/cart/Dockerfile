FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY proto/ ./proto/
COPY Backend/src/cart/ ./Backend/src/cart/
WORKDIR /app/Backend/src/cart
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cart

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
COPY --from=build /bin/cart /bin/cart
COPY Backend/src/cart/sql ./sql
EXPOSE 50051
ENTRYPOINT ["/bin/cart"]
