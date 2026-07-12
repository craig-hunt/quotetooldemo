FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/invoices .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/invoices /invoices
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/invoices"]
