package common

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/foundation/system"
	"github.com/muidea/magicCommon/task"
)

type InvokeFunc func() *cd.Error

type Plugin interface {
	ID() string
	Run(context.Context) *cd.Error
}

type Weighted interface {
	Weight() int
}

type Setupper interface {
	Setup(context.Context, event.Hub, task.BackgroundRoutine) *cd.Error
}

type Teardowner interface {
	Teardown(context.Context)
}

type identifier interface {
	ID() string
}

type PluginMgr struct {
	typeName   string
	entityList []any
	mu         sync.RWMutex
}

func NewPluginMgr(typeName string) *PluginMgr {
	ptr := &PluginMgr{
		typeName:   typeName,
		entityList: []any{},
	}
	return ptr
}

func (s *PluginMgr) getWeight(ptr any) (weight int, err error) {
	if typed, ok := ptr.(Weighted); ok {
		defer func() {
			if info := recover(); info != nil {
				slog.Error("panic in getWeight", "recover", info)
				weight = DefaultWeight
				err = fmt.Errorf("panic invoking %s", weightTag)
			}
		}()
		return typed.Weight(), nil
	}

	vVal := reflect.ValueOf(ptr)
	funcVal := vVal.MethodByName(weightTag)
	if !funcVal.IsValid() {
		return DefaultWeight, nil
	}

	defer func() {
		if info := recover(); info != nil {
			slog.Error("panic in getWeight", "recover", info)
			weight = DefaultWeight
			err = fmt.Errorf("panic invoking %s", weightTag)
		}
	}()

	param := make([]reflect.Value, 0)
	values := funcVal.Call(param)
	if len(values) == 0 {
		return DefaultWeight, nil
	}

	if funcVal.Type().Out(0).String() != "int" {
		return DefaultWeight, fmt.Errorf("weight method must return int, got %s", funcVal.Type().Out(0).String())
	}

	weight = int(values[0].Int())
	return weight, nil
}

func (s *PluginMgr) getID(ptr any) (id string, err error) {
	if typed, ok := ptr.(identifier); ok {
		defer func() {
			if info := recover(); info != nil {
				slog.Error("panic in getID", "recover", info)
				id = ""
				err = fmt.Errorf("panic invoking %s", idTag)
			}
		}()
		return typed.ID(), nil
	}

	vVal := reflect.ValueOf(ptr)
	funcVal := vVal.MethodByName(idTag)
	if !funcVal.IsValid() {
		return "", fmt.Errorf("method %s not found", idTag)
	}

	defer func() {
		if info := recover(); info != nil {
			slog.Error("panic in getID", "recover", info)
			id = ""
			err = fmt.Errorf("panic invoking %s", idTag)
		}
	}()

	param := make([]reflect.Value, 0)
	values := funcVal.Call(param)
	if len(values) == 0 {
		return "", fmt.Errorf("method %s returned no value", idTag)
	}

	if funcVal.Type().Out(0).String() != "string" {
		return "", fmt.Errorf("method %s must return string, got %s", idTag, funcVal.Type().Out(0).String())
	}

	id = values[0].String()
	return id, nil
}

func (s *PluginMgr) validPlugin(ptr any) error {
	if ptr == nil {
		return fmt.Errorf("plugin is nil")
	}

	vType := reflect.TypeOf(ptr)
	if vType == nil {
		return fmt.Errorf("plugin type is nil")
	}
	if vType.Kind() != reflect.Ptr {
		return fmt.Errorf("must be a pointer")
	}
	if reflect.ValueOf(ptr).IsNil() {
		return fmt.Errorf("plugin pointer is nil")
	}

	if err := validateIDMethod(vType); err != nil {
		return err
	}
	if err := validateRunMethod(vType); err != nil {
		return err
	}
	if err := validateWeightMethod(vType); err != nil {
		return err
	}
	if err := validateSetupMethod(vType); err != nil {
		return err
	}
	if err := validateTeardownMethod(vType); err != nil {
		return err
	}

	return nil
}

func validateIDMethod(vType reflect.Type) error {
	method, ok := vType.MethodByName(idTag)
	if !ok {
		return fmt.Errorf("invalid plugin value: missing required method %s", idTag)
	}

	methodType := method.Type
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 || methodType.Out(0).Kind() != reflect.String {
		return fmt.Errorf("method %s must have signature %s() string", idTag, idTag)
	}
	return nil
}

