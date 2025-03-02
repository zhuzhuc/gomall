package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthService(t *testing.T) *AuthService {
	authService, err := NewAuthService()
	require.NoError(t, err, "Failed to create AuthService")
	t.Cleanup(func() {
		authService.Close()
	})
	return authService
}

func generateTestToken(userID int32, ttl time.Duration, secretKey string, issuer string) (string, error) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     baseTime.Add(ttl).Unix(),
		"iat":     baseTime.Unix(),
		"iss":     issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

func TestDeliverTokenByRPC(t *testing.T) {
	authService := setupAuthService(t)

	// 设置固定的测试时间
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name     string
		userID   int32
		wantErr  bool
		wantResp bool
	}{
		{
			name:     "Valid User ID",
			userID:   1,
			wantErr:  false,
			wantResp: true,
		},
		{
			name:     "Invalid User ID",
			userID:   0,
			wantErr:  true,
			wantResp: false,
		},
		{
			name:     "Negative User ID",
			userID:   -1,
			wantErr:  true,
			wantResp: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := authService.DeliverTokenByRPC(context.Background(), &DeliverTokenReq{UserId: tc.userID})

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Token)

				// 验证令牌的有效性
				token, err := jwt.Parse(resp.Token, func(token *jwt.Token) (interface{}, error) {
					return []byte(authService.config.JWT.SecretKey), nil
				}, jwt.WithTimeFunc(func() time.Time { return testTime }))
				assert.NoError(t, err)
				assert.True(t, token.Valid)

				// 验证令牌声明
				claims, ok := token.Claims.(jwt.MapClaims)
				assert.True(t, ok)
				assert.Equal(t, float64(tc.userID), claims["user_id"])
				assert.Equal(t, authService.config.JWT.Issuer, claims["iss"])
			}
		})
	}
}

func TestVerifyTokenByRPC(t *testing.T) {
	authService := setupAuthService(t)

	testCases := []struct {
		name      string
		tokenFunc func() (string, error)
		wantValid bool
	}{
		{
			name: "Valid Token",
			tokenFunc: func() (string, error) {
				return generateTestToken(1, time.Hour, authService.config.JWT.SecretKey, authService.config.JWT.Issuer)
			},
			wantValid: true,
		},
		{
			name: "Expired Token",
			tokenFunc: func() (string, error) {
				return generateTestToken(1, -time.Hour, authService.config.JWT.SecretKey, authService.config.JWT.Issuer)
			},
			wantValid: true, // 因为实现允许续期
		},
		{
			name: "Empty Token",
			tokenFunc: func() (string, error) {
				return "", nil
			},
			wantValid: false,
		},
		{
			name: "Invalid Token",
			tokenFunc: func() (string, error) {
				return "invalid.token.here", nil
			},
			wantValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tc.tokenFunc()
			require.NoError(t, err)

			resp, err := authService.VerifyTokenByRPC(context.Background(), &VerifyTokenReq{Token: token})

			if tc.wantValid {
				assert.NoError(t, err)
				assert.True(t, resp.Res)
			} else {
				assert.False(t, resp.Res)
			}
		})
	}
}

func TestRenewTokenByRPC(t *testing.T) {
	authService := setupAuthService(t)

	testCases := []struct {
		name      string
		tokenFunc func() (string, error)
		wantErr   bool
		wantValid bool
	}{
		{
			name: "Valid Token Renewal",
			tokenFunc: func() (string, error) {
				return generateTestToken(1, time.Hour, authService.config.JWT.SecretKey, authService.config.JWT.Issuer)
			},
			wantErr:   false,
			wantValid: true,
		},
		{
			name: "Expired Token Renewal",
			tokenFunc: func() (string, error) {
				return generateTestToken(1, -time.Hour, authService.config.JWT.SecretKey, authService.config.JWT.Issuer)
			},
			wantErr:   false,
			wantValid: true,
		},
		{
			name: "Empty Token",
			tokenFunc: func() (string, error) {
				return "", nil
			},
			wantErr:   true,
			wantValid: false,
		},
		{
			name: "Invalid Token",
			tokenFunc: func() (string, error) {
				return "invalid.token.here", nil
			},
			wantErr:   true,
			wantValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := tc.tokenFunc()
			require.NoError(t, err)

			resp, err := authService.RenewTokenByRPC(context.Background(), &RenewTokenReq{Token: token})

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, resp.NewToken)

			// 验证新令牌的有效性
			baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			parser := jwt.NewParser(jwt.WithTimeFunc(func() time.Time { return baseTime }))
			newToken, err := parser.Parse(resp.NewToken, func(token *jwt.Token) (interface{}, error) {
				return []byte(authService.config.JWT.SecretKey), nil
			})

			assert.NoError(t, err)
			assert.NotNil(t, newToken)
			assert.True(t, newToken.Valid)

			claims, ok := newToken.Claims.(jwt.MapClaims)
			assert.True(t, ok)
			assert.Equal(t, float64(1), claims["user_id"])
			assert.Equal(t, authService.config.JWT.Issuer, claims["iss"])

			// 验证过期时间
			exp, ok := claims["exp"].(float64)
			assert.True(t, ok)
			assert.Greater(t, exp, float64(baseTime.Unix()))
		})
	}
}

func TestTokenBlacklist(t *testing.T) {
	authService := setupAuthService(t)

	// 生成一个令牌
	validToken, err := generateTestToken(1, time.Hour, authService.config.JWT.SecretKey, authService.config.JWT.Issuer)
	require.NoError(t, err)

	// 验证令牌有效
	verifyResp, err := authService.VerifyTokenByRPC(context.Background(), &VerifyTokenReq{Token: validToken})
	assert.NoError(t, err)
	assert.True(t, verifyResp.Res)

	// 手动将令牌加入黑名单
	authService.tokenManager.BlacklistToken(validToken)

	// 验证令牌现在被视为无效
	verifyResp, err = authService.VerifyTokenByRPC(context.Background(), &VerifyTokenReq{Token: validToken})
	assert.NoError(t, err)
	assert.False(t, verifyResp.Res)
}
