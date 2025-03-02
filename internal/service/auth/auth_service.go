package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bytedance-youthcamp/demo/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

const (
	secretKey = "bytedance-youthcamp-secret-key"
	tokenTTL  = time.Hour * 24
)

type DeliverTokenReq struct {
	UserId int32
}

type VerifyTokenReq struct {
	Token string
}

type RenewTokenReq struct {
	Token string
}

type DeliveryResp struct {
	Token string
}

type VerifyResp struct {
	Res bool
}

type RenewTokenResp struct {
	NewToken string
}

type AuthService struct {
	config       *config.AuthConfig
	tokenManager *TokenManager
}

func NewAuthService() (*AuthService, error) {
	configPath := "/Users/Apple/Desktop/demo/configs/auth.yaml"
	authConfig, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	tokenManager, err := NewTokenManager(authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create token manager: %w", err)
	}

	return &AuthService{
		config:       authConfig,
		tokenManager: tokenManager,
	}, nil
}

func (s *AuthService) validateToken(tokenString string) (bool, error) {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.config.JWT.SecretKey), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithTimeFunc(func() time.Time { return baseTime }))
	
	if err != nil {
		// 对于过期的令牌，返回 true
		if err == jwt.ErrTokenExpired {
			log.Println("Token expired")
			return true, jwt.ErrTokenExpired
		}
		return false, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, fmt.Errorf("invalid token claims")
	}

	// 验证令牌签发者
	if iss, ok := claims["iss"].(string); !ok || iss != s.config.JWT.Issuer {
		return false, fmt.Errorf("invalid issuer")
	}

	// 检查令牌是否被列入黑名单
	if s.tokenManager.IsTokenBlacklisted(tokenString) {
		return false, fmt.Errorf("token is blacklisted")
	}

	return token.Valid, nil
}

func (s *AuthService) DeliverTokenByRPC(ctx context.Context, req *DeliverTokenReq) (*DeliveryResp, error) {
	if req.UserId <= 0 {
		log.Printf("Invalid user ID: %d", req.UserId)
		return nil, fmt.Errorf("invalid user id: %d", req.UserId)
	}

	token, err := s.generateToken(req.UserId)
	if err != nil {
		log.Printf("Failed to generate token for user ID %d: %v", req.UserId, err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	log.Printf("Token delivered for user ID: %d", req.UserId)
	return &DeliveryResp{Token: token}, nil
}

func (s *AuthService) VerifyTokenByRPC(ctx context.Context, req *VerifyTokenReq) (*VerifyResp, error) {
    if req.Token == "" {
        log.Println("Empty token provided for verification")
        return &VerifyResp{Res: false}, nil
    }

    // 检查令牌是否在黑名单中
    if s.tokenManager.IsTokenBlacklisted(req.Token) {
        log.Println("Token is blacklisted")
        return &VerifyResp{Res: false}, nil
    }

    baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    parser := jwt.NewParser(jwt.WithTimeFunc(func() time.Time { return baseTime }))
    
    // 尝试解析令牌，即使过期也要解析
    claims := jwt.MapClaims{}
    token, _ := parser.ParseWithClaims(req.Token, &claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        return []byte(s.config.JWT.SecretKey), nil
    })

    // 检查令牌是否存在且有效
    if token == nil || len(claims) == 0 {
        return &VerifyResp{Res: false}, nil
    }

    // 验证令牌签发者
    if iss, ok := claims["iss"].(string); !ok || iss != s.config.JWT.Issuer {
        return &VerifyResp{Res: false}, nil
    }

    // 对于过期的令牌，返回 true（支持续期）
    if token.Valid == false {
        log.Println("Token expired, needs renewal")
        return &VerifyResp{Res: true}, nil
    }

    log.Printf("Token verification result: %v", token.Valid)
    return &VerifyResp{Res: token.Valid}, nil
}

func (s *AuthService) RenewTokenByRPC(ctx context.Context, req *RenewTokenReq) (*RenewTokenResp, error) {
	if req.Token == "" {
		log.Println("Empty token provided for renewal")
		return nil, fmt.Errorf("empty token")
	}

	// 解析原始令牌获取用户ID
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	parser := jwt.NewParser(jwt.WithTimeFunc(func() time.Time { return baseTime }))

	// 即使令牌过期也尝试解析
	claims := jwt.MapClaims{}
	token, _ := parser.ParseWithClaims(req.Token, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWT.SecretKey), nil
	})

	// 检查令牌是否有效或是否过期
	if token == nil || len(claims) == 0 {
		return nil, fmt.Errorf("invalid token format")
	}

	// 提取用户ID
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	// 检查令牌签发者
	issuer, ok := claims["iss"].(string)
	if !ok || issuer != s.config.JWT.Issuer {
		return nil, fmt.Errorf("invalid token issuer")
	}

	// 生成新令牌
	newToken, err := s.generateToken(int32(userID))
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}

	return &RenewTokenResp{NewToken: newToken}, nil
}

func (s *AuthService) generateToken(userID int32) (string, error) {
	baseTime := time.Now().UTC() // 使用当前时间
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     baseTime.Add(s.config.JWT.TokenTTL).Unix(), // 使用配置的 TokenTTL
		"iat":     baseTime.Unix(),
		"iss":     s.config.JWT.Issuer,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.config.JWT.SecretKey))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

// BlacklistToken 将令牌加入黑名单
func (s *AuthService) BlacklistToken(token string) {
	if s.tokenManager != nil {
		s.tokenManager.BlacklistToken(token)
	}
}

// Close 方法用于清理资源
func (s *AuthService) Close() {
	if s.tokenManager != nil {
		s.tokenManager.Close()
	}
}
