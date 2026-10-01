# Stage 1: Build binary
FROM golang:alpine AS builder

WORKDIR /app

# Enable GOTOOLCHAIN=auto so Go can fetch its toolchain if needed
ENV GOTOOLCHAIN=auto
ENV CGO_ENABLED=0
ENV GOOS=linux

RUN apk add --no-cache ca-certificates tzdata git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-w -s" -o /app/bin/server ./cmd/server

# Stage 2: Minimal runner
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

CMD ["/app/server"]
