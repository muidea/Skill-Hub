package util

import (
	"encoding/json"
	"reflect"
)

// ConvertValue 尝试将 any 类型的值转换为泛型类型 T
func ConvertValue[T any](rawVal any) (T, bool) {
	var zero T
	if rawVal == nil {
		return zero, false
	}

	// 使用反射获取目标类型和值
	targetType := reflect.TypeOf(zero)
	rawType := reflect.TypeOf(rawVal)

	// 如果类型直接匹配，直接返回
	if rawType == targetType {
		return rawVal.(T), true
	}

	// 处理数字类型转换
	if isNumber(rawVal) && isNumberType(targetType) {
		return convertNumber[T](rawVal)
	}

	// 处理字符串类型转换
	if rawType.Kind() == reflect.String && targetType.Kind() == reflect.String {
		return rawVal.(T), true
	}

	// 处理布尔类型转换
	if rawType.Kind() == reflect.Bool && targetType.Kind() == reflect.Bool {
		return rawVal.(T), true
	}

	// 处理数组/切片类型转换
	if rawType.Kind() == reflect.Slice && targetType.Kind() == reflect.Slice {
		return convertSlice[T](rawVal)
	}

	// 处理 map 类型转换
	if rawType.Kind() == reflect.Map && targetType.Kind() == reflect.Map {
		// 简单的类型断言尝试
		if converted, ok := rawVal.(T); ok {
			return converted, true
		}
	}

	// 尝试使用 json 编解码进行转换（作为最后的手段）
	return convertViaJSON[T](rawVal)
}

// isNumber 检查值是否为数字类型
func isNumber(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		json.Number:
		return true
	default:
		return false
	}
}

// isNumberType 检查反射类型是否为数字类型
func isNumberType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// convertNumber 将数字类型转换为目标数字类型
func convertNumber[T any](rawVal any) (T, bool) {
	var zero T
	targetType := reflect.TypeOf(zero)

	// 处理 json.Number 类型
	if num, ok := rawVal.(json.Number); ok {
		switch targetType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// 先尝试 Int64()，如果失败（可能是小数），则尝试 Float64() 然后截断
			if intVal, err := num.Int64(); err == nil {
				// 检查是否在目标类型的范围内
				if !isIntInRange(intVal, targetType) {
					return zero, false
				}
				return reflect.ValueOf(intVal).Convert(targetType).Interface().(T), true
			} else {
				// 可能是小数，尝试转换为浮点数然后截断
				if floatVal, err := num.Float64(); err == nil {
					intVal := int64(floatVal) // 截断小数部分
					// 检查是否在目标类型的范围内
					if !isIntInRange(intVal, targetType) {
						return zero, false
					}
					return reflect.ValueOf(intVal).Convert(targetType).Interface().(T), true
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if intVal, err := num.Int64(); err == nil && intVal >= 0 {
				// 检查是否在目标类型的范围内
				if !isUintInRange(uint64(intVal), targetType) {
					return zero, false
				}
				return reflect.ValueOf(uint64(intVal)).Convert(targetType).Interface().(T), true
			} else {
				// 可能是小数，尝试转换为浮点数然后截断
				if floatVal, err := num.Float64(); err == nil && floatVal >= 0 {
					intVal := int64(floatVal) // 截断小数部分
					// 检查是否在目标类型的范围内
					if !isUintInRange(uint64(intVal), targetType) {
						return zero, false
					}
					return reflect.ValueOf(uint64(intVal)).Convert(targetType).Interface().(T), true
				}
			}
		case reflect.Float32, reflect.Float64:
			if floatVal, err := num.Float64(); err == nil {
				// 检查是否在目标类型的范围内
				if !isFloatInRange(floatVal, targetType) {
					return zero, false
				}
				return reflect.ValueOf(floatVal).Convert(targetType).Interface().(T), true
			}
		}
		return zero, false
	}

	// 其他数字类型，使用反射进行转换
	rawValue := reflect.ValueOf(rawVal)
	if rawValue.CanConvert(targetType) {
		// 检查是否在目标类型的范围内
		if !isValueInRange(rawValue, targetType) {
			return zero, false
		}
		// 对于浮点数到整数的转换，需要额外检查是否溢出
		if (rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64) &&
			(targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Int64 ||
				targetType.Kind() >= reflect.Uint && targetType.Kind() <= reflect.Uint64) {
			floatVal := rawValue.Float()
			intVal := int64(floatVal)
			// 检查转换后的整数是否与原始浮点数相等（即没有小数部分丢失）
			if floatVal != float64(intVal) {
				// 有小数部分，允许截断
				// 但需要检查截断后的值是否在范围内
				if targetType.Kind() >= reflect.Int && targetType.Kind() <= reflect.Int64 {
					if !isIntInRange(intVal, targetType) {
						return zero, false
					}
				} else {
					if intVal < 0 || !isUintInRange(uint64(intVal), targetType) {
						return zero, false
					}
				}
			}
		}
		return rawValue.Convert(targetType).Interface().(T), true
	}

	return zero, false
}

// isIntInRange 检查 int64 值是否在目标整数类型的范围内
func isIntInRange(val int64, targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Int8:
		return val >= -128 && val <= 127
	case reflect.Int16:
		return val >= -32768 && val <= 32767
	case reflect.Int32:
		return val >= -2147483648 && val <= 2147483647
	case reflect.Int64, reflect.Int:
		return true
	default:
		return false
	}
}

