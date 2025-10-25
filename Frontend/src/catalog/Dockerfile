FROM golang:1.24 AS build
WORKDIR /app
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY proto/gen ./proto/gen
COPY Frontend/src/catalog/ ./Frontend/src/catalog/
WORKDIR /app/Frontend/src/catalog
RUN go build -o /bin/catalog-frontend .


FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /srv
USER nonroot:nonroot
COPY --from=build /bin/catalog-frontend /bin/catalog-frontend
COPY --from=build /app/Frontend/src/catalog/templates /srv/templates
COPY --from=build /app/Frontend/src/catalog/static    /srv/static

ENV FRONTEND_CATALOG_ADDR=:8081 \
    CATALOG_GRPC_ADDR=catalog:50051
EXPOSE 8081
ENTRYPOINT ["/bin/catalog-frontend"]
