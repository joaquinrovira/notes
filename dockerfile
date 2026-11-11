FROM golang:1.27-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o app-bin ./cmd/server

FROM alpine:3

RUN adduser -h / -s /bin/sh -D app

USER app
WORKDIR /app

RUN mkdir -p /app/data /app/static

# Copy executable binary
COPY --from=build --chown=app:app /app/app-bin .

ENTRYPOINT ["/app/app-bin"]