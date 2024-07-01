#!/bin/bash

#删除./bin目录
rm -rf ./bin/
mkdir ./bin

# 获取当前时间作为版本号
VERSION=$(date +"%Y%m%d%H%M")

# 设置目标操作系统和架构
TARGETS=(
    "linux/amd64"
    "windows/amd64"
    "darwin/amd64"
)

# 遍历每个目标并构建二进制文件
for target in "${TARGETS[@]}"
do
    # 获取操作系统和架构
    GOOS=$(echo $target | cut -d'/' -f1)
    GOARCH=$(echo $target | cut -d'/' -f2)

    NAME="emit_$GOOS-$GOARCH-$VERSION"

    if [ $GOOS = "windows" ]; then
        # 如果是 Windows 系统则添加 .exe 后缀
        NAME="$NAME.exe"
    fi

    # 构建二进制文件
    env GOOS=$GOOS GOARCH=$GOARCH go build -o ./bin/$NAME ./...
done