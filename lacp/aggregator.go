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
	octetsTx int
	octetsRx int
	framesTx int
	framesRx int
}

// 802.1ax-2014 Section 6.4.6 Variables associated with each Aggregator
// Section 7.3.1.1

type LaAggregator struct {
	// 802.1ax Section 7.3.1.1
	aggId               int
	aggDescription      string   // 255 max chars
	aggName             string   // 255 max chars
	actorSystemId       [6]uint8 // mac address
	actorSystemPriority int
	// aggregation capability
	// TRUE - port attached to this aggregetor is not capable
	//        of aggregation to any other aggregator
	// FALSE - port attached to this aggregator is able of
	//         aggregation to any other aggregator
	aggOrIndividual bool
	actorAdminKey   uint16
	actorOperKey    uint16
	aggMacAddr      [6]uint
	// remote system
	partnerSystemId       [6]uint8
	partnerSystemPriority int
	partnerOperKey        int

	// up/down
	adminState int
	operState  int

	// sum of data rate of each link in aggregation (read-only)
	dataRate int

	timeOfLastOperChange time.Time

	receive_state  bool
	transmit_state bool

	ready bool

	// Port number from LaAggPort
	PortNumList []uint16
}

// TODO add more defaults
func NewLaAggregator(ac *LaAggConfig) *LaAggregator {
	a := &LaAggregator{
		aggId:               ac.Id,
		actorAdminKey:       ac.Key,
		actorSystemId:       gLacpSysGlobalInfo.SystemDefaultParams.actor_system,
		actorSystemPriority: gLacpSysGlobalInfo.SystemDefaultParams.actor_system_priority,
		partnerSystemId:     [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		ready:               true,
		PortNumList:         make([]uint16, 0),
	}

	// add agg to map
	gLacpSysGlobalInfo.AggMap[ac.Id] = a

	for _, pId := range ac.LagMembers {
		a.PortNumList = append(a.PortNumList, pId)
	}

	return a
}

func LaFindAggById(aggId int, agg **LaAggregator) bool {
	a, ok := gLacpSysGlobalInfo.AggMap[aggId]
	if ok {
		*agg = a
	}
	return ok
}