// isUintInRange 检查 uint64 值是否在目标无符号整数类型的范围内
func isUintInRange(val uint64, targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Uint8:
		return val <= 255
	case reflect.Uint16:
		return val <= 65535
	case reflect.Uint32:
		return val <= 4294967295
	case reflect.Uint64, reflect.Uint:
		return true
	default:
		return false
	}
}

// isFloatInRange 检查 float64 值是否在目标浮点类型的范围内
func isFloatInRange(val float64, targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Float32:
		// 检查是否在 float32 的范围内
		const maxFloat32 = 3.40282346638528859811704183484516925440e+38
		const minFloat32 = -maxFloat32
		return val >= minFloat32 && val <= maxFloat32
	case reflect.Float64:
		return true
	default:
		return false
	}
}

// isValueInRange 检查原始值是否在目标类型的范围内
func isValueInRange(rawValue reflect.Value, targetType reflect.Type) bool {
	// 获取原始值的 int64 表示
	var intVal int64
	var uintVal uint64
	var floatVal float64

	switch rawValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal = rawValue.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		uintVal = rawValue.Uint()
	case reflect.Float32, reflect.Float64:
		floatVal = rawValue.Float()
	default:
		return true // 非数字类型，直接返回 true
	}

	// 根据目标类型检查范围
	switch targetType.Kind() {
	case reflect.Int8:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 int8 范围内（允许截断）
			return floatVal >= -128 && floatVal <= 127
		}
		if rawValue.Kind() == reflect.Uint || rawValue.Kind() == reflect.Uint8 || rawValue.Kind() == reflect.Uint16 || rawValue.Kind() == reflect.Uint32 || rawValue.Kind() == reflect.Uint64 || rawValue.Kind() == reflect.Uintptr {
			return uintVal <= 127
		}
		return intVal >= -128 && intVal <= 127
	case reflect.Int16:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 int16 范围内（允许截断）
			return floatVal >= -32768 && floatVal <= 32767
		}
		if rawValue.Kind() == reflect.Uint || rawValue.Kind() == reflect.Uint8 || rawValue.Kind() == reflect.Uint16 || rawValue.Kind() == reflect.Uint32 || rawValue.Kind() == reflect.Uint64 || rawValue.Kind() == reflect.Uintptr {
			return uintVal <= 32767
		}
		return intVal >= -32768 && intVal <= 32767
	case reflect.Int32:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 int32 范围内（允许截断）
			return floatVal >= -2147483648 && floatVal <= 2147483647
		}
		if rawValue.Kind() == reflect.Uint || rawValue.Kind() == reflect.Uint8 || rawValue.Kind() == reflect.Uint16 || rawValue.Kind() == reflect.Uint32 || rawValue.Kind() == reflect.Uint64 || rawValue.Kind() == reflect.Uintptr {
			return uintVal <= 2147483647
		}
		return intVal >= -2147483648 && intVal <= 2147483647
	case reflect.Int64, reflect.Int:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 int64 范围内（允许截断）
			return floatVal >= -9223372036854775808 && floatVal <= 9223372036854775807
		}
		// int 和 int64 可以容纳所有整数
		return true
	case reflect.Uint8:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 uint8 范围内（允许截断）
			return floatVal >= 0 && floatVal <= 255
		}
		if rawValue.Kind() == reflect.Int || rawValue.Kind() == reflect.Int8 || rawValue.Kind() == reflect.Int16 || rawValue.Kind() == reflect.Int32 || rawValue.Kind() == reflect.Int64 {
			return intVal >= 0 && intVal <= 255
		}
		return uintVal <= 255
	case reflect.Uint16:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 uint16 范围内（允许截断）
			return floatVal >= 0 && floatVal <= 65535
		}
		if rawValue.Kind() == reflect.Int || rawValue.Kind() == reflect.Int8 || rawValue.Kind() == reflect.Int16 || rawValue.Kind() == reflect.Int32 || rawValue.Kind() == reflect.Int64 {
			return intVal >= 0 && intVal <= 65535
		}
		return uintVal <= 65535
	case reflect.Uint32:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 uint32 范围内（允许截断）
			return floatVal >= 0 && floatVal <= 4294967295
		}
		if rawValue.Kind() == reflect.Int || rawValue.Kind() == reflect.Int8 || rawValue.Kind() == reflect.Int16 || rawValue.Kind() == reflect.Int32 || rawValue.Kind() == reflect.Int64 {
			return intVal >= 0 && intVal <= 4294967295
		}
		return uintVal <= 4294967295
	case reflect.Uint64, reflect.Uint:
		if rawValue.Kind() == reflect.Float32 || rawValue.Kind() == reflect.Float64 {
			// 对于浮点数，检查是否在 uint64 范围内（允许截断）
			return floatVal >= 0 && floatVal <= 18446744073709551615
		}
		// 对于无符号类型，检查原始值是否为负数
		if rawValue.Kind() == reflect.Int || rawValue.Kind() == reflect.Int8 || rawValue.Kind() == reflect.Int16 || rawValue.Kind() == reflect.Int32 || rawValue.Kind() == reflect.Int64 {
			return intVal >= 0
		}
		return true
	case reflect.Float32:
		// 检查是否在 float32 的范围内
		const maxFloat32 = 3.40282346638528859811704183484516925440e+38
		const minFloat32 = -maxFloat32
		return floatVal >= minFloat32 && floatVal <= maxFloat32
	case reflect.Float64:
		return true
	default:
		return true
	}
}

