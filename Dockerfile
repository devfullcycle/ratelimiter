FROM golang:1.23-alpine

WORKDIR /app

RUN apk add --no-cache git && \
    go install github.com/rakyll/hey@latest

# Keep container running for development
CMD ["tail", "-f", "/dev/null"]
