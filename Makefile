export PATH := $(HOME)/go/bin:$(PATH)

TAILWINDCSS := ./bin/tailwindcss

.PHONY: build run dev clean generate css lint lint-go lint-js

# Build the application
build: generate css
	go build -o bin/server ./cmd/server

# Run the server
run:
	./bin/server

# Development mode: generate, build, and run
dev: build run

# Generate templ files
generate:
	templ generate

# Build CSS with Tailwind
css:
	$(TAILWINDCSS) -i static/css/input.css -o static/css/output.css --minify

# Watch CSS for development
css-watch:
	$(TAILWINDCSS) -i static/css/input.css -o static/css/output.css --watch

# Lint all code
lint: lint-go lint-js

# Lint Go code
lint-go:
	golangci-lint run ./...

# Lint JavaScript (no-op if no JS files exist yet)
lint-js:
	@ls static/**/*.js >/dev/null 2>&1 && npx eslint 'static/**/*.js' || echo "No JS files to lint"

# Clean build artifacts
clean:
	rm -rf bin/server
	rm -f static/css/output.css
	find . -name '*_templ.go' -delete
