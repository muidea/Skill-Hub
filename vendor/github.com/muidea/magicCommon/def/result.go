package def

import (
	"fmt"
	"runtime"
	"strings"
)

type Code int

const (
	Success = iota
	// UnKnownError 未知错误
	UnKnownError
	// NotFound 未找到
	NotFound
	// InvalidParam 无效参数
	InvalidParameter
	// IllegalParam 非法参数
	IllegalParam
	// InvalidAuthority 非法授权
	InvalidAuthority
	// Unexpected 意外错误
	Unexpected
	// Duplicated 重复
	Duplicated
	// DatabaseError 数据库错误
	DatabaseError
	// Timeout 超时
	Timeout
	// NetworkError 网络错误
	NetworkError
	// Unauthorized 未授权
	Unauthorized
	// Forbidden 禁止访问
	Forbidden
	// ResourceExhausted 资源耗尽
	ResourceExhausted
	// TooManyRequests 请求过多
	TooManyRequests
	// ServiceUnavailable 服务不可用
	ServiceUnavailable
	// NotImplemented 未实现
	NotImplemented
	// BadGateway 网关错误
	BadGateway
	// DataCorrupted 数据损坏
	DataCorrupted
	// VersionConflict 版本冲突
	VersionConflict
	// ExternalServiceError 外部服务错误
	ExternalServiceError
	// InvalidOperation 无效操作
	InvalidOperation
	// PermissionDenied 权限不足
	PermissionDenied
)

type Error struct {
	Code       Code   `json:"code"`
	Message    string `json:"message"`
	StackTrace string `json:"stackTrace,omitempty"`
	Cause      error  `json:"cause,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "code:%d, message:%s", e.Code, e.Message)

	if e.Cause != nil {
		fmt.Fprintf(&builder, ", cause:%v", e.Cause)
	}

	if e.StackTrace != "" {
		// 在错误字符串中不包含完整的堆栈跟踪，避免日志过长
		builder.WriteString(" [with stack trace]")
	}

	return builder.String()
}

// Unwrap 支持错误链解包
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

// ToStdError safely converts a custom error pointer to the standard error interface.
func ToStdError(err *Error) error {
	if err == nil {
		return nil
	}

	return err
}

// NewError 创建新错误（不包含堆栈跟踪）
func NewError(errorCode Code, errorMessage string) *Error {
	return &Error{
		Code:    errorCode,
		Message: errorMessage,
	}
}

// NewErrorWithStack 创建包含堆栈跟踪的新错误
func NewErrorWithStack(errorCode Code, errorMessage string) *Error {
	return &Error{
		Code:       errorCode,
		Message:    errorMessage,
		StackTrace: captureStackTrace(2), // 跳过当前函数和调用函数
	}
}

// WrapError 包装现有错误
func WrapError(errorCode Code, err error, message string) *Error {
	if err == nil {
		return nil
	}

	// 如果已经是自定义错误类型，直接返回
	if cdErr, ok := err.(*Error); ok {
		return cdErr
	}

	return &Error{
		Code:    errorCode,
		Message: fmt.Sprintf("%s: %v", message, err),
		Cause:   err,
	}
}

// WrapErrorWithStack 包装现有错误并添加堆栈跟踪
func WrapErrorWithStack(errorCode Code, err error, message string) *Error {
	if err == nil {
		return nil
	}

	// 如果已经是自定义错误类型，直接返回
	if cdErr, ok := err.(*Error); ok {
		return cdErr
	}

	return &Error{
		Code:       errorCode,
		Message:    fmt.Sprintf("%s: %v", message, err),
		Cause:      err,
		StackTrace: captureStackTrace(2), // 跳过当前函数和调用函数
	}
}

// captureStackTrace 捕获堆栈跟踪
func captureStackTrace(skip int) string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])

	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])
	var builder strings.Builder

	for {
		frame, more := frames.Next()
		fmt.Fprintf(&builder, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)

		if !more {
			break
		}
	}

	return builder.String()
}

// HasStackTrace 检查错误是否有堆栈跟踪
func (e *Error) HasStackTrace() bool {
	if e == nil {
		return false
	}

	return e.StackTrace != ""
}

// GetFullStackTrace 获取完整的堆栈跟踪（包括原因链）
func (e *Error) GetFullStackTrace() string {
	if e == nil {
		return ""
	}

	var builder strings.Builder

	// 添加当前错误的堆栈跟踪
	if e.StackTrace != "" {
		builder.WriteString("Current error stack trace:\n")
		builder.WriteString(e.StackTrace)
	}

	// 递归添加原因链的堆栈跟踪
	if e.Cause != nil {
		if causeErr, ok := e.Cause.(*Error); ok && causeErr.StackTrace != "" {
			builder.WriteString("\nCaused by:\n")
			builder.WriteString(causeErr.GetFullStackTrace())
		}
	}

	return builder.String()
}

// ResultWithError 表示包含错误的结果接口
type ResultWithError interface {
	GetError() *Error
}

type Result struct {
	Error *Error `json:"error"`
}

// Success 成功
func (s *Result) Success() bool {
	return s.Error == nil || s.Error.Code == Success
}

// Fail 失败
func (s *Result) Fail() bool {
	return s.Error != nil && s.Error.Code != Success
}

func NewResult() *Result {
	return &Result{
		Error: NewError(Unexpected, "unexpected error, default error message"),
	}
}

// GetError 返回错误指针，便于实现 ResultWithError 接口
func (r *Result) GetError() *Error {
	return r.Error
}

// SetError 设置错误
func (r *Result) SetError(err *Error) {
	r.Error = err
}

// NewSuccessResult 创建一个成功的 Result（无错误）
func NewSuccessResult() *Result {
	return &Result{
		Error: nil,
	}
}
