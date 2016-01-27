// config.go
package stp

import ()

type StpBridgeConfig struct {
	Dot1dBridgeAddressKey      string `SNAPROUTE: KEY`
	Dot1dStpPriorityKey        int32  `SNAPROUTE: KEY`
	Dot1dStpBridgeMaxAge       int32
	Dot1dStpBridgeHelloTime    int32
	Dot1dStpBridgeForwardDelay int32
	Dot1dStpBridgeForceVersion int32
	Dot1dStpBridgeTxHoldCount  int32
}

type StpPortConfig struct {
	Dot1dStpPortKey               int32 `SNAPROUTE: KEY`
	Dot1dStpPortPriority          int32
	Dot1dStpPortEnable            bool
	Dot1dStpPortPathCost          int32
	Dot1dStpPortPathCost32        int32
	Dot1dStpPortProtocolMigration int32
	Dot1dStpPortAdminPointToPoint int32
	Dot1dStpPortAdminEdgePort     int32
	Dot1dStpPortAdminPathCost     int32
}

func StpPortCreate(config *StpPortConfig) {
	var p *StpPort
	if !StpFindPortById(config.Dot1dStpPortKey, &p) {
		p := NewStpPort(config)
		p.BEGIN(false)
	}
}
