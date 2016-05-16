Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
// config.go
package stp

import (
	"errors"
	"fmt"
)

type StpBridgeConfig struct {
	Address      string
	Priority     uint16
	MaxAge       uint16
	HelloTime    uint16
	ForwardDelay uint16
	ForceVersion int32
	TxHoldCount  int32
	Vlan         uint16
}

type StpPortConfig struct {
	IfIndex           int32
	Priority          uint16
	Enable            bool
	PathCost          int32
	ProtocolMigration int32
	AdminPointToPoint int32
	AdminEdgePort     bool
	AdminPathCost     int32
	BrgIfIndex        int32
	BridgeAssurance   bool
	BpduGuard         bool
	BpduGuardInterval int32
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

func StpPortConfigSave(c *StpPortConfig, update bool) error {
	if _, ok := StpPortConfigMap[c.IfIndex]; !ok {
		StpPortConfigMap[c.IfIndex] = *c
	} else {
		if !update && *c != StpPortConfigMap[c.IfIndex] {
			// TODO failing for now will need to add code to update all other bridges that use
			// this physical port
			return errors.New(fmt.Sprintf("Error Port %d Provisioning does not agree with previously created bridge port prev[%#v] new[%#v]",
				c.IfIndex, StpPortConfigMap[c.IfIndex], *c))
		}
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

	if _, ok := validStpPriorityMap[c.Priority]; !ok {
		return errors.New(fmt.Sprintf("Invalid Bridge Priority %d valid values %v", c.Priority, []uint16{4096, 8192, 16384, 32768}))
	}

	// valid values according to Table 17-1
	if c.MaxAge < 6 ||
		c.MaxAge > 40 {
		return errors.New(fmt.Sprintf("Invalid Bridge Max Age %d valid range 6.0 - 40.0", c.MaxAge))
	}

	if c.HelloTime < 1 ||
		c.HelloTime > 2 {
		return errors.New(fmt.Sprintf("Invalid Bridge Hello Time %d valid range 1.0 - 2.0", c.HelloTime))
	}

	if c.ForwardDelay < 3 ||
		c.ForwardDelay > 30 {
		return errors.New(fmt.Sprintf("Invalid Bridge Hello Time %d valid range 3.0 - 30.0", c.ForwardDelay))
	}

	// 1 == STP
	// 2 == RSTP
	// 3 == MSTP currently not support
	if c.ForceVersion != 1 &&
		c.ForceVersion != 2 {
		return errors.New(fmt.Sprintf("Invalid Bridge Force Version %d valid 1 (STP) 2 (RSTP)", c.ForceVersion))
	}

	if c.TxHoldCount < 1 ||
		c.TxHoldCount > 10 {
		return errors.New(fmt.Sprintf("Invalid Bridge Tx Hold Count %d valid range 1 - 10", c.TxHoldCount))
	}

	// if zero is used then we will convert this to use default
	if c.Vlan != 0 &&
		c.Vlan != DEFAULT_STP_BRIDGE_VLAN {
		if c.Vlan < 1 ||
			c.Vlan > 4094 {
			return errors.New(fmt.Sprintf("Invalid Bridge Vlan %d valid range 1 - 4094", c.TxHoldCount))
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

	if c.IfIndex == 0 {
		return errors.New(fmt.Sprintf("Invalid Port %d Must be created against a valid bridge interface", c.IfIndex))
	}

	// Table 17-2
	if _, ok := validStpPortPriorityMap[c.Priority]; !ok {
		return errors.New(fmt.Sprintf("Invalid Port %d Priority %d valid values %v", c.Priority, c.Priority, []uint16{
			0, 16, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240}))
	}

	if c.AdminPathCost > 200000000 {
		return errors.New(fmt.Sprintf("Invalid Port %d Path Cost %d valid values 0 (AUTO) or 1 - 200,000,000", c.Priority, c.AdminPathCost))
	}

	if StpFindPortByIfIndex(c.IfIndex, c.BrgIfIndex, &p) {
		if (!p.OperEdge && !c.AdminEdgePort) &&
			p.BpduGuard {
			return errors.New(fmt.Sprintf("Invalid Port %d Bpdu Guard only available on Edge Ports", c.IfIndex))
		}

		if (p.OperEdge || c.AdminEdgePort) &&
			p.BridgeAssurance {
			return errors.New(fmt.Sprintf("Invalid Port %d Bridge Assurance only available on non Edge Ports", c.IfIndex))
		}
	}
	/*
		Taken care of as part of create
		// all bridge port configurations are applied against all bridge ports applied to a given
		// port
		brgifindex := c.BrgIfIndex
		c.BrgIfIndex = 0
		if _, ok := StpPortConfigMap[c.IfIndex]; !ok {
			StpPortConfigMap[c.IfIndex] = *c
		} else {
			if *c != StpPortConfigMap[c.IfIndex] {
				return errors.New(fmt.Sprintf("Invalid Config params don't equal other bridge port config", c, StpPortConfigMap[c.IfIndex]))
			}
		}
		c.BrgIfIndex = brgifindex
	*/

	return nil
}

func StpBridgeCreate(c *StpBridgeConfig) error {
	var b *Bridge
	tmpaddr := c.Address
	if tmpaddr == "" {
		tmpaddr = "00:AA:AA:BB:BB:DD"
	}

	key := BridgeKey{
		Vlan: c.Vlan,
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
		return errors.New(fmt.Sprintf("Invalid config, bridge vlan %d already exists", c.Vlan))
	}
	return nil
}

func StpBridgeDelete(c *StpBridgeConfig) error {
	var b *Bridge

	key := BridgeKey{
		Vlan: c.Vlan,
	}
	if StpFindBridgeById(key, &b) {
		DelStpBridge(b, true)
		for _, btmp := range StpBridgeConfigMap {
			if btmp.Vlan == c.Vlan {
				delete(StpBridgeConfigMap, b.BrgIfIndex)
			}
		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, bridge vlan %d does not exists", c.Vlan))
	}
	return nil
}

func StpPortCreate(c *StpPortConfig) error {
	var p *StpPort
	var b *Bridge
	if !StpFindPortByIfIndex(c.IfIndex, c.BrgIfIndex, &p) {
		brgIfIndex := c.BrgIfIndex
		c.BrgIfIndex = 0
		// lets store the configuration
		err := StpPortConfigSave(c, false)
		if err != nil {
			return err
		}

		c.BrgIfIndex = brgIfIndex
		// nothing should happen until a birdge is assigned to the port
		if StpFindBridgeByIfIndex(c.BrgIfIndex, &b) {
			p := NewStpPort(c)
			StpPortAddToBridge(p.IfIndex, p.BrgIfIndex)
		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, port %d bridge %d already exists", c.IfIndex, c.BrgIfIndex))
	}
	return nil
}

func StpPortDelete(c *StpPortConfig) error {
	var p *StpPort
	var b *Bridge
	if StpFindPortByIfIndex(c.IfIndex, c.BrgIfIndex, &p) {
		if StpFindBridgeByIfIndex(p.BrgIfIndex, &b) {
			StpPortDelFromBridge(c.IfIndex, p.BrgIfIndex)
		}
		DelStpPort(p)
		foundPort := false
		for _, ptmp := range PortListTable {
			if ptmp.IfIndex == p.IfIndex {
				foundPort = true
			}
		}
		if !foundPort {
			delete(StpPortConfigMap, c.IfIndex)
		}
	} else {
		return errors.New(fmt.Sprintf("Invalid config, port %d bridge %d does not exists", c.IfIndex, c.BrgIfIndex))
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
		p.NotifyPortEnabled("CONFIG DEL", p.PortEnabled, false)
		p.PortEnabled = false
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
			prevval := c.Priority
			c.Priority = priority
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
				c.Priority = prevval
			}
			return err
		} else {
			return nil
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
			prevval := c.ForceVersion
			c.ForceVersion = version
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
				c.ForceVersion = prevval
			}
			return err
		} else {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Force Version", bId))
}

func StpPortPrioritySet(pId int32, bId int32, priority uint16) error {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.Priority != priority {
			c := StpPortConfigGet(pId)
			c.Priority = priority
			err := StpPortConfigParamCheck(c)
			if err == nil {
				// apply to all bridge ports
				for _, port := range p.GetPortListToApplyConfigTo() {

					port.Priority = priority
					port.Selected = false
					port.Reselect = true

					port.b.PrsMachineFsm.PrsEvents <- MachineEvent{
						e:   PrsEventReselect,
						src: "CONFIG: PortPrioritySet",
					}
				}
			}
			return err
		} else {
			return nil
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
			prevval := c.AdminEdgePort
			c.AdminEdgePort = adminedge
			err := StpPortConfigParamCheck(c)
			if err == nil {
				p.AdminEdge = adminedge
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
				c.AdminEdgePort = prevval
			}
			return err
		} else {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Port Admin Edge", pId, bId))
}

func StpBrgForwardDelaySet(bId int32, fwddelay uint16) error {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.ForwardDelay
		c.ForwardDelay = fwddelay
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
			c.ForwardDelay = prevval
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
		prevval := c.HelloTime
		c.HelloTime = hellotime
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
			c.HelloTime = prevval
		}
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Hello Time", bId))
}

func StpBrgMaxAgeSet(bId int32, maxage uint16) error {
	var b *Bridge
	var p *StpPort
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.MaxAge
		c.MaxAge = maxage
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
			c.MaxAge = prevval
		}
		return err
	}
	return errors.New(fmt.Sprintf("Invalid bridge %d supplied for setting Max Age", bId))
}

func StpBrgTxHoldCountSet(bId int32, txholdcount uint16) error {
	var b *Bridge
	if StpFindBridgeByIfIndex(bId, &b) {
		c := StpBrgConfigGet(bId)
		prevval := c.TxHoldCount
		c.TxHoldCount = int32(txholdcount)
		err := StpBrgConfigParamCheck(c)
		if err == nil {
			b.TxHoldCount = uint64(txholdcount)
		} else {
			c.TxHoldCount = prevval
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
			prevval := c.ProtocolMigration
			if protocolmigration {
				c.ProtocolMigration = int32(1)
			} else {
				c.ProtocolMigration = int32(0)
			}
			err := StpPortConfigParamCheck(c)
			if err == nil {
				// apply to all bridge ports
				for _, port := range p.GetPortListToApplyConfigTo() {

					port.PpmmMachineFsm.PpmmEvents <- MachineEvent{e: PpmmEventMcheck,
						src: "CONFIG: ProtocolMigrationSet",
					}
					port.Mcheck = true
				}
			} else {
				c.ProtocolMigration = prevval
			}
			return err
		} else {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Protcol Migration", pId, bId))
}

func StpPortBpduGuardSet(pId int32, bId int32, bpduguard bool) error {
	var p *StpPort
	if StpFindPortByIfIndex(pId, bId, &p) {
		if p.BpduGuard != bpduguard {
			c := StpPortConfigGet(pId)
			prevval := c.BpduGuard
			c.BpduGuard = bpduguard
			err := StpPortConfigParamCheck(c)
			if err == nil {
				// apply to all bridge ports
				for _, port := range p.GetPortListToApplyConfigTo() {
					if bpduguard {
						StpMachineLogger("INFO", "CONFIG", port.IfIndex, port.BrgIfIndex, "Setting BPDU Guard")
					} else {
						StpMachineLogger("INFO", "CONFIG", port.IfIndex, port.BrgIfIndex, "Clearing BPDU Guard")
					}
					port.BpduGuard = bpduguard
				}
			} else {
				c.BpduGuard = prevval
			}
			return err
		} else {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Bpdu Guard", pId, bId))
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
				// apply to all bridge ports
				for _, port := range p.GetPortListToApplyConfigTo() {
					if bridgeassurance {
						StpMachineLogger("INFO", "CONFIG", port.IfIndex, port.BrgIfIndex, "Setting Bridge Assurance")
					} else {
						StpMachineLogger("INFO", "CONFIG", port.IfIndex, port.BrgIfIndex, "Clearing Bridge Assurance")
					}
					port.BridgeAssurance = bridgeassurance
					port.BridgeAssuranceInconsistant = false
					port.BAWhileTimer.count = int32(p.b.RootTimes.HelloTime * 3)
				}
			} else {
				c.BridgeAssurance = prevval
			}
			return err
		} else {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Invalid port %d or bridge %d supplied for setting Bridge Assurance", pId, bId))
}
