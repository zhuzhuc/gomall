#!/bin/bash

# 启动 MySQL Docker 容器
docker run -d --name mysql-test \
    -e MYSQL_ROOT_PASSWORD=yourpassword \
    -e MYSQL_DATABASE=yourdb \
    -p 3306:3306 \
    mysql:8.0

# 等待 MySQL 启动
sleep 30

# 启动 gRPC 服务
cd /Users/Apple/Desktop/demo
go run cmd/server/main.go &

# 等待服务启动
sleep 10
