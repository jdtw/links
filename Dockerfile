FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o /app ./...

FROM gcr.io/distroless/static:latest
WORKDIR /app
COPY --from=builder /app/links /app/links
# Shipped so the Postgres -> SQLite migration can be run on a machine that can
# reach both the database and the volume.
COPY --from=builder /app/migrate /app/migrate

EXPOSE 8080
CMD ["/app/links"]
