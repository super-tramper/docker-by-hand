FROM golang:latest

ENV GOPROXY https://proxy.golang.com.cn,direct
RUN apt-get update \
    && apt-get -y install apt-utils psmisc stress net-tools kmod \
    && apt -y install linux-headers-4.19.0-9-686-pae \
    && apt -y install aufs-tools \
    && apt -y install aufs-dkms \
    && apt -y install aufs-dev

#ADD ./busybox.tar /root/busybox
ADD ./ /go/src/docker/
WORKDIR /go/src/docker
RUN go get
