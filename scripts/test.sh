#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # 无颜色

# 帮助信息
usage() {
    echo "使用方法: $0 [服务名]"
    echo "可用的服务:"
    echo "  auth     - 测试认证服务"
    echo "  user     - 测试用户服务"
    echo "  product  - 测试商品服务"
    echo "  cart     - 测试购物车服务"
    echo "  order    - 测试订单服务"
    echo "  payment  - 测试支付服务"
    echo "  all      - 测试所有服务"
}

# 测试指定服务
test_service() {
    local service=$1
    echo -e "${GREEN}正在测试 $service 服务...${NC}"
    go test -v ./internal/service/$service/...
    local result=$?
    if [ $result -eq 0 ]; then
        echo -e "${GREEN}$service 服务测试通过${NC}"
    else
        echo -e "${RED}$service 服务测试失败${NC}"
    fi
    return $result
}

# 主逻辑
main() {
    if [ $# -eq 0 ]; then
        usage
        exit 1
    fi

    local service=$1
    local total_failures=0

    case $service in
        auth)
            test_service auth
            total_failures=$?
            ;;
        user)
            test_service user
            total_failures=$?
            ;;
        product)
            test_service product
            total_failures=$?
            ;;
        cart)
            test_service cart
            total_failures=$?
            ;;
        order)
            test_service order
            total_failures=$?
            ;;
        payment)
            test_service payment
            total_failures=$?
            ;;
        all)
            services=("auth" "user" "product" "cart" "order" "payment")
            for srv in "${services[@]}"; do
                test_service $srv
                srv_result=$?
                total_failures=$((total_failures + srv_result))
            done
            ;;
        *)
            echo "未知的服务: $service"
            usage
            exit 1
            ;;
    esac

    if [ $total_failures -eq 0 ]; then
        echo -e "${GREEN}所有测试通过！${NC}"
        exit 0
    else
        echo -e "${RED}存在测试失败！${NC}"
        exit 1
    fi
}

# 执行主逻辑
main "$@"
