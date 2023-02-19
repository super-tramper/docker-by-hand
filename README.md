# docker-by-hand
手写docker
开发过程中使用docker环境调试，命令如下：
```
进入容器shell
docker run -it --rm --privileged -v /project/path:/go/src -w /go/src golang /bin/bash
直接运行go run main.go
docker run -it --rm --privileged -v /project/path:/go/src -w /go/src golang /bin/bash -c "go run main.go"
```