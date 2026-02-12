# syntax=docker/dockerfile:1
FROM node:20-slim AS ui-builder
WORKDIR /src
COPY web-ui/package.json web-ui/package-lock.json ./web-ui/
RUN cd web-ui && npm ci
COPY web-ui/ ./web-ui/
COPY pkg/web/static/embed.go ./pkg/web/static/
RUN cd web-ui && npm run build

FROM golang:1.25 AS builder
WORKDIR /src
COPY go.sum go.mod ./
RUN go mod download
COPY . .
COPY --from=ui-builder /src/pkg/web/static/ ./pkg/web/static/
RUN make build

FROM ubuntu:latest
RUN apt-get update && apt-get -y upgrade && apt-get install -y --no-install-recommends \
  libssl-dev \
  ca-certificates \
  jq \
  git \
  curl \
  make \
  sudo \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/* \
  && update-ca-certificates
RUN groupadd -g 10001 assertoor && useradd -m -u 10001 -g assertoor assertoor
RUN echo "assertoor ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers.d/assertoor
WORKDIR /app
COPY --from=builder /src/bin/* /app/
RUN chown -R assertoor:assertoor /app
USER assertoor
ENTRYPOINT ["/app/assertoor"]
