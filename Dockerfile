FROM golang:1.26-alpine AS builder

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source (includes pre-built CSS and templ files when built via CI)
COPY . .

# Generate templ files (idempotent if already generated)
RUN templ generate

# Build CSS: skip download if output.css already exists in context (CI pre-builds it)
ARG TARGETARCH
RUN if [ ! -s static/css/output.css ]; then \
      apk add --no-cache curl \
      && TWARCH=$([ "$TARGETARCH" = "amd64" ] && echo "x64" || echo "$TARGETARCH") \
      && curl -fsSL --retry 3 -o /usr/local/bin/tailwindcss \
         "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-${TWARCH}-musl" \
      && chmod +x /usr/local/bin/tailwindcss \
      && tailwindcss -i static/css/input.css -o static/css/output.css --minify; \
    fi

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
