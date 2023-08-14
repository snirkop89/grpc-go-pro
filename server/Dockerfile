FROM golang:1.20-alpine

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o bin/server ./server

FROM alpine:3.18
WORKDIR /app
COPY ./certs/ ./certs/
COPY --from=0 /app/bin/server .
ENTRYPOINT [ "./server" ]