import grpc
import sys
sys.path.append('./api')  # 确保 protobuf 生成的文件在路径中

# 导入各服务的 Protobuf 模块
import auth_pb2
import auth_pb2_grpc
import user_pb2
import user_pb2_grpc
import product_pb2
import product_pb2_grpc
import cart_pb2
import cart_pb2_grpc
import order_pb2
import order_pb2_grpc
import payment_pb2
import payment_pb2_grpc

def test_services():
    services = [
        {"name": "Auth", "port": 50050, "pb2": auth_pb2, "pb2_grpc": auth_pb2_grpc},
        {"name": "User", "port": 50051, "pb2": user_pb2, "pb2_grpc": user_pb2_grpc},
        {"name": "Product", "port": 50052, "pb2": product_pb2, "pb2_grpc": product_pb2_grpc},
        {"name": "Cart", "port": 50055, "pb2": cart_pb2, "pb2_grpc": cart_pb2_grpc},
        {"name": "Order", "port": 50053, "pb2": order_pb2, "pb2_grpc": order_pb2_grpc},
        {"name": "Payment", "port": 50054, "pb2": payment_pb2, "pb2_grpc": payment_pb2_grpc}
    ]

    for service in services:
        try:
            channel = grpc.insecure_channel(f'localhost:{service["port"]}')
            print(f"测试 {service['name']} 服务...")
            
            # 根据不同服务进行特定测试
            if service['name'] == 'Auth':
                stub = service['pb2_grpc'].AuthServiceStub(channel)
                # 简单的令牌验证测试
                verify_req = service['pb2'].VerifyTokenReq(token="test_token")
                response = stub.VerifyTokenByRPC(verify_req)
                print(f"{service['name']} 服务测试通过")
            
            elif service['name'] == 'User':
                stub = service['pb2_grpc'].UserServiceStub(channel)
                # 获取用户信息测试
                user_req = service['pb2'].GetUserInfoRequest(user_id=1)
                response = stub.GetUserInfo(user_req)
                print(f"{service['name']} 服务测试通过")
            
            elif service['name'] == 'Product':
                stub = service['pb2_grpc'].ProductServiceStub(channel)
                # 获取产品列表测试
                product_req = service['pb2'].GetProductsRequest(page=1, page_size=10)
                response = stub.GetProducts(product_req)
                print(f"{service['name']} 服务测试通过")
            
            elif service['name'] == 'Cart':
                stub = service['pb2_grpc'].CartServiceStub(channel)
                # 获取购物车测试
                cart_req = service['pb2'].GetCartRequest(user_id=1)
                response = stub.GetCart(cart_req)
                print(f"{service['name']} 服务测试通过")
            
            elif service['name'] == 'Order':
                stub = service['pb2_grpc'].OrderServiceStub(channel)
                # 获取用户订单测试
                order_req = service['pb2'].GetUserOrdersRequest(user_id=1)
                response = stub.GetUserOrders(order_req)
                print(f"{service['name']} 服务测试通过")
            
            elif service['name'] == 'Payment':
                stub = service['pb2_grpc'].PaymentServiceStub(channel)
                # 查询支付状态测试
                payment_req = service['pb2'].QueryPaymentRequest(order_id="test_order")
                response = stub.QueryPayment(payment_req)
                print(f"{service['name']} 服务测试通过")

        except Exception as e:
            print(f"{service['name']} 服务测试失败: {e}")

if __name__ == '__main__':
    test_services()
