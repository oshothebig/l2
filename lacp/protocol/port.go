// port
package lacp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"strings"
	"sync"
	"time"
)

type PortProperties struct {
	Mac    net.HardwareAddr
	Speed  int
	Duplex int
	Mtu    int
}

type LacpPortInfo struct {
	System   LacpSystem
	Key      uint16
	Port_pri uint16
	port     uint16
	State    uint8
}

type LacpCounters struct {
	LacpInPkts        uint64
	LacpOutPkts       uint64
	LacpRxErrors      uint64
	LacpTxErrors      uint64
	LacpUnknownErrors uint64
	LacpErrors        uint64
}

type AggPortStatus struct {
	// GET
	AggPortId int
	// GET
	AggPortActorSystemId [6]uint8
	// GET
	AggPortActorOperKey uint16
	// GET
	AggPortPartnerOperSystemPriority uint16
	// GET
	AggPortPartnerOperSystemId [6]uint8
	// GET
	AggPortPartnerOperKey uint16
	// GET
	AggPortSelectedAggID int
	// GET
	AggPortAttachedAggID int
	// GET
	AggPortActorPort int
	// GET
	AggPortPartnerOperPort int
	// GET
	AggPortPartnerOperPortPriority uint8
	// GET
	AggPortActorOperState uint8
	// GET
	AggPortPartnerOperState uint8
	// GET
	AggPortAggregateOrIndividual bool
	// GET
	AggPortOperConversationPasses bool
	// GET
	AggPortOperConversationCollected bool
	// GET
	AggPortStats AggPortStatsObject
	// GET
	AggPortDebug AggPortDebugInformationObject
}

type AggregatorPortObject struct {
	// GET-SET
	Config AggPortConfig
	// GET
	Status AggPortStatus
	/*
		// GET
		AggPortId int
		// GET-SET
		AggPortActorSystemPriority uint16
		// GET
		AggPortActorSystemId [6]uint8
		// GET-SET
		AggPortActorAdminKey uint16
		// GET
		AggPortActorOperKey uint16
		// GET-SET
		AggPortPartnerAdminSystemPriority uint16
		// GET
		AggPortPartnerOperSystemPriority uint16
		// GET-SET
		AggPortPartnerAdminSystemId [6]uint8
		// GET
		AggPortPartnerOperSystemId [6]uint8
		// GET-SET
		AggPortPartnerAdminKey uint16
		// GET
		AggPortPartnerOperKey uint16
		// GET
		AggPortSelectedAggID int
		// GET
		AggPortAttachedAggID int
		// GET
		AggPortActorPort int
		// GET-SET
		AggPortActorPortPriority uint8
		// GET-SET
		AggPortPartnerAdminPort int
		// GET
		AggPortPartnerOperPort int
		// GET-SET
		AggPortPartnerAdminPortPriority uint8
		// GET
		AggPortPartnerOperPortPriority uint8
		// GET-SET
		AggPortActorAdminState uint8
		// GET
		AggPortActorOperState uint8
		// GET-SET
		AggPortPartnerAdminState uint8
		// GET
		AggPortPartnerOperState uint8
		// GET
		AggPortAggregateOrIndividual bool
		// GET
		AggPortOperConversationPasses bool
		// GET
		AggPortOperConversationCollected bool
		// GET-SET
		AggPortLinkNumberID int
		// GET-SET
		AggPortPartnerAdminLInkNumberID int
		// GET-SET
		AggPortWTRTime int
		// GET-SET
		AggPortProtocolDA [6]uint8
		// GET
		AggPortStats AggPortStatsObject
		// GET
		AggPortDebug AggPortDebugInformationObject
	*/

	Internal AggInternalData
}

type AggInternalData struct {
	// Linux Interface Name
	AggNameStr string
}

