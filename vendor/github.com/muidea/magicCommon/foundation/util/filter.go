package util

// Filter value filter
type Filter interface {
	Filter(val interface{}) bool
}
