FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/processor ./cmd/processor/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/simulator ./cmd/simulator/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/dashboard ./cmd/dashboard/main.go

FROM alpine:3.21 AS final
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
RUN adduser -D appuser
USER appuser

FROM final AS processor
COPY --from=builder /bin/processor .
CMD ["./processor"]

FROM final AS simulator
COPY --from=builder /bin/simulator .
CMD ["./simulator"]

FROM final AS dashboard
COPY --from=builder /bin/dashboard .
COPY --from=builder /src/frontend ./frontend
EXPOSE 8080
CMD ["./dashboard"]