// GET
type AggPortStatsObject struct {
	AggPortStatsID                   uint64
	AggPortStatsLACPDUsRx            uint64
	AggPortStatsPDUsRx               uint64
	AggPortStatsMarkerPDUsRx         uint64
	AggPortStatsMarkerResponsePDUsRx uint64
	AggPortStatsUnknownRx            uint64
	AggPortStatsIllegalRx            uint64
	AggPortStatsLACPDUsTx            uint64
	AggPortStatsMarkerPDUsTx         uint64
	AggPortStatsMarkerResponsePDUsTx uint64
}

//GET
type AggPortDebugInformationObject struct {
	// same as AggregationPort
	AggPortDebugInformationID int
	// enum
	AggPortDebugRxState    int
	AggPortDebugLastRxTime int
	// enum
	AggPortDebugMuxState                   int
	AggPortDebugMuxReason                  string
	AggPortDebugActorChurnState            int
	AggPortDebugPartnerChurnState          int
	AggPortDebugActorChurnCount            int
	AggPortDebugPartnerChurnCount          int
	AggPortDebugActorSyncTransitionCount   int
	AggPortDebugPartnerSyncTransitionCount int
	// TODO
	AggPortDebugActorChangeCount     int
	AggPortDebugPartnerChangeCount   int
	AggPortDebugActorCDSChurnState   int
	AggPortDebugPartnerCDSChurnState int
	AggPortDebugActorCDSChurnCount   int
	AggPortDebugPartnerCDSChurnCount int
}

// 802.1ax Section 6.4.7
// Port attributes associated with aggregator
type LaAggPort struct {
	// 802.1ax-2014 Section 6.3.4:
	// Link Aggregation Control uses a Port Identifier (Port ID), comprising
	// the concatenation of a Port Priority (7.3.2.1.15) and a Port Number
	// (7.3.2.1.14), to identify the Aggregation Port....
	// The most significant and second most significant octets are the first
	// and second most significant octets of the Port Priority, respectively.
	// The third and fourth most significant octets are the first and second
	// most significant octets of the Port Number, respectively.
	portId int

	// string id of port
	IntfNum string

	// Key
	Key uint16

	// used to form portId
	PortNum      uint16
	portPriority uint16

	AggId int

	// Once selected reference to agg group will be made
	AggAttached *LaAggregator
	aggSelected int
	// unable to aggregate with other links in an agg
	operIndividual int
	lacpEnabled    bool
	// TRUE - Aggregation port is operable (MAC_Operational == True)
	// FALSE - otherwise
	PortEnabled  bool
	portMoved    bool
	begin        bool
	actorChurn   bool
	partnerChurn bool
	readyN       bool

	macProperties PortProperties

	// determine whether a port is up or down
	LinkOperStatus bool

	// administrative values for State described in 6.4.2.3
	actorAdmin   LacpPortInfo
	ActorOper    LacpPortInfo
	partnerAdmin LacpPortInfo
	PartnerOper  LacpPortInfo

	// State machines
	RxMachineFsm  *LacpRxMachine
	PtxMachineFsm *LacpPtxMachine
	TxMachineFsm  *LacpTxMachine
	CdMachineFsm  *LacpCdMachine
	MuxMachineFsm *LacpMuxMachine

	// Counters
	Counters LacpCounters

	// GET
	AggPortDebug AggPortDebugInformationObject

	// will serialize State transition logging per port
	LacpDebug *LacpDebug

	// on configuration changes need to inform all State
	// machines and wait for a response
	portChan chan string
	log      chan string
	logEna   bool
	wg       sync.WaitGroup

	// handle used to tx packets to linux if
	handle *pcap.Handle

	// Version 2
	partnerLacpPduVersionNumber int
	enableLongPduXmit           bool
	// packet is 1 byte, but spec says save as int.
	// going to save as byte
	partnerVersion uint8

	sysId net.HardwareAddr
}

func (p *LaAggPort) LaPortLog(msg string) {
	if p.logEna {
		p.log <- msg
	}
}

// find a port from the global map table by PortNum
func LaFindPortById(pId uint16, port **LaAggPort) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, p := range sgi.LacpSysGlobalAggPortListGet() {
			if p.PortNum == pId {
				*port = p
				return true
			}
		}
	}
	return false
}

