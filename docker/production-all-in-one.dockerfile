# All-in-one production Dockerfile
# No host dependencies required (except Docker) - everything builds inside containers
#
# Usage:
#   docker build -f docker/production-all-in-one.dockerfile -t dify-sandbox .
#
# With custom tag:
#   docker build -f docker/production-all-in-one.dockerfile -t my-registry/dify-sandbox:v1.0.0 .

ARG PYTHON_VERSION=dhi.io/python:3-debian13-sfw-ent-dev
ARG GOLANG_VERSION=1.25.0
ARG DEBIAN_MIRROR="http://deb.debian.org/debian testing main"
ARG PYTHON_PACKAGES="httpx==0.27.2 requests==2.33.0 jinja2==3.1.6 PySocks httpx[socks]"
ARG NODEJS_VERSION=v20.20.0
ARG NODEJS_MIRROR="https://npmmirror.com/mirrors/node"
ARG TARGETARCH

# ============================================================
# Stage 1: Build all Go binaries inside a container
# ============================================================
FROM golang:${GOLANG_VERSION} AS builder

ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies (CGO requires gcc and libseccomp-dev)
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       -o Dpkg::Options::="--force-confdef" \
       -o Dpkg::Options::="--force-confold" \
       pkg-config gcc libseccomp-dev \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY . /app
WORKDIR /app

RUN go mod tidy

# Build Python shared library
RUN CGO_ENABLED=1 GOOS=linux go build \
    -o internal/core/runner/python/python.so \
    -buildmode=c-shared -ldflags="-s -w" \
    cmd/lib/python/main.go

# Build Node.js shared library
RUN CGO_ENABLED=1 GOOS=linux go build \
    -o internal/core/runner/nodejs/nodejs.so \
    -buildmode=c-shared -ldflags="-s -w" \
    cmd/lib/nodejs/main.go

# Build main server binary
RUN GOOS=linux go build \
    -o main -ldflags="-s -w" \
    cmd/server/main.go

# Build environment initialization binary
RUN GOOS=linux go build \
    -o env -ldflags="-s -w" \
    cmd/dependencies/init.go

# ============================================================
# Stage 2: Production image
# ============================================================
FROM ${PYTHON_VERSION}

ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
RUN echo "deb ${DEBIAN_MIRROR}" > /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
       -o Dpkg::Options::="--force-confdef" \
       -o Dpkg::Options::="--force-confold" \
       pkg-config \
       libseccomp-dev \
       wget \
       curl \
       xz-utils \
       zlib1g \
       expat \
       perl \
       libsqlite3-0 \
       passwd \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Copy compiled binaries from builder stage
COPY --from=builder /app/main /main
COPY --from=builder /app/env /env

# Copy configuration files
COPY conf/config.yaml /conf/config.yaml
COPY dependencies/python-requirements.txt /dependencies/python-requirements.txt
COPY docker/entrypoint.sh /entrypoint.sh

# Set permissions and install Python dependencies
RUN chmod +x /main /env /entrypoint.sh \
    && pip3 install --no-cache-dir ${PYTHON_PACKAGES}

# Download Node.js based on architecture and run environment initialization
RUN case "${TARGETARCH}" in \
    "amd64") \
        NODEJS_ARCH="linux-x64" ;; \
    "arm64") \
        NODEJS_ARCH="linux-arm64" ;; \
    *) \
        echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac \
    && wget -O /opt/node-${NODEJS_VERSION}-${NODEJS_ARCH}.tar.xz \
       ${NODEJS_MIRROR}/${NODEJS_VERSION}/node-${NODEJS_VERSION}-${NODEJS_ARCH}.tar.xz \
    && export NODE_TAR_XZ="/opt/node-${NODEJS_VERSION}-${NODEJS_ARCH}.tar.xz" \
    && export NODE_DIR="/opt/node-${NODEJS_VERSION}-${NODEJS_ARCH}" \
    && /env \
    && rm -f /env

# Set environment variables for entrypoint
ENV NODE_TAR_XZ=/opt/node-${NODEJS_VERSION}-linux-__ARCH__.tar.xz
ENV NODE_DIR=/opt/node-${NODEJS_VERSION}-linux-__ARCH__

ENTRYPOINT ["/entrypoint.sh"]
