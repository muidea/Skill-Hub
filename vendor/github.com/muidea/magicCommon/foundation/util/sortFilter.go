package util

import (
	"fmt"
	"strings"
)

// SortFilter 页面过滤器
type SortFilter struct {
	// true:升序,false:降序
	AscFlag bool `json:"ascFlag"`
	// 排序字段
	FieldName string `json:"fieldName"`
}

func NewSortFilter(name string, ascFlag bool) *SortFilter {
	return &SortFilter{AscFlag: ascFlag, FieldName: name}
}

func (s *SortFilter) String() string {
	ss := strings.Builder{}
	fmt.Fprintf(&ss, "[%s:%t]", s.FieldName, s.AscFlag)
	return ss.String()
}

// Name return name
func (s *SortFilter) Name() string {
	return s.FieldName
}

func (s *SortFilter) AscSort() bool {
	return s.AscFlag
}
