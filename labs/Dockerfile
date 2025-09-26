FROM golang:1.25-alpine

# Build arguments
ARG BUILD_VERSION=0.0.1
ARG BUILD_TIME

WORKDIR /app

COPY . .

RUN go build -ldflags "-X 'd7024e/pkg/build.BuildTime=${BUILD_TIME}' -X 'd7024e/pkg/build.BuildVersion=${BUILD_VERSION}'" -o main ./cmd/

CMD ["/app/main", "start"]
