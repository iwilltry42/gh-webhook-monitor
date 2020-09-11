FROM golang:1.15 as builder
WORKDIR /app
COPY . .
RUN make build

FROM scratch as release
COPY --from=builder /app/bin/gh-webhook-monitor /bin/gh-webhook-monitor
EXPOSE 8080
ENTRYPOINT ["/bin/gh-webhook-monitor"]
