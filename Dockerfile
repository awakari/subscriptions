FROM golang:1.24.5-alpine3.22 AS builder
WORKDIR /go/src/subscriptions
COPY . .
RUN \
    apk add protoc protobuf-dev make git && \
    make build

FROM scratch
COPY --from=builder /go/src/subscriptions/subscriptions /bin/subscriptions
ENTRYPOINT ["/bin/subscriptions"]
