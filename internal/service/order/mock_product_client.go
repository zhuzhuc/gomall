package order

import (
	context "context"

	productpb "github.com/bytedance-youthcamp/demo/api/product"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockProductClient is a mock implementation of the productpb.ProductServiceClient interface
type MockProductClient struct {
	mock.Mock
}

// CreateProduct mocks the CreateProduct method
func (m *MockProductClient) CreateProduct(ctx context.Context, in *productpb.CreateProductRequest, opts ...grpc.CallOption) (*productpb.CreateProductResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*productpb.CreateProductResponse), args.Error(1)
}

// GetProduct mocks the GetProduct method
func (m *MockProductClient) GetProduct(ctx context.Context, in *productpb.GetProductRequest, opts ...grpc.CallOption) (*productpb.GetProductResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*productpb.GetProductResponse), args.Error(1)
}

// GetProducts mocks the GetProducts method
func (m *MockProductClient) GetProducts(ctx context.Context, in *productpb.GetProductsRequest, opts ...grpc.CallOption) (*productpb.GetProductsResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*productpb.GetProductsResponse), args.Error(1)
}

// UpdateProduct mocks the UpdateProduct method
func (m *MockProductClient) UpdateProduct(ctx context.Context, in *productpb.UpdateProductRequest, opts ...grpc.CallOption) (*productpb.UpdateProductResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*productpb.UpdateProductResponse), args.Error(1)
}

// DeleteProduct mocks the DeleteProduct method
func (m *MockProductClient) DeleteProduct(ctx context.Context, in *productpb.DeleteProductRequest, opts ...grpc.CallOption) (*productpb.DeleteProductResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*productpb.DeleteProductResponse), args.Error(1)
}

// We don't need ReduceStock as it's not in the ProductServiceClient interface
// Instead, we'll use UpdateProduct to handle stock reduction