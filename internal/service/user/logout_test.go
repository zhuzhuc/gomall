package user

import (
	"context"
	"testing"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogout(t *testing.T) {
	userService := setupUserService(t)
	username, password, _ := registerTestUser(t, userService)

	// First, login to get a token
	loginResp, err := userService.Login(context.Background(), &userapi.LoginRequest{
		Username: username,
		Password: password,
	})
	require.NoError(t, err)
	assert.True(t, loginResp.Success)
	assert.NotEmpty(t, loginResp.Token)

	// Test successful logout
	logoutResp, err := userService.Logout(context.Background(), &userapi.LogoutRequest{
		Token: loginResp.Token,
	})
	require.NoError(t, err)
	assert.True(t, logoutResp.Success)

	// Test logout with empty token
	logoutResp, err = userService.Logout(context.Background(), &userapi.LogoutRequest{
		Token: "",
	})
	require.NoError(t, err)
	assert.False(t, logoutResp.Success)
	assert.Equal(t, "token is required", logoutResp.ErrorMessage)

	// Test logout with invalid token
	logoutResp, err = userService.Logout(context.Background(), &userapi.LogoutRequest{
		Token: "invalid_token",
	})
	require.NoError(t, err)
	assert.False(t, logoutResp.Success)
	assert.Equal(t, "invalid token", logoutResp.ErrorMessage)
}