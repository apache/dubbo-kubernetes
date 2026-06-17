FROM golang:1.25-alpine AS build

WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -o /out/details main.go

FROM alpine:3.22
COPY --from=build /out/details /usr/local/bin/details

EXPOSE 9080
CMD ["details"]
