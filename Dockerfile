FROM golang:1.15 as builder
WORKDIR /app
COPY . .
RUN apt update && apt install -y ca-certificates && make build

FROM scratch as release
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/bin/gh-webhook-monitor /bin/gh-webhook-monitor
EXPOSE 8080
ENTRYPOINT ["/bin/gh-webhook-monitor"]
