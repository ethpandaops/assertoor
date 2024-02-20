# syntax=docker/dockerfile:1
FROM golang:1.21 AS builder
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
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
COPY --from=builder bin/assertoor /assertoor
ENTRYPOINT ["/assertoor"]