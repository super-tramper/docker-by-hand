FROM golang:latest

RUN apt-get update \
    && apt-get -y install apt-utils \
    && apt-get -y install psmisc \
    && apt-get -y install stress \
    && apt-get -y install net-tools

ENV GOPROXY https://proxy.golang.com.cn,direct
