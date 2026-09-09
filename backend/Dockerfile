# Build stage
FROM golang:1.26.4-alpine3.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /app/bin/worker ./cmd/worker

# Runtime stage — pinned base image tag (immutable release artifacts reference this tag).
FROM alpine:3.21.3

WORKDIR /app

RUN apk --no-cache add ca-certificates curl \
	&& addgroup -S -g 65532 synaudio \
	&& adduser -S -u 65532 -G synaudio synaudio \
	&& chown -R synaudio:synaudio /app

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker

USER 65532:65532

EXPOSE 8080

CMD ["/app/api"]
