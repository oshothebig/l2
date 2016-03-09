// config.go
package stp

import (
	"errors"
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
	BpduGuardInterval             int32
}

var StpPortConfigMap map[int32]StpPortConfig
var StpBridgeConfigMap map[int32]StpBridgeConfig

func StpPortConfigGet(pId int32) *StpPortConfig {
	c, ok := StpPortConfigMap[pId]
	if ok {
		return &c
	}
	return nil
}

func StpBrgConfigGet(bId int32) *StpBridgeConfig {
	c, ok := StpBridgeConfigMap[bId]
	if ok {
		return &c
	}
	return nil
}

func StpBrgConfigParamCheck(c *StpBridgeConfig) error {

	// Table 17-2 says the values can be 0-32768 in increments of 4096
	validStpPriorityMap := map[uint16]bool{
		4096:  true,
		8192:  true,
		16384: true,
		32768: true,
	}

	if _, ok := validStpPriorityMap[c.Dot1dStpPriority]; !ok {
		return errors.New(fmt.Sprintf("Invalid Bridge Priority %d valid values %v", c.Dot1dStpPriority, []uint16{4096, 8192, 16384, 32768}))
	}

	// valid values according to Table 17-1
	if c.Dot1dStpBridgeMaxAge < 6 ||
		c.Dot1dStpBridgeMaxAge > 40 {
		return errors.New(fmt.Sprintf("Invalid Bridge Max Age %d valid range 6.0 - 40.0", c.Dot1dStpBridgeMaxAge))
	}

	if c.Dot1dStpBridgeHelloTime < 1 ||
		c.Dot1dStpBridgeHelloTime > 2 {
		return errors.New(fmt.Sprintf("Invalid Bridge Hello Time %d valid range 1.0 - 2.0", c.Dot1dStpBridgeHelloTime))
	}

	if c.Dot1dStpBridgeForwardDelay < 3 ||
		c.Dot1dStpBridgeForwardDelay > 30 {
		return errors.New(fmt.Sprintf("Invalid Bridge Hello Time %d valid range 3.0 - 30.0", c.Dot1dStpBridgeForwardDelay))
	}

	// 1 == STP
	// 2 == RSTP
	// 3 == MSTP currently not support
	if c.Dot1dStpBridgeForceVersion != 1 &&
		c.Dot1dStpBridgeForceVersion != 2 {
		return errors.New(fmt.Sprintf("Invalid Bridge Force Version %d valid 1 (STP) 2 (RSTP)", c.Dot1dStpBridgeForceVersion))
	}

	if c.Dot1dStpBridgeTxHoldCount < 1 ||
		c.Dot1dStpBridgeTxHoldCount > 10 {
		return errors.New(fmt.Sprintf("Invalid Bridge Tx Hold Count %d valid range 1 - 10", c.Dot1dStpBridgeTxHoldCount))
	}

	// if zero is used then we will convert this to use default
	if c.Dot1dStpBridgeVlan != 0 &&
		c.Dot1dStpBridgeVlan != DEFAULT_STP_BRIDGE_VLAN {
		if c.Dot1dStpBridgeVlan < 1 ||
			c.Dot1dStpBridgeVlan > 4094 {
			return errors.New(fmt.Sprintf("Invalid Bridge Vlan %d valid range 1 - 4094", c.Dot1dStpBridgeTxHoldCount))
		}
	}
	return nil
}

