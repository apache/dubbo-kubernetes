FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /agentmock main.go

FROM scratch
COPY --from=builder /agentmock /agentmock
USER 65532:65532
ENTRYPOINT ["/agentmock"]
