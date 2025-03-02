package user

import (
	"context"
	"fmt"

	"github.com/pquerna/otp/totp"
)

// validateTwoFactorCode validates a two-factor authentication code for a user
func (s *UserService) validateTwoFactorCode(ctx context.Context, userID int32, code string) (bool, error) {
	var secret string

	// 获取用户的两因素认证密钥
	query := `
		SELECT two_factor_secret FROM users WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&secret)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve two-factor secret: %w", err)
	}

	// 验证 TOTP 码
	valid := totp.Validate(code, secret)
	if valid {
		return true, nil
	}

	// 如果 TOTP 验证失败，检查是否是备份码
	backupQuery := `
		SELECT backup_code FROM user_backup_codes 
		WHERE user_id = $1 AND backup_code = $2 AND used = 0
	`
	var backupCode string
	err = s.db.QueryRowContext(ctx, backupQuery, userID, code).Scan(&backupCode)
	if err == nil {
		// 找到有效的备份码，将其标记为已使用
		updateQuery := `
			UPDATE user_backup_codes 
			SET used = 1 
			WHERE user_id = $1 AND backup_code = $2
		`
		_, err = s.db.ExecContext(ctx, updateQuery, userID, code)
		if err != nil {
			// 记录错误但不影响验证结果
			fmt.Printf("Failed to mark backup code as used: %v\n", err)
		}
		return true, nil
	}

	// 验证失败
	return false, nil
}