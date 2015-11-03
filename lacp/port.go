// port
package lacp

import (
	//"fmt"
	"strings"
	//"time"
)

type PortProperties struct {
	Mac    [6]uint8
	speed  int
	duplex int
	mtu    int
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

	aggId int

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

	// Version 2
	partnerLacpPduVersionNumber int
	enableLongPduXmit           bool
	// packet is 1 byte, but spec says save as int.
	// going to save as byte
	partnerVersion uint8

	sysId [6]uint8
}

func (p *LaAggPort) LaPortLog(msg string) {
	if p.logEna {
		p.log <- msg
	}
}

// find a port from the global map table by portNum
func LaFindPortById(pId uint16, port **LaAggPort) bool {
	for _, sgi := range gLacpSysGlobalInfo {
		for _, p := range sgi.PortMap {
			if p.portNum == pId {
				*port = p
				return true
			}
		}
	}
	return false
}

// find a port form the global map table by key
func LaFindPortByKey(key uint16, port **LaAggPort) bool {
	for _, sgi := range gLacpSysGlobalInfo {
		for _, p := range sgi.PortMap {
			if p.key == key {
				*port = p
				return true
			}
		}
	}
	return false
}

// NewLaAggPort
// Allocate a new lag port, creating appropriate timers
func NewLaAggPort(config *LaAggPortConfig) *LaAggPort {
	p := &LaAggPort{
		portId:        int(config.Id | config.Prio<<16),
		portNum:       config.Id,
		portPriority:  config.Prio,
		intfNum:       config.IntfId,
		macProperties: config.Properties,
		key:           config.Key,
		aggSelected:   LacpAggUnSelected,
		sysId:         config.sysId,
		begin:         true,
		portMoved:     false,
		lacpEnabled:   false,
		portEnabled:   config.Enable,
		logEna:        config.traceEna,
		portChan:      make(chan string)}

	// default actor admin
	//fmt.Println(config.sysId, gLacpSysGlobalInfo[config.sysId])
	p.actorAdmin.state = gLacpSysGlobalInfo[config.sysId].ActorStateDefaultParams.state
	p.actorAdmin.system.LacpSystemActorSystemIdSet(gLacpSysGlobalInfo[config.sysId].SystemDefaultParams.actor_system)
	p.actorAdmin.system.LacpSystemActorSystemPrioritySet(gLacpSysGlobalInfo[config.sysId].SystemDefaultParams.actor_system_priority)
	p.actorAdmin.key = p.key
	p.actorAdmin.port = p.portNum
	p.actorAdmin.port_pri = p.portPriority

	// default actor oper same as admin
	p.actorOper = p.actorAdmin

	// default partner admin
	p.partnerAdmin.state = gLacpSysGlobalInfo[config.sysId].PartnerStateDefaultParams.state
	// default partner oper same as admin
	p.partnerOper = p.partnerAdmin

	if config.Mode != LacpModeOn {
		p.lacpEnabled = true
	}

	// add port to port map
	gLacpSysGlobalInfo[config.sysId].PortMap[p.portNum] = p

	// Start Port Logger
	p.LacpDebugEventLogMain()

	//fmt.Printf("%#v", *p)

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
	for _, sgi := range gLacpSysGlobalInfo {
		for pId, p := range sgi.PortMap {
			if pId == p.portNum {
				// remove the port from the port map
				delete(gLacpSysGlobalInfo[p.sysId].PortMap, p.portNum)
				return
			}
		}
	}
}

