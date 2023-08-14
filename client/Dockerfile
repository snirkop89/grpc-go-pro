FROM golang:1.21-alpine

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o bin/client ./client

FROM alpine:3.18
WORKDIR /app
COPY ./certs/ ./certs/
COPY --from=0 /app/bin/client .
ENTRYPOINT [ "./client" ]