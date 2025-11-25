FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o app-bin .

FROM alpine

RUN adduser -h / -s /bin/sh -D app

USER app
WORKDIR /app

RUN mkdir -p /app/routes /app/static

# Copy executable binary
COPY --from=build --chown=app:app /app/app-bin .

ENTRYPOINT ["/app/app-bin"]