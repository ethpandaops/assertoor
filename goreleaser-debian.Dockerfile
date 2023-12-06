FROM debian:latest
RUN apt-get update && apt-get -y upgrade
RUN apt-get install -y --no-install-recommends \
  libssl-dev \
  procps \
  ca-certificates \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/*
COPY assertoor* /assertoor
ENTRYPOINT ["/assertoor"]
