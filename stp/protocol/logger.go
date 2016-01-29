// logger.go
package stp

import ()

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
