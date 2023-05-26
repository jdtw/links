FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./...

FROM gcr.io/distroless/static:latest
WORKDIR /app
COPY --from=builder /app/links /app/links
COPY --from=builder /app/client /app/client

WORKDIR /keys
COPY ./dev/priv.pb priv.pb

CMD ["/app/links"]
