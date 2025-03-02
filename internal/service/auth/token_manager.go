package auth

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bytedance-youthcamp/demo/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type TokenManager struct {
	blacklist     map[string]time.Time
	blacklistLock sync.RWMutex
	etcdClient    *clientv3.Client
}

func NewTokenManager(cfg *config.AuthConfig, options ...Option) (*TokenManager, error) {
	tm := &TokenManager{
		blacklist: make(map[string]time.Time),
	}

	// 应用可选配置
	for _, opt := range options {
		opt(tm)
	}

	// 如果没有提供自定义 ETCD 客户端，尝试使用配置创建
	if tm.etcdClient == nil && len(cfg.Registration.Etcd.Endpoints) > 0 {
		etcdClient, err := clientv3.New(clientv3.Config{
			Endpoints:   cfg.Registration.Etcd.Endpoints,
			DialTimeout: cfg.Registration.Etcd.DialTimeout,
		})
		if err != nil {
			log.Printf("Warning: Failed to create ETCD client: %v", err)
		} else {
			tm.etcdClient = etcdClient
			// 启动服务注册和令牌黑名单清理
			go tm.registerService(cfg)
			go tm.startBlacklistCleanup(cfg)
		}
	}

	return tm, nil
}

// Option 是一个配置 TokenManager 的函数类型
type Option func(*TokenManager)

// WithEtcdClient 允许注入自定义的 ETCD 客户端
func WithEtcdClient(client *clientv3.Client) Option {
	return func(tm *TokenManager) {
		tm.etcdClient = client
	}
}

func (tm *TokenManager) registerService(cfg *config.AuthConfig) {
	if tm.etcdClient == nil {
		return
	}

	session, err := concurrency.NewSession(tm.etcdClient)
	if err != nil {
		log.Printf("Failed to create etcd session: %v", err)
		return
	}
	defer session.Close()

	key := fmt.Sprintf("/services/%s/%s", cfg.Registration.ServiceName, cfg.Registration.ServiceVersion)
	value := "localhost:50051" // 假设服务监听的地址

	_, err = tm.etcdClient.Put(context.Background(), key, value, clientv3.WithLease(session.Lease()))
	if err != nil {
		log.Printf("Failed to register service: %v", err)
	}
}

func (tm *TokenManager) startBlacklistCleanup(cfg *config.AuthConfig) {
	if cfg.Security.TokenBlacklistCleanupPeriod <= 0 {
		return
	}

	ticker := time.NewTicker(cfg.Security.TokenBlacklistCleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		tm.cleanupBlacklist(cfg)
	}
}

func (tm *TokenManager) cleanupBlacklist(cfg *config.AuthConfig) {
	tm.blacklistLock.Lock()
	defer tm.blacklistLock.Unlock()

	now := time.Now()
	for token, blacklistedAt := range tm.blacklist {
		if now.Sub(blacklistedAt) > cfg.JWT.TokenTTL {
			delete(tm.blacklist, token)
		}
	}
}

func (tm *TokenManager) BlacklistToken(token string) {
	tm.blacklistLock.Lock()
	defer tm.blacklistLock.Unlock()

	// 如果黑名单已满，移除最早的令牌
	if len(tm.blacklist) >= config.GetConfig().Security.TokenBlacklistMaxSize {
		var oldestToken string
		var oldestTime time.Time

		for t, bt := range tm.blacklist {
			if oldestTime.IsZero() || bt.Before(oldestTime) {
				oldestToken = t
				oldestTime = bt
			}
		}

		delete(tm.blacklist, oldestToken)
	}

	tm.blacklist[token] = time.Now()
}

func (tm *TokenManager) IsTokenBlacklisted(token string) bool {
	tm.blacklistLock.RLock()
	defer tm.blacklistLock.RUnlock()

	_, exists := tm.blacklist[token]
	return exists
}

func (tm *TokenManager) Close() {
	if tm.etcdClient != nil {
		tm.etcdClient.Close()
	}
}