func (p *LaAggPort) Stop() {
	// stop the state machines
	if p.RxMachineFsm != nil {
		p.RxMachineFsm.Stop()
	}
	if p.PtxMachineFsm != nil {
		p.PtxMachineFsm.Stop()
	}
	if p.TxMachineFsm != nil {
		p.TxMachineFsm.Stop()
	}
	if p.CdMachineFsm != nil {
		p.CdMachineFsm.Stop()
	}
	if p.MuxMachineFsm != nil {
		p.MuxMachineFsm.Stop()
	}
	close(p.portChan)
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
	evt = append(evt, LacpMachineEvent{e: LacpRxmEventBegin})
	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpPtxmEventBegin})
	// Cdm
	mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpCdmEventBegin})
	// Muxm
	mEvtChan = append(mEvtChan, p.MuxMachineFsm.MuxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpMuxmEventBegin})
	// Txm
	mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpTxmEventBegin})

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
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventNotPortEnabledAndNotPortMoved})
	}
	// Ptxm
	mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpPtxmEventNotPortEnabled})

	// Cdm
	mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpCdmEventNotPortEnabled})

	// Txm
	mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
	evt = append(evt, LacpMachineEvent{e: LacpTxmEventLacpDisabled})

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
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpEnabled})
	} else {
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpDisabled})
	}

	// Ptxm
	if p.PtxMachineFsm.LacpPtxIsNoPeriodicExitCondition() {
		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpPtxmEventUnconditionalFallthrough})
	}

	// Cdm
	if LacpStateIsSet(p.actorOper.state, LacpStateSyncBit) {
		mEvtChan = append(mEvtChan, p.CdMachineFsm.CdmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpCdmEventActorOperPortStateSyncOn})
	}

	// Txm
	if p.lacpEnabled {
		mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpTxmEventLacpEnabled})
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
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpDisabled})

		// Ptxm
		mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpPtxmEventLacpDisabled})

		// Txm, if lacp is disabled then should not transmit packets
		mEvtChan = append(mEvtChan, p.TxMachineFsm.TxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpTxmEventLacpDisabled})

		// distribute the port disable event to various machines
		p.DistributeMachineEvents(mEvtChan, evt, true)
	}

	// port is no longer controlling lacp state
	p.actorAdmin.state = gLacpSysGlobalInfo[p.sysId].ActorStateDefaultParams.state
	p.actorOper.state = gLacpSysGlobalInfo[p.sysId].ActorStateDefaultParams.state
}

// LaAggPortEnabled will update the status on the port
// as well as inform the appropriate state machines of the
// state change
func (p *LaAggPort) LaAggPortLacpEnabled() {
	mEvtChan := make([]chan LacpMachineEvent, 0)
	evt := make([]LacpMachineEvent, 0)

	p.LaPortLog("LAPORT: Port LACP Enabled")

	// port can be added to aggregator
	LacpStateSet(&p.actorAdmin.state, LacpStateActivityBit)

	if p.portEnabled &&
		p.aggSelected == LacpAggSelected {
		// Rxm
		mEvtChan = append(mEvtChan, p.RxMachineFsm.RxmEvents)
		evt = append(evt, LacpMachineEvent{e: LacpRxmEventPortEnabledAndLacpEnabled})

		// Ptxm
		if p.PtxMachineFsm.LacpPtxIsNoPeriodicExitCondition() {
			mEvtChan = append(mEvtChan, p.PtxMachineFsm.PtxmEvents)
			evt = append(evt, LacpMachineEvent{e: LacpPtxmEventUnconditionalFallthrough})
		}

		// distribute the port disable event to various machines
		p.DistributeMachineEvents(mEvtChan, evt, true)
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
	toPortInfoPtr.system.LacpSystemActorSystemIdSet(fromPortInfoPtr.system.actor_system)
	toPortInfoPtr.system.LacpSystemActorSystemPrioritySet(fromPortInfoPtr.system.actor_system_priority)
}

// LacpLacpPortInfoIsEqual:
// Compare the LacpPortInfo data except be selective
// about the state bits that is being compared against
func LacpLacpPortInfoIsEqual(aPortInfoPtr *LacpPortInfo, bPortInfoPtr *LacpPortInfo, stateBits uint8) bool {

	return aPortInfoPtr.system.actor_system[0] == bPortInfoPtr.system.actor_system[0] &&
		aPortInfoPtr.system.actor_system[1] == bPortInfoPtr.system.actor_system[1] &&
		aPortInfoPtr.system.actor_system[2] == bPortInfoPtr.system.actor_system[2] &&
		aPortInfoPtr.system.actor_system[3] == bPortInfoPtr.system.actor_system[3] &&
		aPortInfoPtr.system.actor_system[4] == bPortInfoPtr.system.actor_system[4] &&
		aPortInfoPtr.system.actor_system[5] == bPortInfoPtr.system.actor_system[5] &&
		aPortInfoPtr.system.actor_system_priority == bPortInfoPtr.system.actor_system_priority &&
		aPortInfoPtr.port == bPortInfoPtr.port &&
		aPortInfoPtr.port_pri == bPortInfoPtr.port_pri &&
		aPortInfoPtr.key == bPortInfoPtr.key &&
		(LacpStateIsSet(aPortInfoPtr.state, stateBits) && LacpStateIsSet(bPortInfoPtr.state, stateBits))
}
