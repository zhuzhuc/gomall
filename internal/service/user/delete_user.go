package user

import (
	"context"
	"database/sql"
	"fmt"

	userapi "github.com/bytedance-youthcamp/demo/api/user"
	"golang.org/x/crypto/bcrypt"
)

// DeleteUser handles the deletion of a user account
// It requires the user's ID and password for verification
func (s *UserService) DeleteUser(ctx context.Context, req *userapi.DeleteUserRequest) (*userapi.DeleteUserResponse, error) {
	// 开始事务
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// 验证用户是否存在并获取密码哈希
	var (
		userID         int32
		passwordHash   string
		twoFactorToken string
	)
	query := `
		SELECT id, password_hash, COALESCE(two_factor_token, '')
		FROM users 
		WHERE id = ?
	`
	err = tx.QueryRowContext(ctx, query, req.UserId).Scan(
		&userID,
		&passwordHash,
		&twoFactorToken,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return &userapi.DeleteUserResponse{
				Success:      false,
				ErrorMessage: "用户不存在",
			}, nil
		}
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return &userapi.DeleteUserResponse{
			Success:      false,
			ErrorMessage: "密码错误",
		}, nil
	}

	// 删除用户的备份码
	_, err = tx.ExecContext(ctx, "DELETE FROM two_factor_backup_codes WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete two-factor backup codes: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM user_backup_codes WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user backup codes: %w", err)
	}

	// 删除用户的支付和订单信息
	_, err = tx.ExecContext(ctx, "DELETE FROM payments WHERE order_id IN (SELECT id FROM orders WHERE user_id = ?)", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete payments: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM orders WHERE user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete orders: %w", err)
	}

	// 删除用户
	result, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	// 检查是否删除成功
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return &userapi.DeleteUserResponse{
			Success:      false,
			ErrorMessage: "删除用户失败",
		}, nil
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &userapi.DeleteUserResponse{
		Success: true,
	}, nil
}
