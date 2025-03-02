package user

// import (
// 	"context"
// 	"encoding/base32"
// 	"fmt"
// 	"log"
// 	"math/rand"
// 	"strconv"
// 	"time"

// 	user "github.com/bytedance-youthcamp/demo/api/user"
// 	"github.com/pquerna/otp/totp"
// )

// func init() {
// 	rand.Seed(time.Now().UnixNano())
// }

// type TwoFactorAuthManager struct {
// 	secretLength int
// }

// type TwoFactorSetup struct {
// 	Secret    string
// 	QRCodeURL string
// }

// func NewTwoFactorAuthManager() *TwoFactorAuthManager {
// 	return &TwoFactorAuthManager{
// 		secretLength: 32,
// 	}
// }

// func (m *TwoFactorAuthManager) GenerateTwoFactorSecret() (string, error) {
// 	// 生成随机密钥
// 	randomBytes := make([]byte, m.secretLength)
// 	_, err := rand.Read(randomBytes)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to generate random secret: %w", err)
// 	}

// 	// 使用 base32 编码
// 	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
// 	return secret, nil
// }

// func (m *TwoFactorAuthManager) SetupTwoFactor(username string) (*TwoFactorSetup, error) {
// 	secret, err := m.GenerateTwoFactorSecret()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate 2FA secret: %w", err)
// 	}

// 	// 生成 TOTP 密钥配置
// 	key, err := totp.Generate(totp.GenerateOpts{
// 		Issuer:      "ByteDance YouthCamp",
// 		AccountName: username,
// 		Secret:      []byte(secret),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
// 	}

// 	return &TwoFactorSetup{
// 		Secret:    secret,
// 		QRCodeURL: key.URL(),
// 	}, nil
// }

// func (m *TwoFactorAuthManager) ValidateTOTP(secret, passcode string) bool {
// 	// 验证 TOTP 码
// 	return totp.Validate(passcode, secret)
// }

// func (m *TwoFactorAuthManager) GenerateBackupCodes(count int) []string {
// 	backupCodes := make([]string, count)
// 	for i := 0; i < count; i++ {
// 		code := make([]byte, 8)
// 		rand.Read(code)
// 		backupCodes[i] = fmt.Sprintf("%x", code)[:8]
// 	}
// 	return backupCodes
// }

// func generateBackupCodes(count int) ([]string, error) {
// 	backupCodes := make([]string, count)
// 	for i := 0; i < count; i++ {
// 		// 生成 8 位随机备份码
// 		code := make([]byte, 4)
// 		_, err := rand.Read(code)
// 		if err != nil {
// 			return nil, err
// 		}
// 		backupCodes[i] = base32.StdEncoding.EncodeToString(code)[:8]
// 	}
// 	return backupCodes, nil
// }

// func generateRandomBackupCode() string {
// 	// 生成一个8位数字备份码
// 	code := rand.Intn(90000000) + 10000000
// 	return strconv.Itoa(code)
// }

// func (s *UserService) EnableTwoFactor(ctx context.Context, req *user.EnableTwoFactorRequest) (*user.EnableTwoFactorResponse, error) {
// 	// 生成两因素认证密钥
// 	secret, err := totp.Generate(totp.GenerateOpts{
// 		Issuer:      "ByteDance UserService",
// 		AccountName: fmt.Sprintf("user_%d", req.UserId),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to generate two-factor secret: %w", err)
// 	}

// 	// 生成备份码
// 	backupCodes := make([]string, 5)
// 	for i := 0; i < 5; i++ {
// 		code := generateRandomBackupCode()
// 		backupCodes[i] = code
// 	}

// 	// 开启事务
// 	tx, err := s.db.BeginTx(ctx, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to start transaction: %w", err)
// 	}
// 	defer tx.Rollback()

