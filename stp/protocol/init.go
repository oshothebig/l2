// init.go
package stp

import (
	"log/syslog"
	"sync"
)

var gLogger *syslog.Writer

func init() {
	PortConfigMap = make(map[int32]portConfig)
	PortMapTable = make(map[PortMapKey]*StpPort, 0)
	BridgeMapTable = make(map[BridgeKey]*Bridge, 0)
	StpPortConfigMap = make(map[int32]StpPortConfig, 0)
	StpBridgeConfigMap = make(map[int32]StpBridgeConfig, 0)

	asicdmutex = &sync.Mutex{}

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
	PstMachineStrStateMapInit()

	// create the logger used by this module
	gLogger, _ = syslog.New(syslog.LOG_NOTICE|syslog.LOG_INFO|syslog.LOG_DAEMON, "STP")
}
