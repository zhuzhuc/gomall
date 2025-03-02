package order

import (
	context "context"

	userpb "github.com/bytedance-youthcamp/demo/api/user"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockUserClient is a mock implementation of the userpb.UserServiceClient interface
type MockUserClient struct {
	mock.Mock
}

// Register mocks the Register method
func (m *MockUserClient) Register(ctx context.Context, in *userpb.RegisterRequest, opts ...grpc.CallOption) (*userpb.RegisterResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.RegisterResponse), args.Error(1)
}

// Login mocks the Login method
func (m *MockUserClient) Login(ctx context.Context, in *userpb.LoginRequest, opts ...grpc.CallOption) (*userpb.LoginResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.LoginResponse), args.Error(1)
}

// GetUserInfo mocks the GetUserInfo method
func (m *MockUserClient) GetUserInfo(ctx context.Context, in *userpb.GetUserInfoRequest, opts ...grpc.CallOption) (*userpb.GetUserInfoResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.GetUserInfoResponse), args.Error(1)
}

// UpdateUserInfo mocks the UpdateUserInfo method
func (m *MockUserClient) UpdateUserInfo(ctx context.Context, in *userpb.UpdateUserInfoRequest, opts ...grpc.CallOption) (*userpb.UpdateUserInfoResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.UpdateUserInfoResponse), args.Error(1)
}

// EnableTwoFactor mocks the EnableTwoFactor method
func (m *MockUserClient) EnableTwoFactor(ctx context.Context, in *userpb.EnableTwoFactorRequest, opts ...grpc.CallOption) (*userpb.EnableTwoFactorResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.EnableTwoFactorResponse), args.Error(1)
}

// VerifyTwoFactor mocks the VerifyTwoFactor method
func (m *MockUserClient) VerifyTwoFactor(ctx context.Context, in *userpb.VerifyTwoFactorRequest, opts ...grpc.CallOption) (*userpb.VerifyTwoFactorResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.VerifyTwoFactorResponse), args.Error(1)
}

// Logout mocks the Logout method
func (m *MockUserClient) Logout(ctx context.Context, in *userpb.LogoutRequest, opts ...grpc.CallOption) (*userpb.LogoutResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.LogoutResponse), args.Error(1)
}

// DeleteUser mocks the DeleteUser method
func (m *MockUserClient) DeleteUser(ctx context.Context, in *userpb.DeleteUserRequest, opts ...grpc.CallOption) (*userpb.DeleteUserResponse, error) {
	args := m.Called(ctx, in, opts)
	return args.Get(0).(*userpb.DeleteUserResponse), args.Error(1)
}