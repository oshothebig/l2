// global
package lacp

import (
	"fmt"
)

type LacpSysGlobalInfo struct {
	// global port map representation of the LaAggPorts
	LacpEnabled                bool
	PortMap                    map[uint16]*LaAggPort
	AggMap                     map[int]*LaAggregator
	SystemDefaultParams        LacpSystem
	PartnerSystemDefaultParams LacpSystem
	ActorStateDefaultParams    LacpPortInfo
	PartnerStateDefaultParams  LacpPortInfo
}

// holds default lacp state info
var gLacpSysGlobalInfo *LacpSysGlobalInfo

// NewLacpSysGlobalInfo will create a port map, agg map
// as well as set some default parameters to be used
// to setup each new port.
//
// NOTE: Only one instance should exist
func LacpSysGlobalInfoInit() {
	if gLacpSysGlobalInfo == nil {
		fmt.Println("LASYS: global vars init")
		gLacpSysGlobalInfo = &LacpSysGlobalInfo{
			LacpEnabled:                true,
			PortMap:                    make(map[uint16]*LaAggPort),
			AggMap:                     make(map[int]*LaAggregator),
			SystemDefaultParams:        LacpSystem{actor_system_priority: 0x8000},
			PartnerSystemDefaultParams: LacpSystem{actor_system_priority: 0x0},
		}

		// Partner is brought up as aggregatible
		const aggregatible uint8 = (LacpStateAggregationBit | LacpStateSyncBit |
			LacpStateCollectingBit | LacpStateDistributingBit |
			LacpStateDefaultedBit)
		LacpStateSet(&gLacpSysGlobalInfo.PartnerStateDefaultParams.state, aggregatible)

		// Actor is brought up as individual
		const individual uint8 = (LacpStateDefaultedBit)
		LacpStateSet(&gLacpSysGlobalInfo.ActorStateDefaultParams.state, individual)
	}
}

func LacpSysGlobalDefaultSystemGet() *LacpSystem {
	return &gLacpSysGlobalInfo.SystemDefaultParams
}
func LacpSysGlobalDefaultPartnerSystemGet() *LacpSystem {
	return &gLacpSysGlobalInfo.PartnerSystemDefaultParams
}
func LacpSysGlobalDefaultPartnerInfoGet() *LacpPortInfo {
	return &gLacpSysGlobalInfo.PartnerStateDefaultParams
}
func LacpSysGlobalDefaultActorSystemGet() *LacpPortInfo {
	return &gLacpSysGlobalInfo.ActorStateDefaultParams
}
