FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
COPY vendor ./vendor
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /out/hnu-api ./cmd/api
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/hnu-api /hnu-api
EXPOSE 8080
ENTRYPOINT ["/hnu-api"]
