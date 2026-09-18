FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \
        git tmux ca-certificates patch gcc libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY src/ /app/
RUN if [ ! -f go.mod ]; then go mod init host; fi
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