func StpPortConfigParamCheck(c *StpPortConfig) error {

	var p *StpPort
	validStpPortPriorityMap := map[uint16]bool{
		0:   true,
		16:  true,
		32:  true,
		48:  true,
		64:  true,
		80:  true,
		96:  true,
		112: true,
		128: true,
		144: true,
		160: true,
		176: true,
		192: true,
		208: true,
		224: true,
		240: true,
	}

	if c.Dot1dStpBridgeIfIndex == 0 {
		return errors.New(fmt.Sprintf("Invalid Port %d Must be created against a valid bridge interface", c.Dot1dStpPort))
	}

	// Table 17-2
	if _, ok := validStpPortPriorityMap[c.Dot1dStpPortPriority]; !ok {
		return errors.New(fmt.Sprintf("Invalid Port %d Priority %d valid values %v", c.Dot1dStpPortPriority, c.Dot1dStpPortPriority, []uint16{
			0, 16, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240}))
	}

	if c.Dot1dStpPortAdminPathCost > 200000000 {
		return errors.New(fmt.Sprintf("Invalid Port %d Path Cost %d valid values 0 (AUTO) or 1 - 200,000,000", c.Dot1dStpPortPriority, c.Dot1dStpPortAdminPathCost))
	}

	if StpFindPortByIfIndex(c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex, &p) {
		if (!p.OperEdge && !c.Dot1dStpPortAdminEdgePort) &&
			p.BpduGuard {
			return errors.New(fmt.Sprintf("Invalid Port %d Bpdu Guard only available on Edge Ports", c.Dot1dStpPort))
		}

		if (p.OperEdge || c.Dot1dStpPortAdminEdgePort) &&
			p.BridgeAssurance {
			return errors.New(fmt.Sprintf("Invalid Port %d Bridge Assurance only available on non Edge Ports", c.Dot1dStpPort))
		}
	}
	return nil
}

func StpBridgeCreate(c *StpBridgeConfig) error {
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

		// lets store the configuration
		if _, ok := StpBridgeConfigMap[b.BrgIfIndex]; !ok {
			StpBridgeConfigMap[b.BrgIfIndex] = *c
		} else {
			// lets update all other bridge ports in case any of the parameters changed

		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, bridge vlan %d already exists", c.Dot1dStpBridgeVlan))
	}
	return nil
}

