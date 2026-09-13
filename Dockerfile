FROM golang:1.25-alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN apk add --no-cache git
RUN apk add --no-cache ca-certificates
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o p1toinfluxdb .
FROM alpine:latest
COPY --from=builder /build/p1toinfluxdb /app/
COPY --from=builder /build/start.sh /app/
COPY --from=builder /etc/ssl/certs /etc/ssl/certs
WORKDIR /app
CMD [ "./start.sh" ]
