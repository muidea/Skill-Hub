package util

import (
	"errors"
	"fmt"

	cd "github.com/muidea/magicCommon/def"
	"log/slog"
)

// ErrorFactory 错误工厂，用于创建统一格式的错误
type ErrorFactory struct {
	module string
}

// NewErrorFactory 创建错误工厂
func NewErrorFactory(module string) *ErrorFactory {
	return &ErrorFactory{module: module}
}

// New 创建新错误
func (f *ErrorFactory) New(code cd.Code, message string) *cd.Error {
	return cd.NewError(code, fmt.Sprintf("[%s] %s", f.module, message))
}

// NewWithStack 创建包含堆栈跟踪的新错误
func (f *ErrorFactory) NewWithStack(code cd.Code, message string) *cd.Error {
	return cd.NewErrorWithStack(code, fmt.Sprintf("[%s] %s", f.module, message))
}

// Wrap 包装错误
func (f *ErrorFactory) Wrap(code cd.Code, err error, message string) *cd.Error {
	if err == nil {
		return nil
	}

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr
	}

	return cd.WrapError(code, err, fmt.Sprintf("[%s] %s", f.module, message))
}

// WrapWithStack 包装错误并添加堆栈跟踪
func (f *ErrorFactory) WrapWithStack(code cd.Code, err error, message string) *cd.Error {
	if err == nil {
		return nil
	}

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr
	}

	return cd.WrapErrorWithStack(code, err, fmt.Sprintf("[%s] %s", f.module, message))
}

// Wrapf 包装错误，支持格式化消息
func (f *ErrorFactory) Wrapf(code cd.Code, err error, format string, args ...interface{}) *cd.Error {
	return f.Wrap(code, err, fmt.Sprintf(format, args...))
}

// WrapfWithStack 包装错误并添加堆栈跟踪，支持格式化消息
func (f *ErrorFactory) WrapfWithStack(code cd.Code, err error, format string, args ...interface{}) *cd.Error {
	return f.WrapWithStack(code, err, fmt.Sprintf(format, args...))
}

// Common error factories
var (
	// DatabaseErrorFactory 数据库错误工厂
	DatabaseErrorFactory = NewErrorFactory("database")
	// SystemErrorFactory 系统错误工厂
	SystemErrorFactory = NewErrorFactory("system")
	// ValidationErrorFactory 验证错误工厂
	ValidationErrorFactory = NewErrorFactory("validation")
)

// JoinErrors 合并多个错误（Go 1.20+ 兼容）
func JoinErrors(errs ...error) error {
	return errors.Join(errs...)
}

// LogAndWrap 记录错误并包装
func LogAndWrap(operation string, err error, code cd.Code) *cd.Error {
	if err == nil {
		return nil
	}

	slog.Error("operation failed",
		"operation", operation,
		"error", err.Error())

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr
	}

	return cd.NewError(code, err.Error())
}

// LogAndWrapf 记录错误并包装，支持格式化消息
func LogAndWrapf(operation string, err error, code cd.Code, format string, args ...interface{}) *cd.Error {
	message := fmt.Sprintf(format, args...)
	slog.Error("operation failed",
		"operation", operation,
		"message", message,
		"error", err.Error())

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr
	}

	return cd.NewError(code, fmt.Sprintf("%s: %v", message, err))
}

// IsDatabaseError 检查是否为数据库错误
func IsDatabaseError(err error) bool {
	if err == nil {
		return false
	}

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr.Code == cd.DatabaseError
	}

	return false
}

// IsNotFoundError 检查是否为未找到错误
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr.Code == cd.NotFound
	}

	return false
}

// IsValidationError 检查是否为验证错误
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	if cdErr, ok := err.(*cd.Error); ok {
		return cdErr.Code == cd.InvalidParameter || cdErr.Code == cd.IllegalParam
	}

	return false
}
