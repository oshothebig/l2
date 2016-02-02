// init.go
package stp

import (
	"log/syslog"
)

var gLogger *syslog.Writer

func init() {
	PortConfigMap = make(map[int32]portConfig)
	PortMapTable = make(map[int32]*StpPort, 0)
	BridgeMapTable = make(map[BridgeId]*Bridge, 0)

	// Init the state string maps
	TimerTypeStrStateMapInit()
	PtmMachineStrStateMapInit()
	PrxmMachineStrStateMapInit()
	PrsMachineStrStateMapInit()
	PrtMachineStrStateMapInit()
	BdmMachineStrStateMapInit()
	PimMachineStrStateMapInit()
	PpmmMachineStrStateMapInit()
	TcMachineStrStateMapInit()
	PtxmMachineStrStateMapInit()

	// create the logger used by this module
	gLogger, _ = syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "STP")
}
