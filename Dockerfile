FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /bin/telegram-mock-ai ./cmd/telegram-mock-ai

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/telegram-mock-ai /usr/local/bin/telegram-mock-ai

EXPOSE 8081 8082

ENTRYPOINT ["/usr/local/bin/telegram-mock-ai"]
CMD ["-config", "/etc/telegram-mock-ai/config.yaml"]
