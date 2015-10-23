// port
package lacp

import (
	"strings"
	"utils/fsm"
)

type PortProperties struct {
	speed  int
	duplex int
	mtu    int
}

type LacpPortSystemInfo struct {
	systemPriority uint16
	systemId       [6]uint8
}

type LacpPortInfo struct {
	system   LacpPortSystemInfo
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
	key int

	// used to form portId
	portNum      int
	portPriority int

	aggId int
	// Once selected reference to agg group will be made
	aggAttached *LaAggregator
	aggSelected uint32
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

	// Version 2
	partnerLacpPduVersionNumber int
	enableLongPduXmit           bool
	// packet is 1 byte, but spec says save as int.
	// going to save as byte
	partnerVersion uint8
}

func (p *LaAggPort) LaPortLog(msg string) {
	if p.logEna {
		p.log <- msg
	}
}

// find a port from the global map table
func LaFindPortById(pId int, p *LaAggPort) bool {
	p, ok := gLacpSysGlobalInfo.PortMap[pId]
	return ok
}

// NewLaAggPort
// Allocate a new lag port, creating appropriate timers
func NewLaAggPort(port int, portPri int, intfNum string) *LaAggPort {
	p := &LaAggPort{
		portId:       (port | portPri<<16),
		portNum:      port,
		portPriority: portPri,
		intfNum:      intfNum,
		begin:        true,
		portMoved:    false,
		lacpEnabled:  false,
		portEnabled:  false,
		portChan:     make(chan string)}

	// default actor admin
	p.actorAdmin.state = gLacpSysGlobalInfo.ActorStateDefaultParams.state
	// default actor oper same as admin
	p.actorOper = p.actorAdmin

	// default partner admin
	p.partnerAdmin.state = gLacpSysGlobalInfo.PartnerStateDefaultParams.state
	// default partner oper same as admin
	p.partnerOper = p.partnerAdmin

	// add port to port map
	gLacpSysGlobalInfo.PortMap[p.portNum] = p

	return p
}

func DelLaAggPort(p *LaAggPort) {
	p.Stop()
	// remove the port from the port map
	delete(gLacpSysGlobalInfo.PortMap, p.portNum)
}

func (p *LaAggPort) Stop() {
	// stop the state machines
	p.RxMachineFsm.Stop()
	p.PtxMachineFsm.Stop()
	p.TxMachineFsm.Stop()
	p.CdMachineFsm.Stop()
	p.MuxMachineFsm.Stop()
	close(p.portChan)
}

//  BEGIN will initiate all the state machines
// and will send an event back to this caller
// to begin processing.
func (p *LaAggPort) BEGIN(restart bool) {
	mEvtChan := make([]chan fsm.Event, 0)
	evt := make([]fsm.Event, 0)

	// system in being initalized
	p.begin = true

	if !restart {
		// MUST BE CALLED FIRST!!!
		p.LacpDebugEventLogMain()

		// save off a shortcut to the log chan
		p.log = p.LacpDebug.LacpLogChan
		p.logEna = true

		// start all the state machines
		p.LacpRxMachineMain()

		// No begin event
		p.LacpTxMachineMain()

		p.LacpPtxMachineMain()

		p.LacpCdMachineMain()

		p.LacpMuxMachineMain()
	}
	// Rxm
	mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
	evt = append(evt, LacpRxmEventBegin)
	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpPtxmEventBegin)
	// Cdm
	mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
	evt = append(evt, LacpCdmEventBegin)
	// Muxm
	mEvtChan = append(mEvtChan, p.MuxMachineFsm.MuxmEvents)
	evt = append(evt, LacpMuxmEventBegin)

	// call the begin event for each
	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, true)

	p.begin = false
}

// DistributeMachineEvents will distribute the events in parrallel
// to each machine
func (p *LaAggPort) DistributeMachineEvents(mec []chan fsm.Event, e []fsm.Event, waitForResponse bool) {

	length := len(mec)
	if len(mec) != len(e) {
		p.LaPortLog("LAPORT: Distributing of events failed")
		return
	}

	for j := 0; j < length; j++ {
		go func(idx int, machineEventChannel []chan fsm.Event, event []fsm.Event) {
			machineEventChannel[idx] <- event[idx]
		}(j, mec, e)
	}

	if waitForResponse {
		// lets wait for all the machines to respond
		for i := 0; i < length; i++ {
			select {
			case mStr := <-p.portChan:
				p.LaPortLog(strings.Join([]string{"LAPORT:", mStr, "running"}, " "))
			}
		}
	}
}

