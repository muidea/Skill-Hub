package util

import (
	"regexp"
	"sort"
)

// ExistIntArray 是否存在数组中
func ExistIntArray(val int, array []int) bool {
	found := false
	for _, v := range array {
		if val == v {
			found = true
			break
		}
	}

	return found
}

// SameIntArray 是否存在数组中
func SameIntArray(val []int, array []int) bool {
	if len(val) != len(array) {
		return false
	}

	for _, v := range array {
		if !ExistIntArray(v, val) {
			return false
		}
	}

	return true
}

var telephonePattern = "1(3\\d|4[5-9]|5[0-35-9]|6[567]|7[0-8]|8\\d|9[0-35-9])\\d{8}"
var telephoneReg = regexp.MustCompile(telephonePattern)

func ExtractTelephone(val string) (ret string) {
	ret = telephoneReg.FindString(val)
	return
}

type StringSet []string

func (s StringSet) Add(val string) StringSet {
	exist := false
	for _, sv := range s {
		if sv == val {
			exist = true
			break
		}
	}

	if !exist {
		s = append(s, val)
	}
	return s
}

func (s StringSet) Remove(val string) StringSet {
	newVal := []string{}
	for _, sv := range s {
		if sv != val {
			newVal = append(newVal, sv)
		}
	}

	s = newVal
	return s
}

func (s StringSet) Empty() bool {
	return len(s) == 0
}

func (s StringSet) Exist(val string) bool {
	for _, sv := range s {
		if sv == val {
			return true
		}
	}
	return false
}

// 有序字符串集
type StringSortSet []string

func (s StringSortSet) Len() int {
	return len(s)
}

func (s StringSortSet) Add(val string) StringSortSet {
	if s.Exist(val) {
		return s
	}

	// 使用二分查找确定插入位置
	index := sort.SearchStrings(s, val)

	// 在正确位置插入新元素
	newSet := make(StringSortSet, len(s)+1)
	copy(newSet[:index], s[:index])
	newSet[index] = val
	copy(newSet[index+1:], s[index:])

	return newSet
}

func (s StringSortSet) Remove(val string) StringSortSet {
	for i, sv := range s {
		if sv == val {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func (s StringSortSet) Exist(val string) bool {
	for _, sv := range s {
		if sv == val {
			return true
		}
	}
	return false
}

func (s StringSortSet) Range(f func(string) bool) {
	for _, sv := range s {
		if !f(sv) {
			break
		}
	}
}
