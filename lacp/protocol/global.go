// global
package lacp

import (
	"fmt"
	"net"
	//"github.com/google/gopacket/layers"
)

type TxCallback func(port uint16, data interface{})

type PortIdKey struct {
	Name string
	Id   uint16
}

type AggIdKey struct {
	Name string
	Id   int
}

type LacpSysGlobalInfo struct {
	LacpEnabled                bool
	PortMap                    map[PortIdKey]*LaAggPort
	AggMap                     map[AggIdKey]*LaAggregator
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

// holds default lacp State info
var gLacpSysGlobalInfo map[LacpSystem]*LacpSysGlobalInfo

func convertNetHwAddressToSysIdKey(mac net.HardwareAddr) [6]uint8 {
	var macArr [6]uint8
	macArr[0] = mac[0]
	macArr[1] = mac[1]
	macArr[2] = mac[2]
	macArr[3] = mac[3]
	macArr[4] = mac[4]
	macArr[5] = mac[5]
	return macArr
}

func convertSysIdKeyToNetHwAddress(mac [6]uint8) net.HardwareAddr {

	x := make(net.HardwareAddr, 6)
	x[0] = mac[0]
	x[1] = mac[1]
	x[2] = mac[2]
	x[3] = mac[3]
	x[4] = mac[4]
	x[5] = mac[5]
	return x
}

// NewLacpSysGlobalInfo will create a port map, agg map
// as well as set some default parameters to be used
// to setup each new port.
//
// NOTE: Only one instance should exist on live System
func LacpSysGlobalInfoInit(sysId LacpSystem) *LacpSysGlobalInfo {

	if gLacpSysGlobalInfo == nil {
		gLacpSysGlobalInfo = make(map[LacpSystem]*LacpSysGlobalInfo)
	}

	sysKey := sysId

	if _, ok := gLacpSysGlobalInfo[sysKey]; !ok {
		fmt.Println("LASYS: global vars init sysId", sysId)
		gLacpSysGlobalInfo[sysKey] = &LacpSysGlobalInfo{
			LacpEnabled:                true,
			PortMap:                    make(map[PortIdKey]*LaAggPort),
			AggMap:                     make(map[AggIdKey]*LaAggregator),
			SystemDefaultParams:        LacpSystem{Actor_System_priority: 0x8000},
			PartnerSystemDefaultParams: LacpSystem{Actor_System_priority: 0x0},
			TxCallbacks:                make(map[string][]TxCallback),
		}

		gLacpSysGlobalInfo[sysKey].SystemDefaultParams.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(sysId.actor_System))

		// Partner is brought up as aggregatible
		LacpStateSet(&gLacpSysGlobalInfo[sysKey].PartnerStateDefaultParams.State, LacpStateAggregatibleUp)

		// Actor is brought up as individual
		LacpStateSet(&gLacpSysGlobalInfo[sysKey].ActorStateDefaultParams.State, LacpStateIndividual)
	}
	return gLacpSysGlobalInfo[sysKey]
}

func LacpSysGlobalInfoGet() map[LacpSystem]*LacpSysGlobalInfo {
	return gLacpSysGlobalInfo
}

func LacpSysGlobalInfoByIdGet(sysId LacpSystem) *LacpSysGlobalInfo {
	return LacpSysGlobalInfoInit(sysId)
}

func LacpSysGlobalDefaultSystemGet(sysId LacpSystem) *LacpSystem {
	return &gLacpSysGlobalInfo[sysId].SystemDefaultParams
}
func LacpSysGlobalDefaultPartnerSystemGet(sysId LacpSystem) *LacpSystem {
	return &gLacpSysGlobalInfo[sysId].PartnerSystemDefaultParams
}
func LacpSysGlobalDefaultPartnerInfoGet(sysId LacpSystem) *LacpPortInfo {
	return &gLacpSysGlobalInfo[sysId].PartnerStateDefaultParams
}
func LacpSysGlobalDefaultActorSystemGet(sysId LacpSystem) *LacpPortInfo {
	return &gLacpSysGlobalInfo[sysId].ActorStateDefaultParams
}

func (g *LacpSysGlobalInfo) LaSysGlobalRegisterTxCallback(intf string, f TxCallback) {
	g.TxCallbacks[intf] = append(g.TxCallbacks[intf], f)
}

func (g *LacpSysGlobalInfo) LaSysGlobalDeRegisterTxCallback(intf string) {
	delete(g.TxCallbacks, intf)
}

func LaSysGlobalTxCallbackListGet(p *LaAggPort) []TxCallback {

	var a *LaAggregator
	var sysId LacpSystem
	if LaFindAggById(p.AggId, &a) {
		mac, _ := net.ParseMAC(a.Config.SystemIdMac)
		sysId.actor_System = convertNetHwAddressToSysIdKey(mac)
		sysId.Actor_System_priority = a.Config.SystemPriority
	}
	fmt.Println("txCallbackGet %#v", sysId)
	if s, sok := gLacpSysGlobalInfo[sysId]; sok {
		if fList, pok := s.TxCallbacks[p.IntfNum]; pok {
			return fList
		}
	}

	// temporary function
	x := func(port uint16, data interface{}) {
		fmt.Println("TX not registered for port", p.IntfNum, p.portId)
		//lacp := data.(*layers.LACP)
		//fmt.Printf("%#v\n", *lacp)
	}

	debugTxList := make([]TxCallback, 0)
	debugTxList = append(debugTxList, x)
	return debugTxList
}
