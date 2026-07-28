package event

import (
	"context"
	"fmt"
	"reflect"

	cd "github.com/muidea/magicCommon/def"
)

const innerDataKey = "_innerDataKey_"
const innerValKey = "_innerValKey_"

const (
	Action = "_action_"
	Add    = "add"
	Del    = "del"
	Mod    = "mod"
	Notify = "notify"
)

type baseEvent struct {
	eventID          string
	eventSource      string
	eventDestination string
	eventLaneKey     string
	eventHeader      Values
	eventContext     context.Context
	eventData        map[string]any
	eventResult      any
}

type baseResult struct {
	resultData map[string]any
	resultErr  *cd.Error
}

func NewValues() Values {
	return map[string]any{}
}

func NewHeader() Values {
	return map[string]any{}
}

func NewEvent(id, source, destination string, header Values, data any) Event {
	return &baseEvent{
		eventID:          id,
		eventSource:      source,
		eventDestination: destination,
		eventHeader:      header,
		eventData:        map[string]any{innerDataKey: data},
	}
}

func NewEventWithContext(id, source, destination string, header Values, context context.Context, data any) Event {
	return &baseEvent{
		eventID:          id,
		eventSource:      source,
		eventDestination: destination,
		eventHeader:      header,
		eventContext:     context,
		eventData:        map[string]any{innerDataKey: data},
	}
}

func NewResult(id, source, destination string) Result {
	msg := fmt.Sprintf("illegal event, no result returned, id:%s, source:%s, destination:%s", id, source, destination)
	return &baseResult{resultErr: cd.NewError(cd.UnKnownError, msg), resultData: map[string]any{}}
}

func (s *baseEvent) ID() string {
	return s.eventID
}

func (s *baseEvent) Source() string {
	return s.eventSource
}

func (s *baseEvent) Destination() string {
	return s.eventDestination
}

func (s *baseEvent) LaneKey() string {
	if s.eventLaneKey != "" {
		return s.eventLaneKey
	}

	return s.eventDestination
}

func (s *baseEvent) Header() Values {
	if s.eventHeader == nil {
		s.eventHeader = NewHeader()
	}
	return s.eventHeader
}

func (s *baseEvent) Context() context.Context {
	if s.eventContext == nil {
		return context.Background()
	}

	return s.eventContext
}

func (s *baseEvent) BindContext(ctx context.Context) {
	s.eventContext = ctx
}

func (s *baseEvent) BindLaneKey(laneKey string) {
	s.eventLaneKey = laneKey
}

func (s *baseEvent) Data() any {
	val, ok := s.eventData[innerDataKey]
	if ok {
		return val
	}
	return nil
}

func (s *baseEvent) SetData(key string, val any) {
	s.eventData[key] = val
}

func (s *baseEvent) GetData(key string) any {
	val, ok := s.eventData[key]
	if ok {
		return val
	}

	return nil
}

func (s *baseEvent) Result(result any) {
	s.eventResult = result
}

func (s *baseEvent) Match(pattern string) bool {
	return MatchValue(pattern, s.eventID)
}

func (s *baseResult) Set(data any, err *cd.Error) {
	s.resultData[innerValKey] = data
	s.resultErr = err
}

func (s *baseResult) Error() *cd.Error {
	return s.resultErr
}

func (s *baseResult) Get() (ret any, err *cd.Error) {
	ret = s.resultData[innerValKey]
	err = s.resultErr
	return
}

func (s *baseResult) SetVal(key string, val any) {
	s.resultData[key] = val
}

func (s *baseResult) GetVal(key string) any {
	val, ok := s.resultData[key]
	if ok {
		return val
	}

	return nil
}

// GetAs 尝试把 Result.Get() 的值转换为指定类型
func GetAs[T any](r Result) (T, *cd.Error) {
	var zero T
	val, err := r.Get()
	if val == nil {
		return zero, err
	}
	v, ok := val.(T)
	if !ok {
		return zero, cd.NewError(cd.Unexpected, fmt.Sprintf(
			"invalid type: expect %v but got %v",
			reflect.TypeOf(zero), reflect.TypeOf(val),
		))
	}
	return v, err
}

// GetValAs 尝试把 Result.GetVal() 的值转换为指定类型
func GetValAs[T any](r Result, key string) (T, bool) {
	var zero T
	val := r.GetVal(key)
	if val == nil {
		return zero, false
	}
	v, ok := val.(T)
	return v, ok
}

// e.Data()
func GetAsFromEvent[T any](e Event) (T, *cd.Error) {
	var zero T
	val := e.Data()
	if val == nil {
		return zero, cd.NewError(cd.Unexpected, "event data is nil")
	}
	v, ok := val.(T)
	if !ok {
		return zero, cd.NewError(cd.Unexpected, fmt.Sprintf(
			"invalid type: expect %v but got %v",
			reflect.TypeOf(zero), reflect.TypeOf(val),
		))
	}
	return v, nil
}

// e.GetData()
func GetValAsFromEvent[T any](e Event, key string) (T, bool) {
	var zero T
	val := e.GetData(key)
	if val == nil {
		return zero, false
	}
	v, ok := val.(T)
	return v, ok
}

// e.Header()
func GetHeaderValAsFromEvent[T any](e Event, key string) (T, bool) {
	var zero T
	header := e.Header()
	if header == nil {
		return zero, false
	}
	val, ok := header[key]
	if !ok {
		return zero, false
	}
	v, ok := val.(T)
	return v, ok
}

// e.Context()
func GetContextValAsFromEvent[T any](e Event, key any) (T, bool) {
	var zero T
	ctx := e.Context()
	if ctx == nil {
		return zero, false
	}
	val := ctx.Value(key)
	if val == nil {
		return zero, false
	}
	v, ok := val.(T)
	return v, ok
}
