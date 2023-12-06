FROM gcr.io/distroless/static-debian11:latest
COPY assertoor* /assertoor
ENTRYPOINT ["/assertoor"]
