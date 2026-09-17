FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/server ./cmd/server
RUN go build -o /bin/worker ./cmd/worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/server /bin/server
COPY --from=builder /bin/worker /bin/worker
