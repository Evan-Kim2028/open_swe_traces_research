FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \
        git tmux ca-certificates patch gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

ENV GOTOOLCHAIN=local
WORKDIR /app
COPY src/ /app/
RUN find /app -name .git -type d -prune -exec rm -rf {} + || true
RUN if [ ! -f go.mod ]; then go mod init host; fi
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \
RUN cd integration_tests && GOPROXY=https://proxy.golang.org,direct go mod download || true
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
WORKDIR /app/integration_tests
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \
RUN cd integration_tests && GOPROXY=https://proxy.golang.org,direct go mod download || true
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
WORKDIR /app
