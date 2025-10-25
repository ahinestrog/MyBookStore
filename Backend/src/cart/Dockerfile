FROM golang:1.24 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY proto/gen ./proto/gen
COPY Backend/src/cart/ ./Backend/src/cart/
WORKDIR /app/Backend/src/cart/src
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cart

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
COPY --from=build /bin/cart /bin/cart
COPY Backend/src/cart/src/sql ./sql
EXPOSE 50050
# Usuario no root
USER nonroot:nonroot
ENTRYPOINT ["/bin/cart"]