func LaConvertPortAndPriToPortId(pId uint16, prio uint16) int {
	return int(pId | prio<<16)
}

func LaGetPortNext(port **LaAggPort) bool {
	returnNext := false
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, p := range sgi.LacpSysGlobalAggPortListGet() {
			if *port == nil {
				// first port
				*port = p
				return true
			} else if (*port).PortNum == p.PortNum {
				// found port, lets return the next port
				returnNext = true
			} else if returnNext {
				// next port
				*port = p
				return true
			}
		}
	}
	*port = nil
	return false
}

// find a port from the global map table by PortNum
func LaFindPortByPortId(portId int, port **LaAggPort) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, p := range sgi.LacpSysGlobalAggPortListGet() {
			if p.portId == portId {
				*port = p
				return true
			}
		}
	}
	return false
}

// LaFindPortByKey will find a port form the global map table by Key
// index value should input 0 for the first value
func LaFindPortByKey(Key uint16, index *int, port **LaAggPort) bool {
	var i int
	for _, sgi := range LacpSysGlobalInfoGet() {
		i = *index
		aggPortList := sgi.LacpSysGlobalAggPortListGet()
		l := len(aggPortList)
		for _, p := range aggPortList {
			if i < l {
				if p.Key == Key {
					*port = p
					*index++
					return true
				}
			}
			i++
		}
	}
	*index = -1
	return false
}

// NewLaAggPort
// Allocate a new lag port, creating appropriate timers
func NewLaAggPort(config *LaAggPortConfig) *LaAggPort {

	// Lets see if the agg exists and add this port to this config
	// otherwise lets use the default
	var a *LaAggregator
	var sysId LacpSystem
	if LaFindAggById(config.AggId, &a) {
		mac, _ := net.ParseMAC(a.Config.SystemIdMac)
		sysId.actor_System = convertNetHwAddressToSysIdKey(mac)
		sysId.Actor_System_priority = a.Config.SystemPriority
	}
	sgi := LacpSysGlobalInfoByIdGet(sysId)

	p := &LaAggPort{
		portId:       LaConvertPortAndPriToPortId(config.Id, config.Prio),
		PortNum:      config.Id,
		portPriority: config.Prio,
		IntfNum:      config.IntfId,
		Key:          config.Key,
		AggId:        0, // this should be set on config AddLaAggPortToAgg
		aggSelected:  LacpAggUnSelected,
		sysId:        config.SysId,
		begin:        true,
		portMoved:    false,
		lacpEnabled:  false,
		PortEnabled:  config.Enable,
		macProperties: PortProperties{Mac: config.Properties.Mac,
			Speed:  config.Properties.Speed,
			Duplex: config.Properties.Duplex,
			Mtu:    config.Properties.Mtu},
		logEna:   true,
		portChan: make(chan string)}

	// Start Port Logger
	p.LacpDebugEventLogMain()

	// default actor admin
	//fmt.Println(config.sysId, gLacpSysGlobalInfo[config.sysId])
	p.actorAdmin.State = sgi.ActorStateDefaultParams.State
	p.actorAdmin.System.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(sgi.SystemDefaultParams.actor_System))
	p.actorAdmin.System.LacpSystemActorSystemPrioritySet(sgi.SystemDefaultParams.Actor_System_priority)
	p.actorAdmin.Key = p.Key
	p.actorAdmin.port = p.PortNum
	p.actorAdmin.Port_pri = p.portPriority

	// default actor oper same as admin
	p.ActorOper = p.actorAdmin

	// default partner admin
	p.partnerAdmin.State = sgi.PartnerStateDefaultParams.State
	// default partner oper same as admin
	p.PartnerOper = p.partnerAdmin

	if config.Mode != LacpModeOn {
		p.lacpEnabled = true
	}

	// add port to port map
	sgi.PortMap[PortIdKey{Name: p.IntfNum,
		Id: p.PortNum}] = p

	sgi.PortList = append(sgi.PortList, p)

	// wait group used when stopping all the
	// State mahines associated with this port.
	// want to ensure that all routines are stopped
	// before proceeding with cleanup
	p.wg.Add(5)

	handle, err := pcap.OpenLive(p.IntfNum, 65536, false, 50*time.Millisecond)
	if err != nil {
		// failure here may be ok as this may be SIM
		if !strings.Contains(p.IntfNum, "SIM") {
			fmt.Println("Error creating pcap OpenLive handle for port", p.PortNum, p.IntfNum, err)
		}
		return p
	}
	fmt.Println("Creating Listener for intf", p.IntfNum)
	//p.LaPortLog(fmt.Sprintf("Creating Listener for intf", p.IntfNum))
	p.handle = handle
	src := gopacket.NewPacketSource(p.handle, layers.LayerTypeEthernet)
	in := src.Packets()
	// start rx routine
	LaRxMain(p.PortNum, in)
	fmt.Println("Rx Main Started for port", p.PortNum)

	// register the tx func
	sgi.LaSysGlobalRegisterTxCallback(p.IntfNum, TxViaLinuxIf)

	fmt.Println("Tx Callback Registered with")

	//fmt.Println("New Port:\n%#v", *p)

	return p
}

