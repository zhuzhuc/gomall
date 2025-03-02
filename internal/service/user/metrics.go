package user

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 用户注册指标
	userRegistrations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_registrations_total",
			Help: "Total number of user registrations",
		},
		[]string{"status"},
	)

	// 用户登录指标
	userLogins = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_logins_total",
			Help: "Total number of user login attempts",
		},
		[]string{"status"},
	)

	// 用户信息更新指标
	userInfoUpdates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_info_updates_total",
			Help: "Total number of user info updates",
		},
		[]string{"status"},
	)

	// 登录失败次数指标
	loginFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_login_failures_total",
			Help: "Total number of login failures",
		},
		[]string{"reason"},
	)

	// 用户服务请求延迟
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "user_service_request_duration_seconds",
			Help:    "User service request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)
)

// 记录用户注册指标
func (s *UserService) recordRegistrationMetric(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	userRegistrations.WithLabelValues(status).Inc()
}

// 记录用户登录指标
func (s *UserService) recordLoginMetric(success bool, reason string) {
	status := "success"
	if !success {
		status = "failure"
		loginFailures.WithLabelValues(reason).Inc()
	}
	userLogins.WithLabelValues(status).Inc()
}

// 记录用户信息更新指标
func (s *UserService) recordUserInfoUpdateMetric(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	userInfoUpdates.WithLabelValues(status).Inc()
}

// 记录请求延迟
func (s *UserService) observeRequestDuration(method string, success bool, duration float64) {
	status := "success"
	if !success {
		status = "failure"
	}
	requestDuration.WithLabelValues(method, status).Observe(duration)
}
