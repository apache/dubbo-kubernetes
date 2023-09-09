FROM golang:{{.Version}}alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0
{{if .Chinese}}ENV GOPROXY https://goproxy.cn,direct
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
{{end}}
WORKDIR /build

ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .
COPY ./conf /app/conf
RUN go build -ldflags="-s -w" -o /app/{{.ExeFile}} {{.GoMainFrom}}

FROM {{.BaseImage}}

WORKDIR /app
COPY --from=builder /app/{{.ExeFile}} /app/{{.ExeFile}}
COPY --from=builder /app/conf /app/conf
ENV DUBBO_GO_CONFIG_PATH=/app/conf/dubbogo.yaml
{{if .Port}}
EXPOSE {{.}}
{{end}}
CMD ["./{{.ExeFile}}"]