func (p *LaAggPort) EnableLogging(ena bool) {
	p.logEna = ena
}

func (p *LaAggPort) PortChannelGet() chan string {
	return p.portChan
}

// IsPortAdminEnabled will check if provisioned port enable
// State is enabled or disabled
func (p *LaAggPort) IsPortAdminEnabled() bool {
	return p.PortEnabled
}

func (p *LaAggPort) IsPortOperStatusUp() bool {
	p.LinkOperStatus = asicdGetPortLinkStatus(p.IntfNum)
	return p.LinkOperStatus
}

// IsPortEnabled will check if port is admin enabled
// and link is operationally up
func (p *LaAggPort) IsPortEnabled() bool {
	return p.IsPortAdminEnabled() && p.IsPortOperStatusUp()
}

func (p *LaAggPort) DelLaAggPort() {
	p.Stop()
	for _, sgi := range LacpSysGlobalInfoGet() {
		for Key, port := range sgi.PortMap {
			if port.PortNum == p.PortNum ||
				port.IntfNum == p.IntfNum {
				var a *LaAggregator
				var sysId LacpSystem
				if LaFindAggById(p.AggId, &a) {
					mac, _ := net.ParseMAC(a.Config.SystemIdMac)
					sysId.actor_System = convertNetHwAddressToSysIdKey(mac)
					sysId.Actor_System_priority = a.Config.SystemPriority
				}

				sgi := LacpSysGlobalInfoByIdGet(sysId)
				// remove the port from the port map
				delete(sgi.PortMap, Key)
				for i, delPort := range sgi.LacpSysGlobalAggPortListGet() {
					if delPort.PortNum == p.PortNum {
						sgi.PortList = append(sgi.PortList[:i], sgi.PortList[i+1:]...)
					}
				}
				return
			}
		}
	}
}

func (p *LaAggPort) Stop() {

	var a *LaAggregator
	var sysId LacpSystem
	if LaFindAggById(p.AggId, &a) {
		mac, _ := net.ParseMAC(a.Config.SystemIdMac)
		sysId.actor_System = convertNetHwAddressToSysIdKey(mac)
		sysId.Actor_System_priority = a.Config.SystemPriority
	}

	sgi := LacpSysGlobalInfoByIdGet(sysId)
	if sgi != nil {
		sgi.LaSysGlobalDeRegisterTxCallback(p.IntfNum)
	}
	// close rx/tx processing
	// TODO figure out why this call does not return
	if p.handle != nil {
		p.handle.Close()
		fmt.Println("RX/TX handle closed for port", p.PortNum)
	}

	// stop the State machines
	// TODO maybe run these in parrallel?
	if p.RxMachineFsm != nil {
		p.RxMachineFsm.Stop()
	} else {
		p.wg.Done()
	}

	if p.PtxMachineFsm != nil {
		p.PtxMachineFsm.Stop()
	} else {
		p.wg.Done()
	}
	if p.TxMachineFsm != nil {
		p.TxMachineFsm.Stop()
	} else {
		p.wg.Done()
	}
	if p.CdMachineFsm != nil {
		p.CdMachineFsm.Stop()
	} else {
		p.wg.Done()
	}
	if p.MuxMachineFsm != nil {
		p.MuxMachineFsm.Stop()
	} else {
		p.wg.Done()
	}
	// lets wait for all the State machines to have stopped
	p.wg.Wait()
	fmt.Println("All machines stopped for port", p.PortNum)
	close(p.portChan)

	// kill the logger for this port
	p.LacpDebug.logger.Info(fmt.Sprintf("Logger stopped for port\n", p.PortNum))
	p.LacpDebug.Stop()

}

