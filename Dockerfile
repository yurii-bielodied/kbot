FROM --platform=$BUILDPLATFORM quay.io/projectquay/golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY . .
RUN make build TARGETOS=$TARGETOS TARGETARCH=$TARGETARCH

FROM scratch
WORKDIR /
COPY --from=builder /app/kbot .
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["./kbot", "start"]
