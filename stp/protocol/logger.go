// logger.go
package stp

import (
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

func StpMachineLogger(t string, m string, msg string) {
	StpLogger(t, strings.Join([]string{m, msg}, ":"))
}
