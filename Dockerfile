FROM golang:1.26-alpine AS builder

RUN apk add --no-cache curl

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

# Install Tailwind CSS standalone CLI (supports both amd64 and arm64)
ARG TARGETARCH
RUN TWARCH=$([ "$TARGETARCH" = "amd64" ] && echo "x64" || echo "$TARGETARCH") \
    && curl -sfLo /usr/local/bin/tailwindcss \
       "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-${TWARCH}-musl" \
    && chmod +x /usr/local/bin/tailwindcss

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ files
RUN templ generate

# Build CSS
RUN tailwindcss -i static/css/input.css -o static/css/output.css --minify

# Build binary
RUN CGO_ENABLED=0 go build -o /app/bin/server ./cmd/server

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/static ./static

RUN mkdir -p /data/uploads

ENV DATABASE_PATH=/data/libreregistration.db
ENV UPLOAD_DIR=/data/uploads
ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["./server"]
