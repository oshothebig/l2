// init.go
package stp

import (
	"log/syslog"
)

var gLogger *syslog.Writer

func init() {
	PortConfigMap = make(map[int32]portConfig)
	TimerTypeStrStateMapInit()
	PtmMachineStrStateMapInit()
	PrxmMachineStrStateMapInit()

	gLogger, _ = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "STP")
}
