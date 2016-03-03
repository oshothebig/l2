// config.go
package stp

import (
	"fmt"
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
	BridgeAssurance               bool
	BpduGuard                     bool
}

func StpBridgeCreate(c *StpBridgeConfig) {
	var b *Bridge
	tmpaddr := c.Dot1dBridgeAddress
	if tmpaddr == "" {
		tmpaddr = "00:AA:AA:BB:BB:DD"
	}

	key := BridgeKey{
		Vlan: c.Dot1dStpBridgeVlan,
	}

	if !StpFindBridgeById(key, &b) {
		b = NewStpBridge(c)
		b.BEGIN(false)
	}
}

func StpBridgeDelete(c *StpBridgeConfig) {
	var b *Bridge

	key := BridgeKey{
		Vlan: c.Dot1dStpBridgeVlan,
	}
	if StpFindBridgeById(key, &b) {
		DelStpBridge(b, true)
	}
}

func StpPortCreate(c *StpPortConfig) {
	var p *StpPort
	var b *Bridge
	if !StpFindPortByIfIndex(c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex, &p) {
		p := NewStpPort(c)
		// nothing should happen until a birdge is assigned to the port
		if StpFindBridgeByIfIndex(p.BrgIfIndex, &b) {
			StpPortAddToBridge(p.IfIndex, p.BrgIfIndex)
		}
	}
}

func StpPortDelete(c *StpPortConfig) {
	var p *StpPort
	var b *Bridge
	if StpFindPortByIfIndex(c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex, &p) {
		if StpFindBridgeByIfIndex(p.BrgIfIndex, &b) {
			StpPortDelFromBridge(c.Dot1dStpPort, p.BrgIfIndex)
		}
		DelStpPort(p)
	}

}

func StpPortAddToBridge(pId int32, brgifindex int32) {
	var p *StpPort
	var b *Bridge
	if StpFindPortByIfIndex(pId, brgifindex, &p) && StpFindBridgeByIfIndex(brgifindex, &b) {
		p.BridgeId = b.BridgeIdentifier
		b.StpPorts = append(b.StpPorts, pId)
		p.BEGIN(false)
	} else {
		StpLogger("ERROR", fmt.Sprintf("ERROR did not find bridge[%#v] or port[%d]", brgifindex, pId))
	}
}

func StpPortDelFromBridge(pId int32, brgifindex int32) {
	var p *StpPort
	var b *Bridge
	if StpFindPortByIfIndex(pId, brgifindex, &p) && StpFindBridgeByIfIndex(brgifindex, &b) {
		// lets disable the port before we remove it so that way
		// other ports can trigger tc event
		StpPortLinkDown(pId)
		// detach the port from the bridge stp port list
		for idx, ifindex := range b.StpPorts {
			if ifindex == p.IfIndex {
				b.StpPorts = append(b.StpPorts[:idx], b.StpPorts[idx+1:]...)
			}
		}
	} else {
		StpLogger("ERROR", fmt.Sprintf("ERROR did not find bridge[%#v] or port[%d]", brgifindex, pId))
	}
}

func StpPortEnable(pId int32, bId int32, enable bool) {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.AdminPortEnabled != enable {
			if p.AdminPortEnabled {
				if p.PortEnabled {
					defer p.NotifyPortEnabled("CONFIG: ", p.PortEnabled, false)
					p.PortEnabled = false
				}
			} else {
				if asicdGetPortLinkStatus(pId) {
					defer p.NotifyPortEnabled("CONFIG: ", p.PortEnabled, true)
					p.PortEnabled = true
				}
			}
			p.AdminPortEnabled = enable
		}
	}
}

func StpPortLinkUp(pId int32) {
	for _, p := range PortListTable {
		if p.IfIndex == pId {
			if p.AdminPortEnabled {
				defer p.NotifyPortEnabled("LINK EVENT", p.PortEnabled, true)
				p.PortEnabled = true
			}
		}
	}
}

func StpPortLinkDown(pId int32) {
	for _, p := range PortListTable {
		if p.IfIndex == pId {
			defer p.NotifyPortEnabled("LINK EVENT", p.PortEnabled, false)
			p.PortEnabled = false
		}
	}
}

