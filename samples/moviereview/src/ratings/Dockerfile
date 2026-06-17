FROM golang:1.25-alpine AS build

WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -o /out/ratings main.go

FROM alpine:3.22
COPY --from=build /out/ratings /usr/local/bin/ratings

EXPOSE 9080
CMD ["ratings"]
