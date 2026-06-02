FROM golang:1.26-alpine AS builder
# ca-certificates is required for the final scratch image to validate
# TLS chains in the probes — Alpine doesn't include it by default.
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o cactus .

FROM scratch
# Without these, Go's crypto/tls returns "x509: certificate signed by
# unknown authority" for every HTTPS probe — even valid Let's Encrypt
# certs. The cert bundle is the only thing this image needs from the
# host's /etc/ssl/.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /src/cactus /cactus
ENTRYPOINT ["/cactus"]
