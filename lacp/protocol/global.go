// global
package lacp

import (
	"fmt"
)

type TxCallback func(port uint16, data interface{})

type LacpSysGlobalInfo struct {
	LacpEnabled                bool
	PortMap                    map[uint16]*LaAggPort
	AggMap                     map[int]*LaAggregator
	SystemDefaultParams        LacpSystem
	PartnerSystemDefaultParams LacpSystem
	ActorStateDefaultParams    LacpPortInfo
	PartnerStateDefaultParams  LacpPortInfo

	// mux machine coupling
	// false == NOT COUPLING, true == COUPLING
	muxCoupling bool

	// list of tx function which should be called for a given port
	TxCallbacks map[uint16][]TxCallback
}

// holds default lacp state info
var gLacpSysGlobalInfo map[[6]uint8]*LacpSysGlobalInfo

// NewLacpSysGlobalInfo will create a port map, agg map
// as well as set some default parameters to be used
// to setup each new port.
//
// NOTE: Only one instance should exist on live system
func LacpSysGlobalInfoInit(sysId [6]uint8) *LacpSysGlobalInfo {

	if gLacpSysGlobalInfo == nil {
		gLacpSysGlobalInfo = make(map[[6]uint8]*LacpSysGlobalInfo)
	}

	if _, ok := gLacpSysGlobalInfo[sysId]; !ok {
		fmt.Println("LASYS: global vars init sysId", sysId)
		gLacpSysGlobalInfo[sysId] = &LacpSysGlobalInfo{
			LacpEnabled:                true,
			PortMap:                    make(map[uint16]*LaAggPort),
			AggMap:                     make(map[int]*LaAggregator),
			SystemDefaultParams:        LacpSystem{actor_system_priority: 0x8000},
			PartnerSystemDefaultParams: LacpSystem{actor_system_priority: 0x0},
			TxCallbacks:                make(map[uint16][]TxCallback),
		}

		gLacpSysGlobalInfo[sysId].SystemDefaultParams.LacpSystemActorSystemIdSet(sysId)

		// Partner is brought up as aggregatible
		const aggregatible uint8 = (LacpStateActivityBit |
			LacpStateAggregationBit |
			LacpStateSyncBit |
			LacpStateCollectingBit |
			LacpStateDistributingBit |
			LacpStateDefaultedBit)
		LacpStateSet(&gLacpSysGlobalInfo[sysId].PartnerStateDefaultParams.state, aggregatible)

		// Actor is brought up as individual
		const individual uint8 = (LacpStateDefaultedBit | LacpStateActivityBit)
		LacpStateSet(&gLacpSysGlobalInfo[sysId].ActorStateDefaultParams.state, individual)
	}
	return gLacpSysGlobalInfo[sysId]
}

func LacpSysGlobalInfoGet(sysId [6]uint8) *LacpSysGlobalInfo {
	return LacpSysGlobalInfoInit(sysId)
}

func LacpSysGlobalDefaultSystemGet(sysId [6]uint8) *LacpSystem {
	return &gLacpSysGlobalInfo[sysId].SystemDefaultParams
}
func LacpSysGlobalDefaultPartnerSystemGet(sysId [6]uint8) *LacpSystem {
	return &gLacpSysGlobalInfo[sysId].PartnerSystemDefaultParams
}
func LacpSysGlobalDefaultPartnerInfoGet(sysId [6]uint8) *LacpPortInfo {
	return &gLacpSysGlobalInfo[sysId].PartnerStateDefaultParams
}
func LacpSysGlobalDefaultActorSystemGet(sysId [6]uint8) *LacpPortInfo {
	return &gLacpSysGlobalInfo[sysId].ActorStateDefaultParams
}

func (g *LacpSysGlobalInfo) LaSysGlobalRegisterTxCallback(port uint16, f TxCallback) {
	g.TxCallbacks[port] = append(g.TxCallbacks[port], f)
}

func LaSysGlobalTxCallbackListGet(p *LaAggPort) []TxCallback {

	if s, sok := gLacpSysGlobalInfo[p.sysId]; sok {
		if fList, pok := s.TxCallbacks[p.portNum]; pok {
			return fList
		}
	}

	// temporary function
	x := func(port uint16, data interface{}) {
		fmt.Println("TX not registered for port", p.intfNum, p.portId)
		lacp := data.(*EthernetLacpFrame)
		if lacp.lacp.subType == LacpSubType {
			//fmt.Printf("%#v\n", *lacp)
		} else if lacp.lacp.subType == LampSubType {
			//lamp := data.(*EthernetLampFrame)
			//fmt.Printf("%#v\n", *lamp)
		}
	}

	debugTxList := make([]TxCallback, 0)
	debugTxList = append(debugTxList, x)
	return debugTxList
}
