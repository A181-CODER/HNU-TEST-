FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY vendor ./vendor
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /out/hnu-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /out/hnu-healthcheck ./cmd/healthcheck
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/hnu-api /hnu-api
COPY --from=build /out/hnu-healthcheck /hnu-healthcheck
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=5 CMD ["/hnu-healthcheck"]
ENTRYPOINT ["/hnu-api"]
