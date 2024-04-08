FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY . .
RUN  mkdir /app && mkdir /app/logs
RUN go build -o /app/my_fund ./cmd/app/

FROM alpine:latest
COPY --from=builder /app /app
COPY --from=builder /build/migrations /app/migrations
COPY --from=builder /build/startup.sh /app
RUN chmod +x /app/startup.sh
CMD ["/app/startup.sh"]