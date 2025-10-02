FROM gcr.io/distroless/static:debug AS discovery
COPY sail-discovery .
ENTRYPOINT ["./sail-discovery"]

FROM gcr.io/distroless/static:debug AS agent
COPY sail-agent .
ENTRYPOINT ["./sail-agent"]
