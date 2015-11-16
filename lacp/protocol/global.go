// global
package lacp

import (
	"fmt"
	//"github.com/google/gopacket/layers"
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
	TxCallbacks map[string][]TxCallback
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
			TxCallbacks:                make(map[string][]TxCallback),
		}

		gLacpSysGlobalInfo[sysId].SystemDefaultParams.LacpSystemActorSystemIdSet(sysId)

		// Partner is brought up as aggregatible
		LacpStateSet(&gLacpSysGlobalInfo[sysId].PartnerStateDefaultParams.state, LacpStateAggregatibleUp)

		// Actor is brought up as individual
		LacpStateSet(&gLacpSysGlobalInfo[sysId].ActorStateDefaultParams.state, LacpStateIndividual)
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

func (g *LacpSysGlobalInfo) LaSysGlobalRegisterTxCallback(intf string, f TxCallback) {
	g.TxCallbacks[intf] = append(g.TxCallbacks[intf], f)
}

func (g *LacpSysGlobalInfo) LaSysGlobalDeRegisterTxCallback(intf string) {
	delete(g.TxCallbacks, intf)
}

func LaSysGlobalTxCallbackListGet(p *LaAggPort) []TxCallback {

	if s, sok := gLacpSysGlobalInfo[p.sysId]; sok {
		if fList, pok := s.TxCallbacks[p.intfNum]; pok {
			return fList
		}
	}

	// temporary function
	x := func(port uint16, data interface{}) {
		fmt.Println("TX not registered for port", p.intfNum, p.portId)
		//lacp := data.(*layers.LACP)
		//fmt.Printf("%#v\n", *lacp)
	}

	debugTxList := make([]TxCallback, 0)
	debugTxList = append(debugTxList, x)
	return debugTxList
}
