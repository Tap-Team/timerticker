FROM golang:1.20-alpine3.17 as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/ticker /app/cmd/main/main.go

FROM alpine:3.17
COPY --from=builder /app/config /config
COPY --from=builder /app/ticker /ticker
EXPOSE 50001
ENTRYPOINT [ "/ticker" ]