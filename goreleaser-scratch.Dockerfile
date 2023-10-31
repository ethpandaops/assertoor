FROM gcr.io/distroless/static-debian11:latest
COPY sync-test-coordinator* /sync-test-coordinator
ENTRYPOINT ["/sync-test-coordinator"]
