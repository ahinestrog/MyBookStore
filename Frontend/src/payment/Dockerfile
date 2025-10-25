FROM golang:1.24 AS build
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY proto/gen ./proto/gen
COPY Frontend/src/payment/ ./Frontend/src/payment/

WORKDIR /app/Frontend/src/payment
RUN go build -o /bin/payment-frontend .

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot
COPY --from=build /bin/payment-frontend /bin/payment-frontend
COPY --from=build /app/Frontend/src/payment/templates /srv/templates
COPY --from=build /app/Frontend/src/payment/static    /srv/static
ENV HTTP_ADDR=:8084 \
    FRONTEND_PAYMENT_ADDR=:8084 \
    PAYMENT_GRPC_ADDR=payment:50053
EXPOSE 8084
ENTRYPOINT ["/bin/payment-frontend"]

