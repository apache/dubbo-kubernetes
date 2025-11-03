FROM gcr.io/distroless/static:debug AS discovery
COPY sail-discovery /usr/local/bin/sail-discovery
ENTRYPOINT ["/usr/local/bin/sail-discovery"]

FROM gcr.io/distroless/static:debug AS proxy
COPY sail-agent /usr/local/bin/sail-agent
ENTRYPOINT ["/usr/local/bin/sail-agent"]