func StpBridgeDelete(c *StpBridgeConfig) error {
	var b *Bridge

	key := BridgeKey{
		Vlan: c.Dot1dStpBridgeVlan,
	}
	if StpFindBridgeById(key, &b) {
		DelStpBridge(b, true)
		for _, btmp := range StpBridgeConfigMap {
			if btmp.Dot1dStpBridgeVlan == c.Dot1dStpBridgeVlan {
				delete(StpBridgeConfigMap, b.BrgIfIndex)
			}
		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, bridge vlan %d does not exists", c.Dot1dStpBridgeVlan))
	}
	return nil
}

func StpPortCreate(c *StpPortConfig) error {
	var p *StpPort
	var b *Bridge
	if !StpFindPortByIfIndex(c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex, &p) {
		brgIfIndex := c.Dot1dStpBridgeIfIndex
		c.Dot1dStpBridgeIfIndex = 0
		// lets store the configuration
		if _, ok := StpPortConfigMap[c.Dot1dStpPort]; !ok {
			StpPortConfigMap[c.Dot1dStpPort] = *c
		} else {
			if *c != StpPortConfigMap[c.Dot1dStpPort] {
				// TODO failing for now will need to add code to update all other bridges that use
				// this physical port
				return errors.New(fmt.Sprintf("Error Port %d Provisioning does not agree with previously created bridge port prev[%#v] new[%#v]",
					c.Dot1dStpPort, StpPortConfigMap[c.Dot1dStpPort], *c))
			}
		}

		c.Dot1dStpBridgeIfIndex = brgIfIndex
		// nothing should happen until a birdge is assigned to the port
		if StpFindBridgeByIfIndex(c.Dot1dStpBridgeIfIndex, &b) {
			p := NewStpPort(c)
			StpPortAddToBridge(p.IfIndex, p.BrgIfIndex)
		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, port %d bridge %d already exists", c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex))
	}
	return nil
}

func StpPortDelete(c *StpPortConfig) error {
	var p *StpPort
	var b *Bridge
	if StpFindPortByIfIndex(c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex, &p) {
		if StpFindBridgeByIfIndex(p.BrgIfIndex, &b) {
			StpPortDelFromBridge(c.Dot1dStpPort, p.BrgIfIndex)
		}
		DelStpPort(p)
		foundPort := false
		for _, ptmp := range PortListTable {
			if ptmp.IfIndex == p.IfIndex {
				foundPort = true
			}
		}
		if !foundPort {
			delete(StpPortConfigMap, c.Dot1dStpPort)
		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, port %d bridge %d does not exists", c.Dot1dStpPort, c.Dot1dStpBridgeIfIndex))
	}
	return nil
}

func StpPortAddToBridge(pId int32, brgifindex int32) {
	var p *StpPort
	var b *Bridge
	if StpFindPortByIfIndex(pId, brgifindex, &p) && StpFindBridgeByIfIndex(brgifindex, &b) {
		p.BridgeId = b.BridgeIdentifier
		b.StpPorts = append(b.StpPorts, pId)
		p.BEGIN(false)

		// check all other bridge ports to see if any are AdminEdge
		isOtherBrgPortOperEdge := p.IsAdminEdgePort()
		if !p.AdminEdge && isOtherBrgPortOperEdge {
			p.BdmMachineFsm.BdmEvents <- MachineEvent{
				e:   BdmEventBeginAdminEdge,
				src: "CONFIG: AdminEgeSet",
			}
		} else if p.AdminEdge && !isOtherBrgPortOperEdge {
			for _, ptmp := range PortListTable {
				if p != ptmp {
					p.BdmMachineFsm.BdmEvents <- MachineEvent{
						e:   BdmEventBeginAdminEdge,
						src: "CONFIG: AdminEgeSet",
					}
				}
			}
		}

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

func StpPortEnable(pId int32, bId int32, enable bool) error {
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
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Port Enable", pId, bId))
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

func StpBrgPrioritySet(bId int32, priority uint16) error {
	// get bridge
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		prio := GetBridgePriorityFromBridgeId(b.BridgeIdentifier)
		if prio != priority {
			c := StpBrgConfigGet(bId)
			prevval := c.Dot1dStpPriority
			c.Dot1dStpPriority = priority
			err := StpBrgConfigParamCheck(c)
			if err == nil {
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
			} else {
				c.Dot1dStpPriority = prevval
			}
			return err
		}
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Priority", bId))
}

func StpBrgForceVersion(bId int32, version int32) error {

	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		// version 1 STP
		// version 2 RSTP
		if b.ForceVersion != version {
			c := StpBrgConfigGet(bId)
			prevval := c.Dot1dStpBridgeForceVersion
			c.Dot1dStpBridgeForceVersion = version
			err := StpBrgConfigParamCheck(c)
			if err == nil {
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
			} else {
				c.Dot1dStpBridgeForceVersion = prevval
			}
			return err
		}
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Force Version", bId))
}

func StpPortPrioritySet(pId int32, bId int32, priority uint16) error {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.Priority != priority {
			c := StpPortConfigGet(pId)
			c.Dot1dStpPortPriority = priority
			err := StpPortConfigParamCheck(c)
			if err == nil {
				p.Priority = priority
				p.Selected = false
				p.Reselect = true

				p.b.PrsMachineFsm.PrsEvents <- MachineEvent{
					e:   PrsEventReselect,
					src: "CONFIG: PortPrioritySet",
				}
			}
			return err
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Port Priority", pId, bId))
}

func StpPortPortPathCostSet(pId int32, bId int32, pathcost uint32) error {
	// TODO
	return nil
}

func StpPortAdminEdgeSet(pId int32, bId int32, adminedge bool) error {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		p.AdminEdge = adminedge
		if p.OperEdge != adminedge {
			c := StpPortConfigGet(pId)
			prevval := c.Dot1dStpPortAdminEdgePort
			c.Dot1dStpPortAdminEdgePort = adminedge
			err := StpPortConfigParamCheck(c)
			if err == nil {
				isOtherBrgPortOperEdge := p.IsAdminEdgePort()
				// if we transition from Admin Edge to non-Admin edge
				if !p.AdminEdge && !isOtherBrgPortOperEdge {
					p.BdmMachineFsm.BdmEvents <- MachineEvent{
						e:   BdmEventBeginNotAdminEdge,
						src: "CONFIG: AdminEgeSet",
					}
					for _, ptmp := range PortListTable {
						if p != ptmp &&
							p.IfIndex == ptmp.IfIndex {
							p.BdmMachineFsm.BdmEvents <- MachineEvent{
								e:   BdmEventBeginNotAdminEdge,
								src: "CONFIG: AdminEgeSet",
							}
						}
					}

				} else if p.AdminEdge && !isOtherBrgPortOperEdge {
					p.BdmMachineFsm.BdmEvents <- MachineEvent{
						e:   BdmEventBeginAdminEdge,
						src: "CONFIG: AdminEgeSet",
					}

					for _, ptmp := range PortListTable {
						if p != ptmp &&
							p.IfIndex == ptmp.IfIndex {
							p.BdmMachineFsm.BdmEvents <- MachineEvent{
								e:   BdmEventBeginAdminEdge,
								src: "CONFIG: AdminEgeSet",
							}
						}
					}
				}
			} else {
				c.Dot1dStpPortAdminEdgePort = prevval
			}
			return err
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Port Admin Edge", pId, bId))
}

func StpBrgForwardDelaySet(bId int32, fwddelay uint16) error {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.Dot1dStpBridgeForwardDelay
		c.Dot1dStpBridgeForwardDelay = fwddelay
		err := StpBrgConfigParamCheck(c)
		if err == nil {

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
		} else {
			c.Dot1dStpBridgeForwardDelay = prevval
		}
		return err
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Forwarding Delay", bId))
}

func StpBrgHelloTimeSet(bId int32, hellotime uint16) error {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.Dot1dStpBridgeHelloTime
		c.Dot1dStpBridgeHelloTime = hellotime
		err := StpBrgConfigParamCheck(c)
		if err == nil {
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
		} else {
			c.Dot1dStpBridgeHelloTime = prevval
		}
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Hello Time", bId))
}

func StpBrgMaxAgeSet(bId int32, maxage uint16) error {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.Dot1dStpBridgeMaxAge
		c.Dot1dStpBridgeMaxAge = maxage
		err := StpBrgConfigParamCheck(c)
		if err == nil {

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
		} else {
			c.Dot1dStpBridgeMaxAge = prevval
		}
		return err
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Max Age", bId))
}

func StpBrgTxHoldCountSet(bId int32, txholdcount uint16) error {
	var b *Bridge
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.Dot1dStpBridgeTxHoldCount
		c.Dot1dStpBridgeTxHoldCount = int32(txholdcount)
		err := StpBrgConfigParamCheck(c)
		if err == nil {
			b.TxHoldCount = uint64(txholdcount)
		} else {
			c.Dot1dStpBridgeTxHoldCount = prevval
		}
		return err
	}
	return nil
}

func StpPortProtocolMigrationSet(pId int32, bId int32, protocolmigration bool) error {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.Mcheck != protocolmigration && protocolmigration {
			c := StpPortConfigGet(pId)
			prevval := c.Dot1dStpPortProtocolMigration
			if protocolmigration {
				c.Dot1dStpPortProtocolMigration = int32(1)
			} else {
				c.Dot1dStpPortProtocolMigration = int32(0)
			}
			err := StpPortConfigParamCheck(c)
			if err == nil {
				p.PpmmMachineFsm.PpmmEvents <- MachineEvent{e: PpmmEventMcheck,
					src: "CONFIG: ProtocolMigrationSet",
				}
				p.Mcheck = true
			} else {
				c.Dot1dStpPortProtocolMigration = prevval
			}
			return err
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Protcol Migration", pId, bId))
}

func StpPortBridgeAssuranceSet(pId int32, bId int32, bridgeassurance bool) error {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.BridgeAssurance != bridgeassurance &&
			!p.OperEdge {
			c := StpPortConfigGet(pId)
			prevval := c.BridgeAssurance
			c.BridgeAssurance = bridgeassurance
			err := StpPortConfigParamCheck(c)
			if err == nil {
				p.BridgeAssurance = bridgeassurance
				p.BridgeAssuranceInconsistant = false
				p.BAWhileTimer.count = int32(p.b.RootTimes.HelloTime * 3)
			} else {
				c.BridgeAssurance = prevval
			}
			return err
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Bridge Assurance", pId, bId))
}
