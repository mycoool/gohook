#!/bin/bash

# 构建项目
echo "🔨 构建项目..."
go build -o webhook-ui .

if [ $? -eq 0 ]; then
    echo "✅ 构建成功"
    echo ""
    
    # 启动服务
    echo "🌟 启动服务..."
    ./webhook-ui -debug
else
    echo "❌ 构建失败"
    exit 1
fi 