// 	// 更新用户两因素认证状态
// 	updateQuery := `
// 		UPDATE users 
// 		SET two_factor_enabled = 1, 
// 			two_factor_secret = ? 
// 		WHERE id = ?
// 	`
// 	_, err = tx.ExecContext(ctx, updateQuery, secret.Secret(), req.UserId)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to update user two-factor settings: %w", err)
// 	}

// 	// 插入备份码
// 	backupCodeQuery := `
// 		INSERT INTO user_backup_codes (user_id, backup_code, used) 
// 		VALUES (?, ?, 0)
// 	`
// 	for _, code := range backupCodes {
// 		_, err = tx.ExecContext(ctx, backupCodeQuery, req.UserId, code)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to insert backup code: %w", err)
// 		}
// 	}

// 	// 提交事务
// 	if err = tx.Commit(); err != nil {
// 		return nil, fmt.Errorf("failed to commit transaction: %w", err)
// 	}

// 	// 生成 QR 码 URL
// 	qrCodeURL := secret.URL()

// 	return &user.EnableTwoFactorResponse{
// 		Success:     true,
// 		Secret:      secret.Secret(),
// 		QrCodeUrl:   qrCodeURL,
// 		BackupCodes: backupCodes,
// 	}, nil
// }

// func (s *UserService) VerifyTwoFactor(ctx context.Context, req *user.VerifyTwoFactorRequest) (*user.VerifyTwoFactorResponse, error) {
// 	var secret string
// 	var backupCodes []string

// 	// 获取用户的两因素认证密钥和备份码
// 	query := `
// 		SELECT two_factor_secret, two_factor_backup_codes 
// 		FROM users 
// 		WHERE id = $1
// 	`
// 	err := s.db.QueryRowContext(ctx, query, req.UserId).Scan(&secret, &backupCodes)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve two-factor secret: %w", err)
// 	}

// 	// 检查 TOTP 码
// 	valid := totp.Validate(req.Code, secret)
// 	if valid {
// 		return &user.VerifyTwoFactorResponse{Success: true}, nil
// 	}

// 	// 检查备份码
// 	for i, code := range backupCodes {
// 		if code == req.Code {
// 			// 使用后删除备份码
// 			backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)

// 			updateQuery := `
// 				UPDATE users 
// 				SET two_factor_backup_codes = $1 
// 				WHERE id = $2
// 			`
// 			_, err = s.db.ExecContext(ctx, updateQuery, backupCodes, req.UserId)
// 			if err != nil {
// 				log.Printf("Failed to update backup codes: %v", err)
// 			}

// 			return &user.VerifyTwoFactorResponse{Success: true}, nil
// 		}
// 	}

// 	return &user.VerifyTwoFactorResponse{
// 		Success:      false,
// 		ErrorMessage: "无效的两因素认证码",
// 	}, nil
// }

// func (s *UserService) validateTwoFactorCode(ctx context.Context, userID int32, code string) (bool, error) {
// 	var secret string
// 	var backupCodes []string

// 	query := `
// 		SELECT two_factor_secret, two_factor_backup_codes 
// 		FROM users 
// 		WHERE id = $1
// 	`
// 	err := s.db.QueryRowContext(ctx, query, userID).Scan(&secret, &backupCodes)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to retrieve two-factor secret: %w", err)
// 	}

// 	// 验证 TOTP 码
// 	if totp.Validate(code, secret) {
// 		return true, nil
// 	}

// 	// 验证备份码
// 	for i, backupCode := range backupCodes {
// 		if backupCode == code {
// 			// 使用后删除备份码
// 			backupCodes = append(backupCodes[:i], backupCodes[i+1:]...)

// 			updateQuery := `
// 				UPDATE users 
// 				SET two_factor_backup_codes = $1 
// 				WHERE id = $2
// 			`
// 			_, err = s.db.ExecContext(ctx, updateQuery, backupCodes, userID)
// 			if err != nil {
// 				log.Printf("Failed to update backup codes: %v", err)
// 			}

// 			return true, nil
// 		}
// 	}

// 	return false, nil
// }