func validateRunMethod(vType reflect.Type) error {
	method, ok := vType.MethodByName(runTag)
	if !ok {
		return fmt.Errorf("invalid plugin value: missing required method %s", runTag)
	}

	methodType := method.Type
	if methodType.NumIn() != 2 || !methodType.In(1).AssignableTo(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return fmt.Errorf("method %s must accept context.Context", runTag)
	}
	return validateOptionalCDErrorReturn(runTag, methodType)
}

func validateWeightMethod(vType reflect.Type) error {
	method, ok := vType.MethodByName(weightTag)
	if !ok {
		return nil
	}

	methodType := method.Type
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 || methodType.Out(0).Kind() != reflect.Int {
		return fmt.Errorf("method %s must have signature %s() int", weightTag, weightTag)
	}
	return nil
}

func validateSetupMethod(vType reflect.Type) error {
	method, ok := vType.MethodByName(setupTag)
	if !ok {
		return nil
	}

	methodType := method.Type
	if methodType.NumIn() != 4 {
		return fmt.Errorf("method %s must accept context.Context, event.Hub, task.BackgroundRoutine", setupTag)
	}
	if !methodType.In(1).AssignableTo(reflect.TypeOf((*context.Context)(nil)).Elem()) ||
		!methodType.In(2).AssignableTo(reflect.TypeOf((*event.Hub)(nil)).Elem()) ||
		!methodType.In(3).AssignableTo(reflect.TypeOf((*task.BackgroundRoutine)(nil)).Elem()) {
		return fmt.Errorf("method %s must accept context.Context, event.Hub, task.BackgroundRoutine", setupTag)
	}
	return validateOptionalCDErrorReturn(setupTag, methodType)
}

func validateTeardownMethod(vType reflect.Type) error {
	method, ok := vType.MethodByName(teardownTag)
	if !ok {
		return nil
	}

	methodType := method.Type
	if methodType.NumIn() != 2 || !methodType.In(1).AssignableTo(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return fmt.Errorf("method %s must accept context.Context", teardownTag)
	}
	if methodType.NumOut() != 0 {
		return fmt.Errorf("method %s must not return values", teardownTag)
	}
	return nil
}

func validateOptionalCDErrorReturn(methodName string, methodType reflect.Type) error {
	if methodType.NumOut() == 0 {
		return nil
	}
	if methodType.NumOut() == 1 && methodType.Out(0).AssignableTo(reflect.TypeOf((*cd.Error)(nil))) {
		return nil
	}
	return fmt.Errorf("method %s must return no values or *def.Error", methodName)
}

