FROM golang:1.20-alpine AS builder

ENV GOOS=linux
ENV GOARCH=amd64
ENV GO11MODULE=on

WORKDIR /app

RUN apk add make

COPY go.mod go.mod
COPY go.sum go.sum

COPY Makefile Makefile

RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/
COPY main.go main.go

RUN make build

# Package into distroless
FROM gcr.io/distroless/static:nonroot

WORKDIR /

COPY --from=builder /app/out/linux/amd64/traefik-ingressroute-exporter .

USER nonroot:nonroot

ENTRYPOINT ["/traefik-ingressroute-exporter"]

CMD ["start"]
