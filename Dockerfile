FROM golang:1.23-alpine

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev && \
    go install github.com/rakyll/hey@v0.1.4 && \
    go install github.com/bojand/ghz/cmd/ghz@v0.117.0

# Keep container running for development
CMD ["tail", "-f", "/dev/null"]
