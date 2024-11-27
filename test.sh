#!/bin/bash

# 要检查的容器名称
container_name="redis"

# 检查容器是否存在且正在运行
if docker ps --filter "name=^${container_name}$" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    echo "running already"
    exit 1
fi

# 检查容器是否存在且处于停止状态
if docker ps -a --filter "name=^${container_name}$" --filter "status=exited" --format '{{.Names}}' | grep -q "^${container_name}$"; then
    docker start "${container_name}"
else
    echo "container not exist"
fi