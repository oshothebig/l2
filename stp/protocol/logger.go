// logger.go
package stp

import (
	"fmt"
	"strings"
)

func StpLogger(t string, msg string) {

	switch t {
	case "INFO":
		gLogger.Info(msg)
	case "DEBUG":
		gLogger.Debug(msg)
	case "ERROR":
		gLogger.Err(msg)
	case "WARNING":
		gLogger.Warning(msg)
	}
}

func StpLoggerInfo(msg string) {
	StpLogger("INFO", msg)
}

func StpMachineLogger(t string, m string, p int32, b int32, msg string) {
	StpLogger(t, strings.Join([]string{m, fmt.Sprintf("port %d", p), fmt.Sprintf("brg %d", b), msg}, ":"))
}