// convertSlice 尝试将切片转换为目标切片类型
func convertSlice[T any](rawVal any) (T, bool) {
	var zero T
	rawSlice := reflect.ValueOf(rawVal)
	if rawSlice.Kind() != reflect.Slice {
		return zero, false
	}

	targetType := reflect.TypeOf(zero)
	if targetType.Kind() != reflect.Slice {
		return zero, false
	}

	// 创建目标切片
	elemType := targetType.Elem()
	targetSlice := reflect.MakeSlice(targetType, rawSlice.Len(), rawSlice.Len())

	for i := 0; i < rawSlice.Len(); i++ {
		elem := rawSlice.Index(i).Interface()
		// 递归转换每个元素
		converted, ok := convertValueByReflection(elem, elemType)
		if !ok {
			return zero, false
		}
		targetSlice.Index(i).Set(reflect.ValueOf(converted))
	}

	return targetSlice.Interface().(T), true
}

// convertValueByReflection 辅助函数，用于递归转换值
func convertValueByReflection(value any, targetType reflect.Type) (any, bool) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), true
	}

	// 简化实现：只处理基本类型
	valType := reflect.TypeOf(value)
	if valType == targetType {
		return value, true
	}

	// 尝试数字转换
	if isNumber(value) && isNumberType(targetType) {
		converted, ok := convertNumberByReflection(value, targetType)
		if ok {
			return converted, true
		}
	}

	// 尝试字符串转换
	if valType.Kind() == reflect.String && targetType.Kind() == reflect.String {
		return value, true
	}

	// 尝试布尔转换
	if valType.Kind() == reflect.Bool && targetType.Kind() == reflect.Bool {
		return value, true
	}

	// 尝试 any 类型转换（目标类型是 interface{}）
	if targetType.Kind() == reflect.Interface && targetType.NumMethod() == 0 {
		return value, true
	}

	return nil, false
}

// convertNumberByReflection 反射版本的数字转换
func convertNumberByReflection(value any, targetType reflect.Type) (any, bool) {
	// 处理 json.Number 类型
	if num, ok := value.(json.Number); ok {
		switch targetType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if intVal, err := num.Int64(); err == nil {
				// 检查是否在目标类型的范围内
				if !isIntInRange(intVal, targetType) {
					return nil, false
				}
				return reflect.ValueOf(intVal).Convert(targetType).Interface(), true
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if intVal, err := num.Int64(); err == nil && intVal >= 0 {
				// 检查是否在目标类型的范围内
				if !isUintInRange(uint64(intVal), targetType) {
					return nil, false
				}
				return reflect.ValueOf(uint64(intVal)).Convert(targetType).Interface(), true
			}
		case reflect.Float32, reflect.Float64:
			if floatVal, err := num.Float64(); err == nil {
				// 检查是否在目标类型的范围内
				if !isFloatInRange(floatVal, targetType) {
					return nil, false
				}
				return reflect.ValueOf(floatVal).Convert(targetType).Interface(), true
			}
		}
		return nil, false
	}

	rawValue := reflect.ValueOf(value)
	if rawValue.CanConvert(targetType) {
		// 检查是否在目标类型的范围内
		if !isValueInRange(rawValue, targetType) {
			return nil, false
		}
		return rawValue.Convert(targetType).Interface(), true
	}
	return nil, false
}

// convertViaJSON 使用 JSON 编解码进行转换
func convertViaJSON[T any](rawVal any) (T, bool) {
	var zero T

	// 检查 rawVal 是否为指针类型
	rawType := reflect.TypeOf(rawVal)
	if rawType != nil && rawType.Kind() == reflect.Ptr {
		// 对于指针类型，直接返回失败
		return zero, false
	}

	// 将 rawVal 编码为 JSON
	jsonBytes, err := json.Marshal(rawVal)
	if err != nil {
		return zero, false
	}

	// 将 JSON 解码为目标类型
	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return zero, false
	}

	return result, true
}
