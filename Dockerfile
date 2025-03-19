# syntax=docker/dockerfile:1
FROM golang:1.23 AS builder
WORKDIR /src
COPY go.sum go.mod ./
RUN go mod download
COPY . .
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

