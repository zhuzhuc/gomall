#!/bin/bash

# 启动 Auth 服务
echo "启动 Auth 服务..."
cd /Users/Apple/Desktop/demo/cmd/auth
go run main.go &
AUTH_PID=$!
sleep 2

# 启动 User 服务
echo "启动 User 服务..."
cd /Users/Apple/Desktop/demo/cmd/user
go run main.go &
USER_PID=$!
sleep 2

# 启动 Product 服务
echo "启动 Product 服务..."
cd /Users/Apple/Desktop/demo/cmd/product
go run main.go &
PRODUCT_PID=$!
sleep 2

# 启动 Cart 服务
echo "启动 Cart 服务..."
cd /Users/Apple/Desktop/demo/cmd/cart
go run main.go &
CART_PID=$!
sleep 2

# 启动 Order 服务
echo "启动 Order 服务..."
cd /Users/Apple/Desktop/demo/cmd/order
go run main.go &
ORDER_PID=$!
sleep 2

# 启动 Payment 服务
echo "启动 Payment 服务..."
cd /Users/Apple/Desktop/demo/cmd/payment
go run main.go &
PAYMENT_PID=$!
sleep 2

echo "所有服务已启动"
echo "按 Ctrl+C 停止所有服务"

# 等待所有后台进程
wait $AUTH_PID $USER_PID $PRODUCT_PID $CART_PID $ORDER_PID $PAYMENT_PID
