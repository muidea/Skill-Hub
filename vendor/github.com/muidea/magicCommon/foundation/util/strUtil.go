package util

import (
	"encoding/json"
	"strconv"
	"strings"
)

// MarshalString 将任意类型的值转换为 JSON 字符串
func MarshalString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	default:
		byteVal, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(byteVal)
	}
}

// UnmarshalString 将 JSON 字符串解析为相应的 Go 类型
func UnmarshalString(val string) interface{} {
	val = strings.TrimSpace(val)
	if val == "" {
		return nil
	}

	// 尝试解析为 map
	var mVal map[string]interface{}
	if err := json.Unmarshal([]byte(val), &mVal); err == nil {
		return mVal
	}

	// 尝试解析为 float64
	var fVal float64
	if err := json.Unmarshal([]byte(val), &fVal); err == nil {
		return fVal
	}

	// 尝试解析为 bool
	var bVal bool
	if err := json.Unmarshal([]byte(val), &bVal); err == nil {
		return bVal
	}

	// 尝试解析为 []float64
	var fArr []float64
	if err := json.Unmarshal([]byte(val), &fArr); err == nil {
		return fArr
	}

	// 尝试解析为 []bool
	var bArr []bool
	if err := json.Unmarshal([]byte(val), &bArr); err == nil {
		return bArr
	}

	// 尝试解析为 []string
	var sArr []string
	if err := json.Unmarshal([]byte(val), &sArr); err == nil {
		return sArr
	}

	// 如果以上都不成功，返回原始字符串
	return val
}

// ExtractSummary 抽取摘要
func ExtractSummary(content string) string {
	content = strings.TrimLeft(content, "\n")
	offset := strings.Index(content, "\n")
	if offset > 0 {
		return content[:offset]
	}
	return content
}

// cleanStr 清理字符串中的逗号和空格
func cleanStr(str string) string {
	str = strings.TrimSpace(str)
	if len(str) == 0 {
		return ""
	}
	if str[0] == ',' {
		str = str[1:]
	}
	if len(str) > 0 && str[len(str)-1] == ',' {
		str = str[:len(str)-1]
	}
	return strings.TrimSpace(str)
}

// Str2IntArray 将逗号分隔的字符串转换为整数数组
func Str2IntArray(str string) ([]int, bool) {
	str = cleanStr(str)
	if str == "" {
		return []int{}, true
	}

	vals := strings.Split(str, ",")
	ids := make([]int, 0, len(vals))
	for _, val := range vals {
		if val == "" {
			continue
		}
		id, err := strconv.Atoi(val)
		if err != nil {
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

// IntArray2Str 将整数数组转换为逗号分隔的字符串
func IntArray2Str(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.Itoa(id))
	}
	return sb.String()
}

func StringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
