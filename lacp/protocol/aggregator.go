// aggregator
package lacp

import (
	"time"
)

// Indicates on a port what state
// the aggSelected is in
const (
	LacpAggSelected = iota + 1
	LacpAggStandby
	LacpAggUnSelected
)

type LacpAggrigatorStats struct {
	// does not include lacp or marker pdu
	octetsTx              int
	octetsRx              int
	framesTx              int
	framesRx              int
	mcFramesTxOk          int
	mcFramesRxOk          int
	bcFramesTxOk          int
	bcFramesRxOk          int
	framesDiscardedonTx   int
	framesDiscardedonRx   int
	framesWithTxErrors    int
	framesWithRxErrors    int
	unknownProtocolFrames int
}

// 802.1ax-2014 Section 6.4.6 Variables associated with each Aggregator
// Section 7.3.1.1

type LaAggregator struct {
	// 802.1ax Section 7.3.1.1 && 6.3.2
	// Aggregator_Identifier
	aggId               int
	aggDescription      string   // 255 max chars
	aggName             string   // 255 max chars
	actorSystemId       [6]uint8 // mac address
	actorSystemPriority uint16
	// aggregation capability
	// TRUE - port attached to this aggregetor is not capable
	//        of aggregation to any other aggregator
	// FALSE - port attached to this aggregator is able of
	//         aggregation to any other aggregator
	// Individual_Aggregator
	aggOrIndividual bool
	// Actor_Admin_Aggregator_Key
	actorAdminKey uint16
	// Actor_Oper_Aggregator_Key
	actorOperKey uint16
	//Aggregator_MAC_address
	aggMacAddr [6]uint
	// Partner_System
	partnerSystemId [6]uint8
	// Partner_System_Priority
	partnerSystemPriority int
	// Partner_Oper_Aggregator_Key
	partnerOperKey int

	// UP/DOWN
	adminState int
	operState  int

	// date of last oper change
	timeOfLastOperChange time.Time

	// aggrigator stats
	stats LacpAggrigatorStats

	// Receive_State
	rxState bool
	// Transmit_State
	txState bool

	// sum of data rate of each link in aggregation (read-only)
	dataRate int

	// LAG is ready to add a port in the ReadyN state
	ready bool

	// Port number from LaAggPort
	// LAG_Ports
	PortNumList []uint16
}

// TODO add more defaults
func NewLaAggregator(ac *LaAggConfig) *LaAggregator {
	sgi := LacpSysGlobalInfoGet(ac.SysId)
	a := &LaAggregator{
		aggId:               ac.Id,
		actorAdminKey:       ac.Key,
		actorSystemId:       sgi.SystemDefaultParams.actor_system,
		actorSystemPriority: sgi.SystemDefaultParams.actor_system_priority,
		partnerSystemId:     [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		ready:               true,
		PortNumList:         make([]uint16, 0),
	}

	// add agg to map
	sgi.AggMap[ac.Id] = a

	for _, pId := range ac.LagMembers {
		a.PortNumList = append(a.PortNumList, pId)
	}

	return a
}

func LaFindAggById(aggId int, agg **LaAggregator) bool {
	for _, sgi := range gLacpSysGlobalInfo {
		if a, ok := sgi.AggMap[aggId]; ok {
			*agg = a
			return ok
		}
	}
	return false
}

func LaAggPortNumListPortIdExist(aggId int, portId uint16) bool {
	var a *LaAggregator
	if LaFindAggById(aggId, &a) {
		for _, pId := range a.PortNumList {
			if pId == portId {
				return true
			}
		}
	}
	return false
}
