FROM gcr.io/distroless/static-debian11:latest
COPY minccino* /minccino
ENTRYPOINT ["/minccino"]
