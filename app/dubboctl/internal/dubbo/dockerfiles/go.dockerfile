# Please modify your template according to your business needs!!!

FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0
ENV GOPROXY https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

WORKDIR /build

ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .
COPY ./conf /app/conf
RUN go build -ldflags="-s -w" -trimpath -o /app/dubbogo ./cmd

FROM scratch

WORKDIR /app
COPY --from=builder /app/dubbogo /app/dubbogo
COPY --from=builder /app/conf /app/conf
ENV DUBBO_GO_CONFIG_PATH=/app/conf/dubbogo.yaml

CMD ["./dubbogo"]