func StpBrgPrioritySet(bId int32, priority uint16) {
	// get bridge
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		prio := GetBridgePriorityFromBridgeId(b.BridgeIdentifier)
		if prio != priority {
			addr := GetBridgeAddrFromBridgeId(b.BridgeIdentifier)
			vlan := GetBridgeVlanFromBridgeId(b.BridgeIdentifier)
			b.BridgeIdentifier = CreateBridgeId(addr, priority, vlan)
			b.BridgePriority.DesignatedBridgeId = b.BridgeIdentifier

			for _, pId := range b.StpPorts {
				if StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
					p.Selected = false
					p.Reselect = true
				}
			}
			b.PrsMachineFsm.PrsEvents <- MachineEvent{
				e:   PrsEventReselect,
				src: "CONFIG: BrgPrioritySet",
			}
		}
	}
}

func StpBrgForceVersion(bId int32, version int32) {

	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		// version 1 STP
		// version 2 RSTP
		if b.ForceVersion != version {
			b.ForceVersion = version
			for _, pId := range b.StpPorts {
				if StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
					if b.ForceVersion == 1 {
						p.RstpVersion = false
					} else {
						p.RstpVersion = true
					}
					p.BEGIN(true)
				}
			}
		}
	}
}

func StpPortPrioritySet(pId int32, bId int32, priority uint16) {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.Priority != priority {
			p.Priority = priority
			p.Selected = false
			p.Reselect = true

			p.b.PrsMachineFsm.PrsEvents <- MachineEvent{
				e:   PrsEventReselect,
				src: "CONFIG: PortPrioritySet",
			}
		}
	}

}

func StpPortPortPathCostSet(pId int32, bId int32, pathcost uint32) {
	// TODO
}

func StpPortAdminEdgeSet(pId int32, bId int32, adminedge bool) {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.AdminEdge != adminedge {
			p.AdminEdge = adminedge

			if !p.AdminEdge {
				p.BdmMachineFsm.BdmEvents <- MachineEvent{
					e:   BdmEventBeginNotAdminEdge,
					src: "CONFIG: AdminEgeSet",
				}
			} else {
				p.BdmMachineFsm.BdmEvents <- MachineEvent{
					e:   BdmEventBeginAdminEdge,
					src: "CONFIG: AdminEgeSet",
				}
			}
		}
	}
}

func StpBrgForwardDelaySet(bId int32, fwddelay uint16) {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		b.BridgeTimes.ForwardingDelay = fwddelay

		// if we are root lets update the port times
		if b.RootPortId == 0 {
			b.RootTimes.ForwardingDelay = fwddelay
			for _, pId := range b.StpPorts {
				if StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
					p.PortTimes.ForwardingDelay = b.RootTimes.ForwardingDelay
				}
			}
		}
	}
}

func StpBrgHelloTimeSet(bId int32, hellotime uint16) {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		b.BridgeTimes.HelloTime = hellotime

		// if we are root lets update the port times
		if b.RootPortId == 0 {
			b.RootTimes.HelloTime = hellotime
			for _, pId := range b.StpPorts {
				if StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
					p.PortTimes.HelloTime = b.RootTimes.HelloTime
				}
			}
		}
	}
}

func StpBrgMaxAgeSet(bId int32, maxage uint16) {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		b.BridgeTimes.MaxAge = maxage

		// if we are root lets update the port times
		if b.RootPortId == 0 {
			b.RootTimes.MaxAge = maxage
			for _, pId := range b.StpPorts {
				if StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
					p.PortTimes.MaxAge = b.RootTimes.MaxAge
				}
			}
		}
	}
}

func StpBrgTxHoldCountSet(bId int32, txholdcount uint16) {
	var b *Bridge
	if StpFindBridgeByIfIndex(bId, &b) {
		b.TxHoldCount = uint64(txholdcount)
	}
}

func StpPortProtocolMigrationSet(pId int32, bId int32, protocolmigration bool) {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.Mcheck != protocolmigration && protocolmigration {
			p.PpmmMachineFsm.PpmmEvents <- MachineEvent{e: PpmmEventMcheck,
				src: "CONFIG: ProtocolMigrationSet",
			}
			p.Mcheck = true
		}
	}
}

func StpPortBridgeAssuranceSet(pId int32, bId int32, bridgeassurance bool) {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.BridgeAssurance != bridgeassurance &&
			!p.OperEdge {
			p.BridgeAssurance = bridgeassurance
			p.BridgeAssuranceInconsistant = false
			p.BAWhileTimer.count = int32(p.b.RootTimes.HelloTime * 3)
		}
	}
}
