FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/telegram-mock-ai ./cmd/telegram-mock-ai

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 app

COPY --from=builder /bin/telegram-mock-ai /usr/local/bin/telegram-mock-ai

USER app
WORKDIR /home/app

EXPOSE 8081 8082

ENTRYPOINT ["telegram-mock-ai"]
CMD ["-config", "/etc/telegram-mock-ai/config.yaml"]
