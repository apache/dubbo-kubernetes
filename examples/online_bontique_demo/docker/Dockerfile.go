FROM node:22-bookworm-slim AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY src/ad/ ./src/ad/
COPY src/checkout/ ./src/checkout/
COPY src/frontend/ ./src/frontend/
COPY src/recommendation/ ./src/recommendation/
COPY api/money_vectors.json ./api/money_vectors.json
ARG SERVICE
RUN go test ./src/${SERVICE} && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/service ./src/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
ENV PORT=8080
COPY --from=build /out/service /app/service
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/service"]

FROM runtime AS frontend
ENV STATIC_DIR=/app/web
COPY --from=web /web/dist/ /app/web/
