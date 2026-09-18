FROM golang:1.23

RUN apt-get update && apt-get install -y --no-install-recommends \
        git tmux ca-certificates patch \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV GOTOOLCHAIN=auto
COPY src/go.mod src/go.sum /app/
RUN GOPROXY=https://proxy.golang.org,direct go mod download
COPY src/ /app/
RUN GOPROXY=https://proxy.golang.org,direct go mod download || \
    GOPROXY=https://proxy.golang.org,direct go mod tidy || true
RUN go version
