FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN templ generate && go build -ldflags "$(./scripts/version.sh ldflags)" -o /jumpgate ./cmd/jumpgate

FROM alpine:3

RUN adduser -D -u 1000 app

WORKDIR /app

COPY --from=builder /jumpgate .

RUN mkdir -p data && chown app:app data

USER app

VOLUME /app/data

EXPOSE 8080

ENTRYPOINT ["./jumpgate"]
CMD ["server"]
