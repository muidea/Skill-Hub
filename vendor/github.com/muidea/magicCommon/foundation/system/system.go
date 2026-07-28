package system

import (
	"fmt"
	"reflect"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/foundation/util"
)

func InvokeEntityFunc(entityVal interface{}, funcName string, params ...interface{}) (err *cd.Error) {
	if entityVal == nil {
		return cd.NewError(cd.IllegalParam, "entityVal is nil")
	}

	vVal := reflect.ValueOf(entityVal)
	if vVal.Kind() == reflect.Ptr && vVal.IsNil() {
		return cd.NewError(cd.IllegalParam, "entityVal is nil")
	}

	funcVal := vVal.MethodByName(funcName)
	if !isValidMethod(funcVal) {
		return cd.NewError(cd.NotFound, fmt.Sprintf("no such method:%s", funcName))
	}

	defer func() {
		if errInfo := recover(); errInfo != nil {
			err = cd.NewError(cd.Unexpected, fmt.Sprintf("recover! invoke %s unexpected, err:%v\nstack:\n%s", funcName, errInfo, util.GetStack(3)))
		}
	}()

	param, err := prepareParams(funcVal, params)
	if err != nil {
		return err
	}

	rVals := funcVal.Call(param)
	if len(rVals) == 0 {
		return nil
	}

	errVal, ok := rVals[0].Interface().(*cd.Error)
	if !ok {
		return cd.NewError(cd.Unexpected, "invoke method return illegal result")
	}

	return errVal
}

func isValidMethod(funcVal reflect.Value) bool {
	return funcVal.IsValid() && !funcVal.IsZero()
}

func prepareParams(funcVal reflect.Value, params []interface{}) ([]reflect.Value, *cd.Error) {
	funcType := funcVal.Type()
	inNum := funcType.NumIn()
	if inNum == 0 {
		return nil, nil
	}

	if len(params) != inNum {
		return nil, cd.NewError(cd.IllegalParam,
			fmt.Sprintf("param count mismatch, expect:%d, actual:%d", inNum, len(params)))
	}

	param := make([]reflect.Value, inNum)
	for idx := 0; idx < inNum; idx++ {
		var err *cd.Error
		param[idx], err = convertParam(params[idx], funcType.In(idx))
		if err != nil {
			return nil, err
		}
	}

	return param, nil
}

func convertParam(val interface{}, expectedType reflect.Type) (reflect.Value, *cd.Error) {
	if val == nil {
		switch expectedType.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Slice,
			reflect.Map, reflect.Chan, reflect.Func, reflect.UnsafePointer:
			return reflect.Zero(expectedType), nil
		default:
			return reflect.Value{}, cd.NewError(cd.IllegalParam,
				fmt.Sprintf("nil cannot convert to type:%s", expectedType))
		}
	}

	rVal := reflect.ValueOf(val)
	if rVal.Kind() == reflect.Interface {
		rVal = rVal.Elem()
	}

	if rVal.Type().AssignableTo(expectedType) {
		return rVal, nil
	}

	if rVal.Type().ConvertibleTo(expectedType) {
		return rVal.Convert(expectedType), nil
	}

	return reflect.Value{}, cd.NewError(cd.IllegalParam,
		fmt.Sprintf("type mismatch, expect:%s, actual:%s",
			expectedType.String(), rVal.Type().String()))
}
