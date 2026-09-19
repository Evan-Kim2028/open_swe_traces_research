FROM golang:1.23
RUN apt-get update && apt-get install -y --no-install-recommends git tmux ca-certificates patch gcc libc6-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY src/ /app/
RUN find /app -name .git -type d -prune -exec rm -rf {} + || true
RUN GOPROXY=https://proxy.golang.org,direct go mod download || GOPROXY=https://proxy.golang.org,direct go mod tidy || true
RUN if [ -d integration_tests ]; then cd integration_tests && GOPROXY=https://proxy.golang.org,direct GOFLAGS=-mod=mod go build ./... || true; fi
