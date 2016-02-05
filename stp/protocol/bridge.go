// bridge.go
package stp

import (
	"net"
	"sync"
)

const BridgeConfigModuleStr = "Bridge Config"

var BridgeMapTable map[BridgeId]*Bridge

type BridgeId [8]uint8

type Bridge struct {

	// 17.18.1
	Begin bool
	// 17.18.2
	BridgeIdentifier BridgeId
	// 17.18.3
	// Root/Designated equal to Bridge Identifier
	BridgePriority PriorityVector
	// 17.18.4
	BridgeTimes Times
	// 17.18.6
	RootPortId int32
	// 17.18.7
	RootTimes Times
	// Stp IfIndex
	StpPorts []int32

	PrsMachineFsm *PrsMachine

	// a way to sync all machines
	wg sync.WaitGroup
}

type PriorityVector struct {
	RootBridgeId       BridgeId
	RootPathCost       uint32
	DesignatedBridgeId BridgeId
	DesignatedPortId   uint16
	BridgePortId       uint16
}

type Times struct {
	ForwardingDelay uint16
	HelloTime       uint16
	MaxAge          uint16
	MessageAge      uint16
}

func NewStpBridge(c *StpBridgeConfig) *Bridge {

	/*
		Dot1dBridgeAddressKey      string `SNAPROUTE: KEY`
		Dot1dStpPriorityKey        int32  `SNAPROUTE: KEY`
		Dot1dStpBridgeMaxAge       int32
		Dot1dStpBridgeHelloTime    int32
		Dot1dStpBridgeForwardDelay int32
		Dot1dStpBridgeForceVersion int32
		Dot1dStpBridgeTxHoldCount  int32
	*/
	var addr [6]uint8
	netAddr, _ := net.ParseMAC(c.Dot1dBridgeAddressKey)
	for i := 0; i < 6; i++ {
		addr[i] = netAddr[i]

	}
	bridgeId := CreateBridgeId(addr, c.Dot1dStpPriorityKey)

	b := &Bridge{
		Begin:            true,
		BridgeIdentifier: bridgeId,
		BridgePriority: PriorityVector{
			RootBridgeId:       bridgeId,
			RootPathCost:       0,
			DesignatedBridgeId: bridgeId,
			DesignatedPortId:   0,
			BridgePortId:       0,
		},
		BridgeTimes: Times{
			ForwardingDelay: c.Dot1dStpBridgeForwardDelay,
			HelloTime:       c.Dot1dStpBridgeHelloTime,
			MaxAge:          c.Dot1dStpBridgeMaxAge,
			MessageAge:      0,
		},
		RootPortId: 0, // this will be set once a port is set as root
		RootTimes: Times{ForwardingDelay: c.Dot1dStpBridgeForwardDelay,
			HelloTime:  c.Dot1dStpBridgeHelloTime,
			MaxAge:     c.Dot1dStpBridgeMaxAge,
			MessageAge: 0}, // this will be set once a port is set as root
	}

	BridgeMapTable[b.BridgeIdentifier] = b
	return b
}

func (b *Bridge) Stop() {

	if b.PrsMachineFsm != nil {
		b.PrsMachineFsm.Stop()
		b.PrsMachineFsm = nil
	}

	// lets wait for the machines to close
	b.wg.Wait()

}

func (b *Bridge) BEGIN(restart bool) {
	bridgeResponse := make(chan string)
	if !restart {
		// start all the State machines
		// Port Role Selection State Machine (one instance per bridge)
		b.PrsMachineMain()

	}

	// Prsm
	if b.PrsMachineFsm != nil {
		b.PrsMachineFsm.PrsEvents <- MachineEvent{e: PrsEventBegin,
			src:          BridgeConfigModuleStr,
			responseChan: bridgeResponse}
	}

	<-bridgeResponse
	b.Begin = false
}

func StpFindBridgeById(bridgeId BridgeId, brg **Bridge) bool {

	for bId, b := range BridgeMapTable {
		if bridgeId == bId {
			*brg = b
			return true
		}
	}
	return false
}

