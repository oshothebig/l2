// aggregator
package lacp

import (
	"fmt"
	"net"
	"time"
)

// Indicates on a port what state
// the aggSelected is in
const (
	LacpAggSelected = iota + 1
	LacpAggStandby
	LacpAggUnSelected
)

type LacpAggregatorStats struct {
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
	aggId          int
	AggDescription string // 255 max chars
	AggName        string // 255 max chars
	AggType        uint32 // LACP/STATIC
	AggMinLinks    uint16

	// lacp configuration info
	Config LacpConfigInfo

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
	aggMacAddr [6]uint8
	// Partner_System
	partnerSystemId [6]uint8
	// Partner_System_Priority
	partnerSystemPriority int
	// Partner_Oper_Aggregator_Key
	partnerOperKey int

	//		1 : string 	NameKey
	//	    2 : i32 	Interval
	// 	    3 : i32 	LacpMode
	//	    4 : string 	SystemIdMac
	//	    5 : i16 	SystemPriority

	// UP/DOWN
	adminState int
	operState  int

	// date of last oper change
	timeOfLastOperChange time.Time

	// aggrigator stats
	stats LacpAggregatorStats

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

	// Ports in Distributed state
	DistributedPortNumList []string
}

func NewLaAggregator(ac *LaAggConfig) *LaAggregator {
	netMac, _ := net.ParseMAC(ac.Lacp.SystemIdMac)
	sysId := LacpSystem{
		actor_system:          convertNetHwAddressToSysIdKey(netMac),
		actor_system_priority: ac.Lacp.SystemPriority,
	}
	sgi := LacpSysGlobalInfoByIdGet(sysId)
	a := &LaAggregator{
		AggName:                ac.Name,
		aggId:                  ac.Id,
		aggMacAddr:             sysId.actor_system,
		actorAdminKey:          ac.Key,
		AggType:                ac.Type,
		AggMinLinks:            ac.MinLinks,
		Config:                 ac.Lacp,
		partnerSystemId:        [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		ready:                  true,
		PortNumList:            make([]uint16, 0),
		DistributedPortNumList: make([]string, 0),
	}

	// want to ensure that the application can use a string name or id
	// to uniquely identify a lag
	key := AggIdKey{Id: ac.Id,
		Name: ac.Name}

	// add agg to map
	sgi.AggMap[key] = a

	for _, pId := range ac.LagMembers {
		a.PortNumList = append(a.PortNumList, pId)
	}

	return a
}

func LaGetAggNext(agg **LaAggregator) bool {
	returnNext := false
	for _, sgi := range LacpSysGlobalInfoGet() {
		for id, a := range sgi.AggMap {
			if *agg == nil {
				fmt.Println("agg map curr %d", a.aggId)
			} else {
				fmt.Println("agg map prev %d curr %d found %d", *agg.aggId, a.aggId)
			}
			if *agg == nil {
				// first agg
				*agg = a
			} else if *agg == a {
				// found agg
				returnNext = true
			} else if returnNext {
				// next agg
				*agg = a
				return true
			}
		}
	}
	*agg = nil
	return false
}

func LaFindAggById(aggId int, agg **LaAggregator) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.AggMap {
			if a.aggId == aggId {
				*agg = a
				return true
			}
		}
	}
	return false
}

func LaFindAggByName(AggName string, agg **LaAggregator) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.AggMap {
			if a.AggName == AggName {
				*agg = a
				return true
			}
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

func LaFindAggByKey(key uint16, agg **LaAggregator) bool {

	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.AggMap {
			if a.actorAdminKey == key {
				*agg = a
				return true
			}
		}
	}
	return false
}