//  BEGIN will initiate all the State machines
// and will send an event back to this caller
// to begin processing.
func (p *LaAggPort) BEGIN(restart bool) {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	// System in being initalized
	p.begin = true

	if !restart {
		// save off a shortcut to the log chan
		p.log = p.LacpDebug.LacpLogChan

		// start all the State machines
		// Order here matters as Rx machine
		// will send event to Mux machine
		// thus machine must be up and
		// running first
		// Mux Machine
		p.LacpMuxMachineMain()
		// Periodic Tx Machine
		p.LacpPtxMachineMain()
		// Churn Detection Machine
		p.LacpActorCdMachineMain()
		// Rx Machine
		p.LacpRxMachineMain()
		// Tx Machine
		p.LacpTxMachineMain()
	}
	// Rxm
	mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpRxmEventBegin,
		src: PortConfigModuleStr})
	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpPtxmEventBegin,
		src: PortConfigModuleStr})
	// Cdm
	mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpCdmEventBegin,
		src: PortConfigModuleStr})
	// Muxm
	mEvtChan = append(mEvtChan, p.MuxMachineFsm.MuxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpMuxmEventBegin,
		src: PortConfigModuleStr})
	// Txm
	mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpTxmEventBegin,
		src: PortConfigModuleStr})

	// call the begin event for each
	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, true)

	p.begin = false
}

// DistributeMachineEvents will distribute the events in parrallel
// to each machine
func (p *LaAggPort) DistributeMachineEvents(mec []chan LacpMachineEvent, e []LacpMachineEvent, waitForResponse bool) {

	length := len(mec)
	if len(mec) != len(e) {
		p.LaPortLog("LAPORT: Distributing of events failed")
		return
	}

	// send all begin events to each machine in parrallel
	for j := 0; j < length; j++ {
		go func(port *LaAggPort, w bool, idx int, machineEventChannel []chan LacpMachineEvent, event []LacpMachineEvent) {
			if w {
				event[idx].responseChan = p.portChan
			}
			event[idx].src = PortConfigModuleStr
			machineEventChannel[idx] <- event[idx]
		}(p, waitForResponse, j, mec, e)
	}

	if waitForResponse {
		i := 0
		// lets wait for all the machines to respond
		for {
			select {
			case mStr := <-p.portChan:
				i++
				p.LaPortLog(strings.Join([]string{"LAPORT:", mStr, "response received"}, " "))
				//fmt.Println("LAPORT: Waiting for response Delayed", length, "curr", i, time.Now())
				if i >= length {
					// 10/24/15 fixed hack by sending response after Machine.ProcessEvent
					// HACK, found that port is pre-empting the State machine callback return
					// lets delay for a short period to allow for event to be received
					// and other routines to process their events
					/*
						if p.logEna {
							time.Sleep(time.Millisecond * 3)
						} else {
							time.Sleep(time.Millisecond * 1)
						}
					*/
					return
				}
			}
		}
	}
}