func CreateBridgeId(bridgeAddress [6]uint8, bridgePriority uint16) BridgeId {
	return BridgeId{uint8(bridgePriority >> 8 & 0xff),
		uint8(bridgePriority & 0xff),
		bridgeAddress[0],
		bridgeAddress[1],
		bridgeAddress[2],
		bridgeAddress[3],
		bridgeAddress[4],
		bridgeAddress[5]}

}

// Compare BridgeId
// 0 == equal
// > 1 == greater than
// < 1 == less than
func CompareBridgeId(b1 BridgeId, b2 BridgeId) int {
	if b1[0] < b2[0] ||
		b1[1] < b2[1] ||
		b1[2] < b2[2] ||
		b1[3] < b2[3] ||
		b1[4] < b2[4] ||
		b1[5] < b2[5] ||
		b1[6] < b2[6] ||
		b1[7] < b2[7] {
		StpLogger("INFO", "CompareBridgeId returns -1")
		return -1
	} else if b1 == b2 {

		StpLogger("INFO", "CompareBridgeId returns 0")
		return 0
	} else {

		StpLogger("INFO", "CompareBridgeId returns 1")
		return 1
	}
}

// 17.6 Priority vector calculations
func IsMsgPriorityVectorSuperiorThanPortPriorityVector(msg *PriorityVector, port *PriorityVector) bool {
	return (CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) < 0) ||
		((CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) == 0) && (msg.RootPathCost < port.RootPathCost)) ||
		((CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) == 0) && (msg.RootPathCost == port.RootPathCost) && (CompareBridgeId(msg.DesignatedBridgeId, port.DesignatedBridgeId) < 0)) ||
		((CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) == 0) && (msg.RootPathCost == port.RootPathCost) && (CompareBridgeId(msg.DesignatedBridgeId, port.DesignatedBridgeId) == 0) && (msg.DesignatedPortId < port.DesignatedPortId)) ||
		((msg.DesignatedBridgeId[2] == port.DesignatedBridgeId[2] &&
			msg.DesignatedBridgeId[3] == port.DesignatedBridgeId[3] &&
			msg.DesignatedBridgeId[4] == port.DesignatedBridgeId[4] &&
			msg.DesignatedBridgeId[5] == port.DesignatedBridgeId[5] &&
			msg.DesignatedBridgeId[6] == port.DesignatedBridgeId[6] &&
			msg.DesignatedBridgeId[7] == port.DesignatedBridgeId[7]) && (msg.DesignatedPortId == port.DesignatedPortId))
}

func IsMsgPriorityVectorTheSameOrWorseThanPortPriorityVector(msg *PriorityVector, port *PriorityVector) bool {
	return (msg == port) ||
		IsMsgPriorityVectorSuperiorThanPortPriorityVector(port, msg)
}

func IsMsgPriorityVectorWorseThanPortPriorityVector(msg *PriorityVector, port *PriorityVector) bool {
	return (CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) > 0) ||
		((CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) == 0 && (msg.RootPathCost > port.RootPathCost)) ||
			((CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) == 0) && (msg.RootPathCost == port.RootPathCost) && (CompareBridgeId(msg.DesignatedBridgeId, port.DesignatedBridgeId) > 0))) ||
		((CompareBridgeId(msg.RootBridgeId, port.RootBridgeId) == 0) && (msg.RootPathCost == port.RootPathCost) && (CompareBridgeId(msg.DesignatedBridgeId, port.DesignatedBridgeId) == 0) && (msg.DesignatedPortId > port.DesignatedPortId))

}

func CalcRootPathPriorityVector(msg *PriorityVector, port *PriorityVector) (rootPriorityVector *PriorityVector) {

	rootPriorityVector = &PriorityVector{
		RootBridgeId:       msg.RootBridgeId,
		RootPathCost:       msg.RootPathCost + port.RootPathCost,
		DesignatedBridgeId: msg.DesignatedBridgeId,
		DesignatedPortId:   msg.DesignatedPortId,
		BridgePortId:       msg.BridgePortId,
	}

	return rootPriorityVector
}

func (b *Bridge) AllSynced() bool {

	var p *StpPort
	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			if !p.Synced {
				return false
			}
		}
	}
	return true
}
