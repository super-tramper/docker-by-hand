# docker-by-hand
手写docker
开发过程中使用docker环境调试，命令如下：
```
# 构建docker镜像
docker build -t golang:lab

# 进入容器shell
docker run -it --rm --privileged --name golab -v /project/path:/go/src -w /go/src golang:lab /bin/bash

# 直接运行go run main.go
docker run -it --rm --privileged --name golab -v /project/path:/go/src -w /go/src golang:lab go run main.go

# 限制容器的内存
docker run -m 1G --memory-swap 1G -it --name golab --rm --privileged -v /project/path:/go/src -w /go/src golang:lab
go run main.go run -ti -m 100m stress --vm 1 --vm-bytes 200m --vm-keep

# cpu时间片分配比例
nohup stress --vm-bytes 200m --vm-keep --vm 1 &
go run main.go run -ti -cpushare 512 stress --vm-bytes 200m --vm-keep --vm 1
```
