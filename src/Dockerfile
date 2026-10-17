# ---- Build stage: compile a static binary ----
FROM golang:1.27 AS build
WORKDIR /src

# Copy the module files first so this layer is cached until dependencies change.
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
# CGO off gives a static binary that runs on a minimal image; -s -w strips debug info.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /product-api .

# ---- Run stage: small image with just the binary ----
FROM --platform=linux/amd64 alpine:3.22
# Don't run the server as root.
RUN adduser -D -u 10001 app
USER app
COPY --from=build /product-api /product-api

ENV GIN_MODE=release

# The server listens on :8080 (see main in product-service.go).
EXPOSE 8080
ENTRYPOINT ["/product-api"]
