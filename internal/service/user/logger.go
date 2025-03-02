package user

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type UserServiceLogger struct {
	logger *zap.Logger
}

func NewUserServiceLogger() *UserServiceLogger {
	// 日志轮转配置
	logRotator := &lumberjack.Logger{
		Filename:   "/var/log/user-service/user.log", // 日志文件路径
		MaxSize:    100,   // 单个日志文件最大 100 MB
		MaxBackups: 3,     // 保留 3 个备份
		MaxAge:     28,    // 保留 28 天的日志
		Compress:   true,  // 启用压缩
	}

	// 日志编码配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// 创建日志核心
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logRotator),
		zap.NewAtomicLevelAt(zap.InfoLevel),
	)

	// 创建日志记录器
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))

	return &UserServiceLogger{
		logger: logger,
	}
}

// 记录用户注册日志
func (l *UserServiceLogger) LogRegistration(userID int32, success bool, reason string) {
	if success {
		l.logger.Info("User Registration",
			zap.Int32("user_id", userID),
			zap.Bool("success", true),
		)
	} else {
		l.logger.Warn("User Registration Failed",
			zap.Int32("user_id", userID),
			zap.Bool("success", false),
			zap.String("reason", reason),
		)
	}
}

// 记录用户登录日志
func (l *UserServiceLogger) LogLogin(username string, success bool, reason string) {
	if success {
		l.logger.Info("User Login",
			zap.String("username", username),
			zap.Bool("success", true),
		)
	} else {
		l.logger.Warn("User Login Failed",
			zap.String("username", username),
			zap.Bool("success", false),
			zap.String("reason", reason),
		)
	}
}

// 记录用户信息更新日志
func (l *UserServiceLogger) LogUserInfoUpdate(userID int32, success bool, reason string) {
	if success {
		l.logger.Info("User Info Updated",
			zap.Int32("user_id", userID),
			zap.Bool("success", true),
		)
	} else {
		l.logger.Warn("User Info Update Failed",
			zap.Int32("user_id", userID),
			zap.Bool("success", false),
			zap.String("reason", reason),
		)
	}
}

// 记录安全相关事件
func (l *UserServiceLogger) LogSecurityEvent(eventType string, details map[string]interface{}) {
	l.logger.Warn("Security Event",
		zap.String("event_type", eventType),
		zap.Any("details", details),
	)
}

// 关闭日志记录器
func (l *UserServiceLogger) Close() {
	l.logger.Sync()
}