// LaAggPortDisable will update the status on the port
// as well as inform the appropriate State machines of the
// State change
func (p *LaAggPort) LaAggPortDisable() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port Disabled")

	// port is disabled
	p.PortEnabled = false

	// Rxm
	if !p.portMoved {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventNotPortEnabledAndNotPortMoved,
			src: PortConfigModuleStr})
	}
	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpPtxmEventNotPortEnabled,
		src: PortConfigModuleStr})

	// Cdm
	mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpCdmEventNotPortEnabled,
		src: PortConfigModuleStr})

	// Txm
	mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpTxmEventLacpDisabled,
		src: PortConfigModuleStr})

	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, true)
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate State machines of the
// State change
// When this is called, it is assumed that all States are
// in their default State.
func (p *LaAggPort) LaAggPortEnabled() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port Enabled")

	// port is enabled
	p.PortEnabled = true

	// Rxm
	if p.lacpEnabled {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpEnabled,
			src: PortConfigModuleStr})
	} else {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpDisabled,
			src: PortConfigModuleStr})
	}

	// Ptxm
	if p.PtxMachineFsm.LacpPtxIsNoPeriodicExitCondition() {
		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpPtxmEventUnconditionalFallthrough,
			src: PortConfigModuleStr})
	}

	// Cdm
	if LacpStateIsSet(p.ActorOper.State, LacpStateSyncBit) {
		mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpCdmEventActorOperPortStateSyncOn,
			src: PortConfigModuleStr})
	}

	// Txm
	if p.lacpEnabled {
		mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpTxmEventLacpEnabled,
			src: PortConfigModuleStr})
	}

	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, true)
}

// LaAggPortLacpDisable will update the status on the port
// as well as inform the appropriate State machines of the
// State change
func (p *LaAggPort) LaAggPortLacpDisable() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port LACP Disabled")

	// port is disabled
	p.lacpEnabled = false

	// Rxm
	if p.PortEnabled {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpDisabled,
			src: PortConfigModuleStr})

		// Ptxm
		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpPtxmEventLacpDisabled,
			src: PortConfigModuleStr})

		// Txm, if lacp is disabled then should not transmit packets
		mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpTxmEventLacpDisabled,
			src: PortConfigModuleStr})

		// distribute the port disable event to various machines
		p.DistributeMachineEvents(mEvtChan, evt, true)
	}

	var a *LaAggregator
	var sysId LacpSystem
	if LaFindAggById(p.AggId, &a) {
		mac, _ := net.ParseMAC(a.Config.SystemIdMac)
		sysId.actor_System = convertNetHwAddressToSysIdKey(mac)
		sysId.Actor_System_priority = a.Config.SystemPriority
	}

	// port is no longer controlling lacp State
	sgi := LacpSysGlobalInfoByIdGet(sysId)
	p.actorAdmin.State = sgi.ActorStateDefaultParams.State
	p.ActorOper.State = sgi.ActorStateDefaultParams.State
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate State machines of the
// State change
func (p *LaAggPort) LaAggPortLacpEnabled(mode int) {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port LACP Enabled")

	// port can be added to aggregator
	LacpStateSet(&p.actorAdmin.State, LacpStateAggregationBit)

	// Activity mode
	if mode == LacpModeActive {
		LacpStateSet(&p.actorAdmin.State, LacpStateActivityBit)
	} else {
		LacpStateClear(&p.actorAdmin.State, LacpStateActivityBit)
	}

	if p.PortEnabled &&
		p.aggSelected == LacpAggSelected {
		// Rxm
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpEnabled,
			src: PortConfigModuleStr})

		// Ptxm
		if p.PtxMachineFsm.LacpPtxIsNoPeriodicExitCondition() {
			mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
			evt = append(evt, LacpMachineEvent{e: LacpPtxmEventUnconditionalFallthrough,
				src: PortConfigModuleStr})
		}

		// distribute the port disable event to various machines
		p.DistributeMachineEvents(mEvtChan, evt, true)
	}
	// port has lacp enabled
	p.lacpEnabled = true

}

