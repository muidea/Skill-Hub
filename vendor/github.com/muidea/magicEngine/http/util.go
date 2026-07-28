package http

import "fmt"

func panicInfo(info string) {
	msg := fmt.Sprintf("[%s] %s\n", serverName, info)
	panic(msg)
}
