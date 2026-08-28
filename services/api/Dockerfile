FROM golang:1.23-alpine

WORKDIR /app

COPY server /app/server
RUN chmod +x /app/server

EXPOSE 8080

ENTRYPOINT ["/app/server"]