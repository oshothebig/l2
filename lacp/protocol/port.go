// port
package lacp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"reflect"
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
	system   LacpSystem
	key      uint16
	port_pri uint16
	port     uint16
	state    uint8
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
	intfNum string

	// key
	key uint16

	// used to form portId
	portNum      uint16
	portPriority uint16

	AggId int

	//config LacpConfigInfo

	// Once selected reference to agg group will be made
	aggAttached *LaAggregator
	aggSelected int
	// unable to aggregate with other links in an agg
	operIndividual int
	lacpEnabled    bool
	// TRUE - Aggregation port is operable (MAC_Operational == True)
	// FALSE - otherwise
	portEnabled  bool
	portMoved    bool
	begin        bool
	actorChurn   bool
	partnerChurn bool
	readyN       bool

	macProperties PortProperties

	// determine whether a port is up or down
	linkOperStatus bool

	// administrative values for state described in 6.4.2.3
	actorAdmin   LacpPortInfo
	actorOper    LacpPortInfo
	partnerAdmin LacpPortInfo
	partnerOper  LacpPortInfo

	// state machines
	RxMachineFsm  *LacpRxMachine
	PtxMachineFsm *LacpPtxMachine
	TxMachineFsm  *LacpTxMachine
	CdMachineFsm  *LacpCdMachine
	MuxMachineFsm *LacpMuxMachine

	// will serialize state transition logging per port
	LacpDebug *LacpDebug

	// on configuration changes need to inform all state
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

// find a port from the global map table by portNum
func LaFindPortById(pId uint16, port **LaAggPort) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, p := range sgi.PortMap {
			if p.portNum == pId {
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

// find a port from the global map table by portNum
func LaFindPortByPortId(portId int, port **LaAggPort) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, p := range sgi.PortMap {
			if p.portId == portId {
				*port = p
				return true
			}
		}
	}
	return false
}

