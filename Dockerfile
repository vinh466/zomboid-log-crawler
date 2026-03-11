FROM golang:1.26.1 AS builder
WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/log-crawler ./main.go

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/log-crawler /app/log-crawler
COPY config.yaml.example /app/config.yaml

EXPOSE 8080
ENV CONFIG_PATH=/app/config.yaml
ENTRYPOINT ["/app/log-crawler"]
