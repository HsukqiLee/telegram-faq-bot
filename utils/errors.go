package utils

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// BotError 统一的错误类型
type BotError struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Detail    string    `json:"detail,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *BotError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
}

// 错误代码定义
const (
	ErrCodeValidation = 1001
	ErrCodeDatabase   = 1002
	ErrCodePermission = 1003
	ErrCodeAI         = 1004
	ErrCodeRateLimit  = 1005
	ErrCodeConfig     = 1006
	ErrCodeNetwork    = 1007
)

// 错误包装函数
func WrapError(err error, code int, message string) *BotError {
	detail := ""
	if err != nil {
		detail = err.Error()
	}

	return &BotError{
		Code:      code,
		Message:   message,
		Detail:    detail,
		Timestamp: time.Now(),
	}
}

// 常用错误创建函数
func ValidationError(message string, err error) *BotError {
	return WrapError(err, ErrCodeValidation, message)
}

func DatabaseError(message string, err error) *BotError {
	return WrapError(err, ErrCodeDatabase, message)
}

func PermissionError(message string) *BotError {
	return WrapError(nil, ErrCodePermission, message)
}

func AIError(message string, err error) *BotError {
	return WrapError(err, ErrCodeAI, message)
}

func RateLimitError(message string) *BotError {
	return WrapError(nil, ErrCodeRateLimit, message)
}

// 重试机制
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2.0,
	}
}

// WithRetry 执行带重试的函数
func WithRetry(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error
	wait := config.InitialWait

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt < config.MaxAttempts-1 {
			time.Sleep(wait)
			wait = time.Duration(float64(wait) * config.Multiplier)
			if wait > config.MaxWait {
				wait = config.MaxWait
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %v", config.MaxAttempts, lastErr)
}

// 断路器模式
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        string // "closed", "open", "half-open"
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	switch cb.state {
	case "open":
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failures = 0
		} else {
			return errors.New("circuit breaker is open")
		}
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.failures >= cb.maxFailures {
			cb.state = "open"
		}
		return err
	}

	// 成功执行
	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0

	return nil
}
