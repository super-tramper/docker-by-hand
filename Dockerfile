FROM golang:latest

RUN apt-get update \
    && apt-get -y install apt-utils \
    && apt-get -y install psmisc \
    && apt-get -y install net-tools