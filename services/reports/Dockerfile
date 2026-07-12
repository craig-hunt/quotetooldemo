FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/reports .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/reports /reports
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/reports"]