func (s *PluginMgr) Register(ptr any) error {
	if err := s.validPlugin(ptr); err != nil {
		return err
	}

	curWeight, err := s.getWeight(ptr)
	if err != nil {
		return err
	}
	curID, err := s.getID(ptr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newList := []any{}
	if len(s.entityList) == 0 {
		newList = append(newList, ptr)
	} else {
		ok := false
		for idx, val := range s.entityList {
			existID, err := s.getID(val)
			if err != nil {
				return err
			}
			if existID == curID {
				return fmt.Errorf("duplicate plugin id: %s", curID)
			}

			preWeight, err := s.getWeight(val)
			if err != nil {
				return err
			}
			if preWeight <= curWeight {
				newList = append(newList, val)
				continue
			}

			ok = true
			newList = append(newList, ptr)
			newList = append(newList, s.entityList[idx:]...)
			break
		}

		if !ok {
			newList = append(newList, ptr)
		}
	}

	s.entityList = newList
	return nil
}

func (s *PluginMgr) GetEntity(id string) (ret any, err *cd.Error) {
	s.mu.RLock()
	entityList := append([]any(nil), s.entityList...)
	s.mu.RUnlock()

	for _, val := range entityList {
		idVal, idErr := s.getID(val)
		if idErr != nil {
			err = cd.NewError(cd.Unexpected, fmt.Sprintf("get ID failed: %v", idErr))
			return
		}
		if idVal == id {
			ret = val
			return
		}
	}

	err = cd.NewError(cd.Unexpected, fmt.Sprintf("%s not found", s.typeName))
	return
}

func (s *PluginMgr) Setup(ctx context.Context, eventHub event.Hub, backgroundRoutine task.BackgroundRoutine) (err *cd.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.RLock()
	entityList := append([]any(nil), s.entityList...)
	s.mu.RUnlock()

	setupList := []any{}
	for _, val := range entityList {
		setupCalled := false
		err, setupCalled = s.invokeSetup(val, ctx, eventHub, backgroundRoutine)
		if err != nil {
			idVal, idErr := s.getID(val)
			if idErr != nil {
				slog.Error("invoke setup failed, get ID error", "type", s.typeName, "error", idErr)
			} else {
				slog.Error("invoke setup failed", "type", s.typeName, "id", idVal, "error", err)
			}
			s.rollbackSetup(ctx, setupList)
			return
		}
		if setupCalled {
			setupList = append(setupList, val)
		}
	}

	return
}

func (s *PluginMgr) Run(ctx context.Context) (err *cd.Error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.RLock()
	entityList := append([]any(nil), s.entityList...)
	s.mu.RUnlock()

	for _, val := range entityList {
		err = s.invokeRun(val, ctx)
		if err != nil {
			idVal, idErr := s.getID(val)
			if idErr != nil {
				slog.Error("invoke run failed, get ID error", "type", s.typeName, "error", idErr)
			} else {
				slog.Error("invoke run failed", "type", s.typeName, "id", idVal, "error", err)
			}
			return
		}

		//slog.Info("invoke run success", "type", s.typeName, "id", idVal)
	}

	return
}

func (s *PluginMgr) Teardown(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.RLock()
	entityList := append([]any(nil), s.entityList...)
	s.mu.RUnlock()

	totalSize := len(entityList)
	for idx := range entityList {
		val := entityList[totalSize-idx-1]
		err := s.invokeTeardown(val, ctx)
		if err != nil {
			idVal, idErr := s.getID(val)
			if idErr != nil {
				slog.Error("invoke teardown failed, get ID error", "type", s.typeName, "error", idErr)
			} else {
				slog.Error("invoke teardown failed", "type", s.typeName, "id", idVal, "error", err)
			}
		}

		//slog.Info("invoke teardown success", "type", s.typeName, "id", idVal)
	}
}

func (s *PluginMgr) invokeSetup(ptr any, ctx context.Context, eventHub event.Hub, backgroundRoutine task.BackgroundRoutine) (*cd.Error, bool) {
	if typed, ok := ptr.(Setupper); ok {
		return typed.Setup(ctx, eventHub, backgroundRoutine), true
	}

	err := system.InvokeEntityFunc(ptr, setupTag, ctx, eventHub, backgroundRoutine)
	if err != nil && err.Code == cd.NotFound {
		return nil, false
	}
	return err, true
}

func (s *PluginMgr) invokeRun(ptr any, ctx context.Context) *cd.Error {
	if typed, ok := ptr.(Plugin); ok {
		return typed.Run(ctx)
	}

	err := system.InvokeEntityFunc(ptr, runTag, ctx)
	if err != nil && err.Code == cd.NotFound {
		return cd.NewError(cd.Unexpected, fmt.Sprintf("method %s not found", runTag))
	}
	return err
}

func (s *PluginMgr) invokeTeardown(ptr any, ctx context.Context) *cd.Error {
	if typed, ok := ptr.(Teardowner); ok {
		typed.Teardown(ctx)
		return nil
	}

	err := system.InvokeEntityFunc(ptr, teardownTag, ctx)
	if err != nil && err.Code == cd.NotFound {
		return nil
	}
	return err
}

func (s *PluginMgr) rollbackSetup(ctx context.Context, setupList []any) {
	for idx := len(setupList) - 1; idx >= 0; idx-- {
		val := setupList[idx]
		err := s.invokeTeardown(val, ctx)
		if err != nil {
			idVal, idErr := s.getID(val)
			if idErr != nil {
				slog.Error("rollback teardown failed, get ID error", "type", s.typeName, "error", idErr)
			} else {
				slog.Error("rollback teardown failed", "type", s.typeName, "id", idVal, "error", err)
			}
		}
	}
}