func (p *LaAggPort) LaAggPortActorAdminInfoSet(sysIdMac [6]uint8, sysPrio uint16) {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.actorAdmin.System.actor_System = sysIdMac
	p.actorAdmin.System.Actor_System_priority = sysPrio

	p.aggSelected = LacpAggUnSelected

	if p.ModeGet() == LacpModeOn ||
		p.lacpEnabled == false ||
		p.PortEnabled == false {
		return
	}

	// partner info should be wrong so lets force sync to be off
	LacpStateClear(&p.PartnerOper.State, LacpStateSyncBit)

	mEvtChan = append(mEvtChan, p.MuxMachineFsm.MuxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpMuxmEventSelectedEqualUnselected,
		src: PortConfigModuleStr})

	// unselected event
	p.DistributeMachineEvents(mEvtChan, evt, true)

}

func (p *LaAggPort) TimeoutGet() time.Duration {
	return p.PtxMachineFsm.PeriodicTxTimerInterval
}

func (p *LaAggPort) ModeGet() int {
	return LacpModeGet(p.ActorOper.State, p.lacpEnabled)
}

func LacpCopyLacpPortInfoFromPkt(fromPortInfoPtr *layers.LACPPortInfo, toPortInfoPtr *LacpPortInfo) {
	toPortInfoPtr.Key = fromPortInfoPtr.Key
	toPortInfoPtr.port = fromPortInfoPtr.Port
	toPortInfoPtr.Port_pri = fromPortInfoPtr.PortPri
	toPortInfoPtr.State = fromPortInfoPtr.State
	toPortInfoPtr.System.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(fromPortInfoPtr.System.SystemId))
	toPortInfoPtr.System.LacpSystemActorSystemPrioritySet(fromPortInfoPtr.System.SystemPriority)
}

// LacpCopyLacpPortInfo:
// Copy the LacpPortInfo data from->to
func LacpCopyLacpPortInfo(fromPortInfoPtr *LacpPortInfo, toPortInfoPtr *LacpPortInfo) {
	toPortInfoPtr.Key = fromPortInfoPtr.Key
	toPortInfoPtr.port = fromPortInfoPtr.port
	toPortInfoPtr.Port_pri = fromPortInfoPtr.Port_pri
	toPortInfoPtr.State = fromPortInfoPtr.State
	toPortInfoPtr.System.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(fromPortInfoPtr.System.actor_System))
	toPortInfoPtr.System.LacpSystemActorSystemPrioritySet(fromPortInfoPtr.System.Actor_System_priority)
}

func LacpLacpPktPortInfoIsEqual(aPortInfoPtr *layers.LACPPortInfo, bPortInfoPtr *LacpPortInfo, StateBits uint8) bool {

	return aPortInfoPtr.System.SystemId == bPortInfoPtr.System.actor_System &&
		aPortInfoPtr.System.SystemPriority == bPortInfoPtr.System.Actor_System_priority &&
		aPortInfoPtr.Port == bPortInfoPtr.port &&
		aPortInfoPtr.PortPri == bPortInfoPtr.Port_pri &&
		aPortInfoPtr.Key == bPortInfoPtr.Key &&
		(LacpStateIsSet(aPortInfoPtr.State, StateBits) && LacpStateIsSet(bPortInfoPtr.State, StateBits))
}

// LacpLacpPortInfoIsEqual:
// Compare the LacpPortInfo data except be selective
// about the State bits that is being compared against
func LacpLacpPortInfoIsEqual(aPortInfoPtr *LacpPortInfo, bPortInfoPtr *LacpPortInfo, StateBits uint8) bool {

	return aPortInfoPtr.System.actor_System == bPortInfoPtr.System.actor_System &&
		aPortInfoPtr.System.Actor_System_priority == bPortInfoPtr.System.Actor_System_priority &&
		aPortInfoPtr.port == bPortInfoPtr.port &&
		aPortInfoPtr.Port_pri == bPortInfoPtr.Port_pri &&
		aPortInfoPtr.Key == bPortInfoPtr.Key &&
		(LacpStateIsSet(aPortInfoPtr.State, StateBits) && LacpStateIsSet(bPortInfoPtr.State, StateBits))
}
