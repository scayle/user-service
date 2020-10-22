FROM golang:1.15-alpine AS build

ENV GO111MODULE=on

RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
    git

RUN mkdir /app
COPY . /app
WORKDIR /app

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o user-service

FROM alpine:latest

RUN apk update && \
    apk upgrade && \
    apk add --no-cache \
    ca-certificates

WORKDIR /app

COPY --from=build /app .

CMD ["./user-service"]