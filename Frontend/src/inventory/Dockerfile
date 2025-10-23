FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o inventory-frontend .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/inventory-frontend .
COPY static ./static
COPY templates ./templates
EXPOSE 8082
ENTRYPOINT ["./inventory-frontend"]
