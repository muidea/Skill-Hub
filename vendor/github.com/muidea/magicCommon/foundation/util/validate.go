package util

import (
	"fmt"
	"reflect"

	sysValidator "github.com/go-playground/validator/v10"
)

// ValidateFunc 校验是否是函数
func ValidateFunc(fun interface{}) error {
	if reflect.TypeOf(fun).Kind() != reflect.Func {
		return fmt.Errorf("fun must be a callable func")
	}
	return nil
}

// ValidatePtr 校验是否是指针
func ValidatePtr(ptr interface{}) error {
	if reflect.TypeOf(ptr).Kind() != reflect.Ptr {
		return fmt.Errorf("ptr must be a object pointer")
	}
	return nil
}

type Validator interface {
	Validate(value interface{}) error
}

func NewFormValidator() Validator {
	return &validator{validate: sysValidator.New()}
}

type validator struct {
	validate *sysValidator.Validate
}

func (s *validator) Validate(value interface{}) error {
	return s.validate.Struct(value)
}
