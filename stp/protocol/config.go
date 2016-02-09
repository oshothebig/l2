// config.go
package stp

import (
	"fmt"
	"net"
)

type StpBridgeConfig struct {
	Dot1dBridgeAddressKey      string `SNAPROUTE: KEY`
	Dot1dStpPriorityKey        uint16 `SNAPROUTE: KEY`
	Dot1dStpBridgeMaxAge       uint16
	Dot1dStpBridgeHelloTime    uint16
	Dot1dStpBridgeForwardDelay uint16
	Dot1dStpBridgeForceVersion int32
	Dot1dStpBridgeTxHoldCount  int32
}

type StpPortConfig struct {
	Dot1dStpPortKey               int32 `SNAPROUTE: KEY`
	Dot1dStpPortPriority          uint16
	Dot1dStpPortEnable            bool
	Dot1dStpPortPathCost          int32
	Dot1dStpPortPathCost32        int32
	Dot1dStpPortProtocolMigration int32
	Dot1dStpPortAdminPointToPoint int32
	Dot1dStpPortAdminEdgePort     int32
	Dot1dStpPortAdminPathCost     int32
	Dot1dStpBridgeId              BridgeId
}

func StpBridgeCreate(c *StpBridgeConfig) {
	var b *Bridge
	netAddr, _ := net.ParseMAC(c.Dot1dBridgeAddressKey)
	addr := [6]uint8{netAddr[0], netAddr[1], netAddr[2], netAddr[3], netAddr[4], netAddr[5]}
	bridgeId := CreateBridgeId(addr, c.Dot1dStpPriorityKey)
	if !StpFindBridgeById(bridgeId, &b) {
		b = NewStpBridge(c)
		b.BEGIN(false)
	}
}

func StpBridgeDelete(c *StpBridgeConfig) {
	var b *Bridge
	netAddr, _ := net.ParseMAC(c.Dot1dBridgeAddressKey)
	addr := [6]uint8{netAddr[0], netAddr[1], netAddr[2], netAddr[3], netAddr[4], netAddr[5]}
	bridgeId := CreateBridgeId(addr, c.Dot1dStpPriorityKey)
	if StpFindBridgeById(bridgeId, &b) {
		DelStpBridge(b, true)
	}
}

func StpPortCreate(c *StpPortConfig) {
	var p *StpPort
	var b *Bridge
	if !StpFindPortById(c.Dot1dStpPortKey, &p) {
		p := NewStpPort(c)
		// nothing should happen until a birdge is assigned to the port
		if StpFindBridgeById(p.BridgeId, &b) {
			p.BEGIN(false)
		}
	}
}

func StpPortAddToBridge(pId int32, bridgeId BridgeId) {
	var p *StpPort
	var b *Bridge
	if StpFindPortById(pId, &p) && StpFindBridgeById(bridgeId, &b) {
		p.BridgeId = b.BridgeIdentifier
		p.BEGIN(false)
	} else {
		StpLogger("ERROR", fmt.Sprintf("ERROR did not find bridge[%#v] or port[%d]", bridgeId, pId))
	}
}
