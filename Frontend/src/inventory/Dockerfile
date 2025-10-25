FROM golang:1.24 AS build
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
COPY go.mod go.sum ./
RUN go mod download

COPY proto/gen ./proto/gen
COPY Frontend/src/inventory/ ./Frontend/src/inventory/

WORKDIR /app/Frontend/src/inventory
RUN go build -o /bin/inventory-frontend .

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot
COPY --from=build /bin/inventory-frontend /bin/inventory-frontend
COPY --from=build /app/Frontend/src/inventory/templates /srv/templates
COPY --from=build /app/Frontend/src/inventory/static    /srv/static

ENV FRONTEND_INVENTORY_ADDR=:8082 \
    INVENTORY_GRPC_ADDR=inventory:50052
EXPOSE 8082
ENTRYPOINT ["/bin/inventory-frontend"]
