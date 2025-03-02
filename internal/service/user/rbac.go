package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Permission 定义权限结构
type Permission struct {
	ID          int32
	Name        string
	Description string
	Resource    string
	Action      string
}

// Role 定义角色结构
type Role struct {
	ID          int32
	Name        string
	Description string
	Permissions []Permission
}

// RBACManager 权限管理器
type RBACManager struct {
	db           *sql.DB
	mutex        sync.RWMutex
	permCache    map[int32][]Permission
	cacheTimeout time.Duration
}

// NewRBACManager 创建权限管理器实例
func NewRBACManager(db *sql.DB) *RBACManager {
	return &RBACManager{
		db:           db,
		permCache:    make(map[int32][]Permission),
		cacheTimeout: 5 * time.Minute,
	}
}

// GetUserPermissions 获取用户权限
func (rm *RBACManager) GetUserPermissions(ctx context.Context, userID int32) ([]Permission, error) {
	// 首先尝试从缓存获取
	rm.mutex.RLock()
	if perms, ok := rm.permCache[userID]; ok {
		rm.mutex.RUnlock()
		return perms, nil
	}
	rm.mutex.RUnlock()

	// 从数据库获取权限
	query := `
		SELECT DISTINCT p.id, p.name, p.description, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ?
	`

	rows, err := rm.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, p)
	}

	// 更新缓存
	rm.mutex.Lock()
	rm.permCache[userID] = permissions
	rm.mutex.Unlock()

	// 异步更新权限缓存表
	go rm.updatePermissionCache(userID, permissions)

	return permissions, nil
}

// HasPermission 检查用户是否有特定权限
func (rm *RBACManager) HasPermission(ctx context.Context, userID int32, resource, action string) (bool, error) {
	perms, err := rm.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, p := range perms {
		if p.Resource == resource && p.Action == action {
			return true, nil
		}
	}

	return false, nil
}

// updatePermissionCache 更新权限缓存表
func (rm *RBACManager) updatePermissionCache(userID int32, permissions []Permission) {
	permsJSON, err := json.Marshal(permissions)
	if err != nil {
		return
	}

	query := `
		INSERT INTO permission_cache (user_id, permissions, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
		permissions = ?,
		updated_at = ?
	`

	now := time.Now()
	_, err = rm.db.Exec(query, userID, string(permsJSON), now, string(permsJSON), now)
	if err != nil {
		// 记录错误日志，但不中断程序
		fmt.Printf("Failed to update permission cache: %v\n", err)
	}
}

// InvalidateCache 使指定用户的权限缓存失效
func (rm *RBACManager) InvalidateCache(userID int32) {
	rm.mutex.Lock()
	delete(rm.permCache, userID)
	rm.mutex.Unlock()
}

// AssignRole 为用户分配角色
func (rm *RBACManager) AssignRole(ctx context.Context, userID int32, roleName string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id)
		SELECT ?, id FROM roles WHERE name = ?
	`

	_, err := rm.db.ExecContext(ctx, query, userID, roleName)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	// 使缓存失效
	rm.InvalidateCache(userID)
	return nil
}