// LaAggPortDisable will update the status on the port
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortDisable() {
	mEvtChan := make([]chan fsm.Event, 0)
	evt := make([]fsm.Event, 0)

	p.LaPortLog("LAPORT: Port Disabled")

	// Rxm
	if !p.portMoved {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpRxmEventNotPortEnabledAndNotPortMoved)
	}
	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpPtxmEventNotPortEnabled)

	// Cdm
	mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
	evt = append(evt, LacpCdmEventNotPortEnabled)

	// Txm
	mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
	evt = append(evt, LacpTxmEventLacpDisabled)

	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, false)

	// port is disabled
	p.portEnabled = false
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate state machines of the
// state change
// When this is called, it is assumed that all states are
// in their default state.
func (p *LaAggPort) LaAggPortEnabled() {
	mEvtChan := make([]chan fsm.Event, 0)
	evt := make([]fsm.Event, 0)

	p.LaPortLog("LAPORT: Port Enabled")

	// restart state machines, LACP has been enabled
	// TODO: is this necessary all states should be in defaulted mode
	// if run into problems then check states of all machines
	// may need to run BEGIN again
	//p.BEGIN(true)

	// Rxm
	if p.lacpEnabled {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpRxmEventPortEnabledAndLacpEnabled)
	} else {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpRxmEventPortEnabledAndLacpDisabled)
	}

	// Ptxm
	if p.PtxMachineFsm.LacpPtxIsNoPeriodicExitCondition() {
		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, LacpPtxmEventUnconditionalFallthrough)
	}

	// Cdm
	if LacpStateIsSet(p.actorOper.state, LacpStateSyncBit) {
		mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
		evt = append(evt, LacpCdmEventActorOperPortStateSyncOn)
	}

	// Txm
	if p.lacpEnabled {
		mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
		evt = append(evt, LacpTxmEventLacpEnabled)
	}

	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, false)

	// port is disabled
	p.portEnabled = true
}

// LaAggPortLacpDisable will update the status on the port
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortLacpDisable() {
	mEvtChan := make([]chan fsm.Event, 0)
	evt := make([]fsm.Event, 0)

	p.LaPortLog("LAPORT: Port LACP Disabled")

	// port is disabled
	p.lacpEnabled = false

	// Rxm
	if p.portEnabled {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpRxmEventPortEnabledAndLacpDisabled)
	}

	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpPtxmEventLacpDisabled)

	// Txm, if lacp is disabled then should not transmit packets
	mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
	evt = append(evt, LacpTxmEventLacpDisabled)

	// distribute the port disable event to various machines
	p.DistributeMachineEvents(mEvtChan, evt, false)

	// port is no longer controlling lacp state
	p.actorAdmin.state = gLacpSysGlobalInfo.ActorStateDefaultParams.state
	p.actorOper.state = gLacpSysGlobalInfo.ActorStateDefaultParams.state
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortLacpEnabled() {
	mEvtChan := make([]chan fsm.Event, 0)
	evt := make([]fsm.Event, 0)

	p.LaPortLog("LAPORT: Port LACP Enabled")

	// port can be added to aggregator
	LacpStateSet(p.actorAdmin.state, LacpStateActivityBit)

	// restart state machines, LACP has been enabled
	// TODO: is this necessary all states should be in defaulted mode
	// if run into problems then check states of all machines
	// may need to run BEGIN again
	//p.BEGIN(true)

	if p.portEnabled {
		// Rxm
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpRxmEventPortEnabledAndLacpEnabled)

		// Ptxm
		if p.PtxMachineFsm.LacpPtxIsNoPeriodicExitCondition() {
			mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
			evt = append(evt, LacpPtxmEventUnconditionalFallthrough)
		}

		// distribute the port disable event to various machines
		p.DistributeMachineEvents(mEvtChan, evt, false)
	}
	// port is disabled
	p.lacpEnabled = true
}

// LacpCopyLacpPortInfo:
// Copy the LacpPortInfo data from->to
func LacpCopyLacpPortInfo(fromPortInfoPtr *LacpPortInfo, toPortInfoPtr *LacpPortInfo) {
	toPortInfoPtr.key = fromPortInfoPtr.key
	toPortInfoPtr.port = fromPortInfoPtr.port
	toPortInfoPtr.port_pri = fromPortInfoPtr.port_pri
	toPortInfoPtr.state = fromPortInfoPtr.state
	toPortInfoPtr.system.systemId = fromPortInfoPtr.system.systemId
	toPortInfoPtr.system.systemPriority = fromPortInfoPtr.system.systemPriority
}

// LacpLacpPortInfoIsEqual:
// Compare the LacpPortInfo data except be selective
// about the state bits that is being compared against
func LacpLacpPortInfoIsEqual(aPortInfoPtr *LacpPortInfo, bPortInfoPtr *LacpPortInfo, stateBits uint8) bool {

	return aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[0] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[1] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[2] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[3] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[4] &&
		aPortInfoPtr.system.systemId[0] == bPortInfoPtr.system.systemId[5] &&
		aPortInfoPtr.system.systemPriority == bPortInfoPtr.system.systemPriority &&
		aPortInfoPtr.port == bPortInfoPtr.port &&
		aPortInfoPtr.port_pri == bPortInfoPtr.port_pri &&
		aPortInfoPtr.key == bPortInfoPtr.key &&
		(LacpStateIsSet(aPortInfoPtr.state, stateBits) && LacpStateIsSet(bPortInfoPtr.state, stateBits))
}
