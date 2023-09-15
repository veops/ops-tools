FROM golang:1.20.4-alpine3.17
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add tzdata
ENV TZ=Asia/Shanghai
WORKDIR /messenger
COPY . .
RUN go env -w GOPROXY=https://goproxy.cn,direct \
    && go build -o ./messenger ./main.go

FROM alpine:latest
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add tzdata
ENV TZ=Asia/Shanghai
WORKDIR /messenger
COPY --from=0 /messenger/conf.yaml .
COPY --from=0 /messenger/messenger .
CMD [ "./messenger"]
