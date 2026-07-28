package util

// ObjectSorter help object slice sort
type ObjectSorter interface {
	Less(left, right interface{}) bool
}

// NewSortHelper new SortHelper
func NewSortHelper(objList []interface{}, sorter ObjectSorter) *SortHelper {
	return &SortHelper{objList: objList, sorter: sorter}
}

// SortHelper sortHelper
type SortHelper struct {
	objList []interface{}
	sorter  ObjectSorter
}

func (s SortHelper) Len() int {
	return len(s.objList)
}

func (s SortHelper) Less(i, j int) bool {
	return s.sorter.Less(s.objList[i], s.objList[j])
}

func (s SortHelper) Swap(i, j int) {
	s.objList[i], s.objList[j] = s.objList[j], s.objList[i]
}
