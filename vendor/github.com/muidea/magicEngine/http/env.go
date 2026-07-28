package http

import (
	"os"
	"strconv"
	"time"
)

// Envs
const (
	Dev  string = "development"
	Prod string = "production"
	Test string = "test"
)

const serverName = "magic_engine"

type StaticOptionsKey struct{}

var Env = Dev

var Root string

var enableTrace = false
var elapseThreshold = 10 * time.Second

func setENV(env string) {
	if len(env) > 0 {
		Env = env
	}
}

func init() {
	setENV(os.Getenv("MAGICENGINE_ENV"))
	var err error
	Root, err = os.Getwd()
	if err != nil {
		panic(err)
	}

	enableTrace = Env != Prod

	elapseStr := os.Getenv("MAGICENGINE_ELAPSE")
	if elapseStr != "" {
		elapseVal, elapseErr := strconv.ParseInt(elapseStr, 10, 32)
		if elapseErr == nil {
			if elapseVal <= 0 {
				elapseVal = 1
			}
			elapseThreshold = time.Duration(elapseVal) * time.Second
		}
	}
}

func EnableTrace() bool {
	return enableTrace
}

func GetElapseThreshold() time.Duration {
	return elapseThreshold
}
