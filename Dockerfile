FROM golang:latest

ENV GOPROXY https://proxy.golang.com.cn,direct
RUN apt-get update \
    && apt-get -y install apt-utils \
    && apt-get -y install psmisc \
    && apt-get -y install stress \
    && apt-get -y install net-tools

ADD ./busybox.tar /root/busybox
ADD ./ /go/src/docker/
WORKDIR /go/src/docker
RUN go get
