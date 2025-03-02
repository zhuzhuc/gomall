# ByteDance YouthCamp Demo 项目

## 项目概述
这是一个微服务架构的示例项目，展示了现代 Go 语言微服务开发的最佳实践。

## 技术栈
- Go 1.22+
- gRPC
- PostgreSQL
- Docker
- Prometheus
- Grafana
- Etcd
- JWT

## 服务列表
- 用户服务
- 认证服务
- 订单服务（待开发）

## 快速开始

### 先决条件
- Go 1.22+
- Docker
- Docker Compose

### 安装依赖
```bash
make deps
```

### 生成 Protobuf
```bash
make proto
```

### 运行测试
```bash
# 运行所有服务的测试
./scripts/test.sh all

# 测试特定服务
./scripts/test.sh auth     # 测试认证服务
./scripts/test.sh user     # 测试用户服务
./scripts/test.sh product  # 测试商品服务
./scripts/test.sh cart     # 测试购物车服务
./scripts/test.sh order    # 测试订单服务
./scripts/test.sh payment  # 测试支付服务
```

### 测试脚本特性
- 支持测试单个服务或所有服务
- 彩色输出，方便识别测试结果
- 详细的测试输出
- 错误时返回非零状态码

### 启动服务
```bash
make docker-up
```

## 监控
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000

## 配置
服务配置位于 `configs/` 目录下的 YAML 文件中。

## 安全特性
- JWT 认证
- 双因素认证
- 密码加密
- 请求速率限制

## 性能监控
使用 Prometheus 和 Grafana 进行实时性能监控。