// LaFindPortByKey will find a port form the global map table by key
// index value should input 0 for the first value
func LaFindPortByKey(key uint16, index *int, port **LaAggPort) bool {
	var i int
	for _, sgi := range LacpSysGlobalInfoGet() {
		i = *index
		l := len(sgi.PortMap)
		for _, p := range sgi.PortMap {
			if i < l {
				if p.key == key {
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
		sysId.actor_system = convertNetHwAddressToSysIdKey(mac)
		sysId.actor_system_priority = a.Config.SystemPriority
	}

	sgi := LacpSysGlobalInfoByIdGet(sysId)

	p := &LaAggPort{
		portId:       LaConvertPortAndPriToPortId(config.Id, config.Prio),
		portNum:      config.Id,
		portPriority: config.Prio,
		intfNum:      config.IntfId,
		key:          config.Key,
		AggId:        0, // this should be set on config AddLaAggPortToAgg
		aggSelected:  LacpAggUnSelected,
		sysId:        config.SysId,
		begin:        true,
		portMoved:    false,
		lacpEnabled:  false,
		portEnabled:  config.Enable,
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
	p.actorAdmin.state = sgi.ActorStateDefaultParams.state
	p.actorAdmin.system.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(sgi.SystemDefaultParams.actor_system))
	p.actorAdmin.system.LacpSystemActorSystemPrioritySet(sgi.SystemDefaultParams.actor_system_priority)
	p.actorAdmin.key = p.key
	p.actorAdmin.port = p.portNum
	p.actorAdmin.port_pri = p.portPriority

	// default actor oper same as admin
	p.actorOper = p.actorAdmin

	// default partner admin
	p.partnerAdmin.state = sgi.PartnerStateDefaultParams.state
	// default partner oper same as admin
	p.partnerOper = p.partnerAdmin

	if config.Mode != LacpModeOn {
		p.lacpEnabled = true
	}

	// add port to port map
	sgi.PortMap[PortIdKey{Name: p.intfNum,
		Id: p.portNum}] = p

	// wait group used when stopping all the
	// state mahines associated with this port.
	// want to ensure that all routines are stopped
	// before proceeding with cleanup
	p.wg.Add(5)

	handle, err := pcap.OpenLive(p.intfNum, 65536, true, pcap.BlockForever)
	if err != nil {
		// failure here may be ok as this may be SIM
		if !strings.Contains(p.intfNum, "SIM") {
			fmt.Println("Error creating pcap OpenLive handle for port", p.portNum, p.intfNum, err)
		}
		return p
	}
	fmt.Println("Creating Listener for intf", p.intfNum)
	//p.LaPortLog(fmt.Sprintf("Creating Listener for intf", p.intfNum))
	p.handle = handle
	src := gopacket.NewPacketSource(p.handle, layers.LayerTypeEthernet)
	in := src.Packets()
	// start rx routine
	LaRxMain(p.portNum, in)
	fmt.Println("Rx Main Started")

	// register the tx func
	sgi.LaSysGlobalRegisterTxCallback(p.intfNum, TxViaLinuxIf)

	fmt.Println("Tx Callback Registered")

	//fmt.Println("New Port:\n%#v", *p)

	return p
}

func (p *LaAggPort) EnableLogging(ena bool) {
	p.logEna = ena
}

func (p *LaAggPort) PortChannelGet() chan string {
	return p.portChan
}

func (p *LaAggPort) DelLaAggPort() {
	p.Stop()
	for _, sgi := range LacpSysGlobalInfoGet() {
		for key, port := range sgi.PortMap {
			if port.portNum == p.portNum ||
				port.intfNum == p.intfNum {
				var a *LaAggregator
				var sysId LacpSystem
				if LaFindAggById(p.AggId, &a) {
					mac, _ := net.ParseMAC(a.Config.SystemIdMac)
					sysId.actor_system = convertNetHwAddressToSysIdKey(mac)
					sysId.actor_system_priority = a.Config.SystemPriority
				}

				sgi := LacpSysGlobalInfoByIdGet(sysId)
				// remove the port from the port map
				delete(sgi.PortMap, key)
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
		sysId.actor_system = convertNetHwAddressToSysIdKey(mac)
		sysId.actor_system_priority = a.Config.SystemPriority
	}

	sgi := LacpSysGlobalInfoByIdGet(sysId)
	if sgi != nil {
		sgi.LaSysGlobalDeRegisterTxCallback(p.intfNum)
	}
	// close rx/tx processing
	// TODO figure out why this call does not return
	if p.handle != nil {
		p.handle.Close()
		fmt.Println("RX/TX handle closed for port", p.portNum)
	}

	// stop the state machines
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
	// lets wait for all the state machines to have stopped
	p.wg.Wait()
	fmt.Println("All machines stopped for port", p.portNum)
	close(p.portChan)

	// kill the logger for this port
	p.LacpDebug.Stop()
	fmt.Println("Logger stopped for port", p.portNum)

}

//  BEGIN will initiate all the state machines
// and will send an event back to this caller
// to begin processing.
func (p *LaAggPort) BEGIN(restart bool) {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	// system in being initalized
	p.begin = true

	if !restart {
		// save off a shortcut to the log chan
		p.log = p.LacpDebug.LacpLogChan

		// start all the state machines
		// Order here matters as Rx machine
		// will send event to Mux machine
		// thus machine must be up and
		// running first
		// Mux Machine
		p.LacpMuxMachineMain()
		// Periodic Tx Machine
		p.LacpPtxMachineMain()
		// Churn Detection Machine
		p.LacpCdMachineMain()
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
					// HACK, found that port is pre-empting the state machine callback return
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
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortDisable() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port Disabled")

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

	// port is disabled
	p.portEnabled = false
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate state machines of the
// state change
// When this is called, it is assumed that all states are
// in their default state.
func (p *LaAggPort) LaAggPortEnabled() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port Enabled")

	// port is enabled
	p.portEnabled = true

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
	if LacpStateIsSet(p.actorOper.state, LacpStateSyncBit) {
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
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortLacpDisable() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port LACP Disabled")

	// port is disabled
	p.lacpEnabled = false

	// Rxm
	if p.portEnabled {
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
		sysId.actor_system = convertNetHwAddressToSysIdKey(mac)
		sysId.actor_system_priority = a.Config.SystemPriority
	}

	// port is no longer controlling lacp state
	sgi := LacpSysGlobalInfoByIdGet(sysId)
	p.actorAdmin.state = sgi.ActorStateDefaultParams.state
	p.actorOper.state = sgi.ActorStateDefaultParams.state
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortLacpEnabled(mode int) {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port LACP Enabled")

	// port can be added to aggregator
	LacpStateSet(&p.actorAdmin.state, LacpStateAggregationBit)

	// Activity mode
	if mode == LacpModeActive {
		LacpStateSet(&p.actorAdmin.state, LacpStateActivityBit)
	} else {
		LacpStateClear(&p.actorAdmin.state, LacpStateActivityBit)
	}

	if p.portEnabled &&
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

func (p *LaAggPort) TimeoutGet() time.Duration {
	return p.PtxMachineFsm.PeriodicTxTimerInterval
}

func (p *LaAggPort) ModeGet() int {
	return LacpModeGet(p.actorOper.state, p.lacpEnabled)
}

func LacpCopyLacpPortInfoFromPkt(fromPortInfoPtr *layers.LACPPortInfo, toPortInfoPtr *LacpPortInfo) {
	toPortInfoPtr.key = fromPortInfoPtr.Key
	toPortInfoPtr.port = fromPortInfoPtr.Port
	toPortInfoPtr.port_pri = fromPortInfoPtr.PortPri
	toPortInfoPtr.state = fromPortInfoPtr.State
	toPortInfoPtr.system.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(fromPortInfoPtr.System.SystemId))
	toPortInfoPtr.system.LacpSystemActorSystemPrioritySet(fromPortInfoPtr.System.SystemPriority)
}

// LacpCopyLacpPortInfo:
// Copy the LacpPortInfo data from->to
func LacpCopyLacpPortInfo(fromPortInfoPtr *LacpPortInfo, toPortInfoPtr *LacpPortInfo) {
	toPortInfoPtr.key = fromPortInfoPtr.key
	toPortInfoPtr.port = fromPortInfoPtr.port
	toPortInfoPtr.port_pri = fromPortInfoPtr.port_pri
	toPortInfoPtr.state = fromPortInfoPtr.state
	toPortInfoPtr.system.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(fromPortInfoPtr.system.actor_system))
	toPortInfoPtr.system.LacpSystemActorSystemPrioritySet(fromPortInfoPtr.system.actor_system_priority)
}

func LacpLacpPktPortInfoIsEqual(aPortInfoPtr *layers.LACPPortInfo, bPortInfoPtr *LacpPortInfo, stateBits uint8) bool {

	return reflect.DeepEqual(aPortInfoPtr.System.SystemId, bPortInfoPtr.system.actor_system) &&
		aPortInfoPtr.System.SystemPriority == bPortInfoPtr.system.actor_system_priority &&
		aPortInfoPtr.Port == bPortInfoPtr.port &&
		aPortInfoPtr.PortPri == bPortInfoPtr.port_pri &&
		aPortInfoPtr.Key == bPortInfoPtr.key &&
		(LacpStateIsSet(aPortInfoPtr.State, stateBits) && LacpStateIsSet(bPortInfoPtr.state, stateBits))
}

// LacpLacpPortInfoIsEqual:
// Compare the LacpPortInfo data except be selective
// about the state bits that is being compared against
func LacpLacpPortInfoIsEqual(aPortInfoPtr *LacpPortInfo, bPortInfoPtr *LacpPortInfo, stateBits uint8) bool {

	return reflect.DeepEqual(aPortInfoPtr.system.actor_system, bPortInfoPtr.system.actor_system) &&
		aPortInfoPtr.system.actor_system_priority == bPortInfoPtr.system.actor_system_priority &&
		aPortInfoPtr.port == bPortInfoPtr.port &&
		aPortInfoPtr.port_pri == bPortInfoPtr.port_pri &&
		aPortInfoPtr.key == bPortInfoPtr.key &&
		(LacpStateIsSet(aPortInfoPtr.state, stateBits) && LacpStateIsSet(bPortInfoPtr.state, stateBits))
}
