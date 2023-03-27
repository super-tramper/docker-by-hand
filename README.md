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

5.1 实现容器的后台运行
我当前开发的环境内核版本为4.4.0，当主进程退出后，容器进程的PPID不会变为1，而是变成`init --user`的PID，这是从systemd 219 开始引入的新特性，它被用来管理用户级别的 systemd 服务。

6.5 将网桥地址转换成宿主机出口网卡的IP
```
sysctl -w net.ipv4.conf.all.forwarding=1
iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -o eth0 -j MASQUERADE
```