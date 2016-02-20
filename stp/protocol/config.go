// config.go
package stp

import (
	"fmt"
	"net"
)

type StpBridgeConfig struct {
	Dot1dBridgeAddress         string
	Dot1dStpPriority           uint16
	Dot1dStpBridgeMaxAge       uint16
	Dot1dStpBridgeHelloTime    uint16
	Dot1dStpBridgeForwardDelay uint16
	Dot1dStpBridgeForceVersion int32
	Dot1dStpBridgeTxHoldCount  int32
	Dot1dStpBridgeVlan         uint16
}

type StpPortConfig struct {
	Dot1dStpPort                  int32
	Dot1dStpPortPriority          uint16
	Dot1dStpPortEnable            bool
	Dot1dStpPortPathCost          int32
	Dot1dStpPortProtocolMigration int32
	Dot1dStpPortAdminPointToPoint int32
	Dot1dStpPortAdminEdgePort     bool
	Dot1dStpPortAdminPathCost     int32
	Dot1dStpBridgeIfIndex         int32
}

func StpBridgeCreate(c *StpBridgeConfig) {
	var b *Bridge
	tmpaddr := c.Dot1dBridgeAddress
	if tmpaddr == "" {
		tmpaddr = "00:AA:AA:BB:BB:DD"
	}
	netAddr, _ := net.ParseMAC(tmpaddr)
	addr := [6]uint8{netAddr[0], netAddr[1], netAddr[2], netAddr[3], netAddr[4], netAddr[5]}
	bridgeId := CreateBridgeId(addr, c.Dot1dStpPriority)
	if !StpFindBridgeById(bridgeId, &b) {
		b = NewStpBridge(c)
		b.BEGIN(false)
	}
}

func StpBridgeDelete(c *StpBridgeConfig) {
	var b *Bridge
	netAddr, _ := net.ParseMAC(c.Dot1dBridgeAddress)
	addr := [6]uint8{netAddr[0], netAddr[1], netAddr[2], netAddr[3], netAddr[4], netAddr[5]}
	bridgeId := CreateBridgeId(addr, c.Dot1dStpPriority)
	if StpFindBridgeById(bridgeId, &b) {
		DelStpBridge(b, true)
	}
}

func StpPortCreate(c *StpPortConfig) {
	var p *StpPort
	var b *Bridge
	if !StpFindPortById(c.Dot1dStpPort, &p) {
		p := NewStpPort(c)
		// nothing should happen until a birdge is assigned to the port
		if StpFindBridgeByIfIndex(p.BrgIfIndex, &b) {
			StpPortAddToBridge(p.IfIndex, p.BrgIfIndex)
		}
	}
}

func StpPortAddToBridge(pId int32, brgifindex int32) {
	var p *StpPort
	var b *Bridge
	if StpFindPortById(pId, &p) && StpFindBridgeByIfIndex(brgifindex, &b) {
		p.BridgeId = b.BridgeIdentifier
		b.StpPorts = append(b.StpPorts, pId)
		p.BEGIN(false)
	} else {
		StpLogger("ERROR", fmt.Sprintf("ERROR did not find bridge[%#v] or port[%d]", brgifindex, pId))
	}
}

func StpPortLinkUp(pId int32) {
	var p *StpPort
	if StpFindPortById(pId, &p) {
		if p.AdminPortEnabled {
			defer p.NotifyPortEnabled("LINK EVENT", p.PortEnabled, true)
			p.PortEnabled = true
		}
	}
}

func StpPortLinkDown(pId int32) {
	var p *StpPort
	if StpFindPortById(pId, &p) {
		defer p.NotifyPortEnabled("LINK EVENT", p.PortEnabled, false)
		p.PortEnabled = false
	}

}
