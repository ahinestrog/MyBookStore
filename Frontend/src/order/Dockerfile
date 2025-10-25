FROM golang:1.24 AS build

WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
COPY go.mod go.sum ./
RUN go mod download

COPY Frontend/src/order ./Frontend/src/order
COPY proto/gen ./proto/gen

WORKDIR /app/Frontend/src/order
RUN go build -o /bin/order-frontend .

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot
COPY --from=build /bin/order-frontend /bin/order-frontend
COPY --from=build /app/Frontend/src/order/templates /srv/templates
COPY --from=build /app/Frontend/src/order/static    /srv/static
ENV FRONTEND_ADDR=:8083 \
    ORDER_SVC_ADDR=order:50054
EXPOSE 8083
ENTRYPOINT ["/bin/order-frontend"]