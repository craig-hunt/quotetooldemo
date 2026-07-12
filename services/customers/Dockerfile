# Multi-stage build: compile in a Go toolchain image, ship a static binary
# on a distroless base for a small, low-attack-surface runtime image.

FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/customers .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/customers /customers

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/customers"]
