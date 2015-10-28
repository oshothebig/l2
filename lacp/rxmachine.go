// rxmachine
package lacp

import (
	//"fmt"
	"strconv"
	"strings"
	"time"
	"utils/fsm"
)

const RxMachineModuleStr = "Rx Machine"

// rxm states
/*
const LacpRxmNone = "LacpRxmNone"    // not apart of the state machine, but used as an initial state
const LacpRxmInitialize = "LacpRxmInitialize"
const LacpRxmPortDisable = "LacpRxmPortDisable"
const LacpRxmExpired = "LacpRxmExpired"
const LacpRxmLacpDisabled = "LacpRxmLacpDisabled"
const LacpRxmDefaulted = "LacpRxmDefaulted"
const LacpRxmCurrent = "LacpRxmCurrent"
*/
const (
	LacpRxmStateNone = iota + 1
	LacpRxmStateInitialize
	LacpRxmStatePortDisabled
	LacpRxmStateExpired
	LacpRxmStateLacpDisabled
	LacpRxmStateDefaulted
	LacpRxmStateCurrent
)

var RxmStateStrMap map[fsm.State]string

func RxMachineStrStateMapCreate() {
	RxmStateStrMap = make(map[fsm.State]string)
	RxmStateStrMap[LacpRxmStateNone] = "LacmRxmNone"
	RxmStateStrMap[LacpRxmStateInitialize] = "LacpRxmStateInitialize"
	RxmStateStrMap[LacpRxmStatePortDisabled] = "LacpRxmStatePortDisabled"
	RxmStateStrMap[LacpRxmStateExpired] = "LacpRxmStateExpired"
	RxmStateStrMap[LacpRxmStateLacpDisabled] = "LacpRxmStateLacpDisabled"
	RxmStateStrMap[LacpRxmStateDefaulted] = "LacpRxmStateDefaulted"
	RxmStateStrMap[LacpRxmStateCurrent] = "LacpRxmStateCurrent"
}

// rxm events
/*
const LacpRxmBeginEvent = "LacpRxmBeginEvent"
const LacpRxmPortMovedEvent = "LacpRxmPortMovedEvent"
const LacpRxmCurrentWhileTimerExpiredEvent = "LacpRxmCurrentWhileTimerExpiredEvent"
const LacpRxmLacpDisabledEvent = "LacpRxmLacpDisabledEvent"
const LacpRxmLacpEnabledEvent = "LacpRxmLacpEnabledEvent"
const LacpRxmLacpPortDisabledEvent = "LacpRxmLacpPortDisabledEvent"
const LacpRxmLacpPktRxEvent = "LacpRxmLacpPktRxEvent"
*/
const (
	LacpRxmEventBegin = iota + 1
	LacpRxmEventUnconditionalFallthrough
	LacpRxmEventNotPortEnabledAndNotPortMoved
	LacpRxmEventPortMoved
	LacpRxmEventPortEnabledAndLacpEnabled
	LacpRxmEventPortEnabledAndLacpDisabled
	LacpRxmEventCurrentWhileTimerExpired
	LacpRxmEventLacpEnabled
	LacpRxmEventLacpPktRx
	LacpRxmEventKillSignal
)

type LacpRxLacpPdu struct {
	pdu          *LacpPdu
	src          string
	responseChan chan string
}

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpRxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *LaAggPort

	// debug log
	log chan string

	// timer interval
	currentWhileTimerTimeout time.Duration

	// timers
	currentWhileTimer *time.Timer

	// machine specific events
	RxmEvents          chan LacpMachineEvent
	RxmPktRxEvent      chan LacpRxLacpPdu
	RxmKillSignalEvent chan bool
	RxmLogEnableEvent  chan bool
}

func (rxm *LacpRxMachine) PrevState() fsm.State { return rxm.PreviousState }

// PrevStateSet will set the previous state
func (rxm *LacpRxMachine) PrevStateSet(s fsm.State) { rxm.PreviousState = s }

// Stop should clean up all resources
func (rxm *LacpRxMachine) Stop() {
	rxm.CurrentWhileTimerStop()

	// stop the go routine
	rxm.RxmKillSignalEvent <- true

	close(rxm.RxmEvents)
	close(rxm.RxmPktRxEvent)
	close(rxm.RxmKillSignalEvent)
	close(rxm.RxmLogEnableEvent)

}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpRxMachine(port *LaAggPort) *LacpRxMachine {
	rxm := &LacpRxMachine{
		p:                  port,
		log:                port.LacpDebug.LacpLogChan,
		PreviousState:      LacpRxmStateNone,
		RxmEvents:          make(chan LacpMachineEvent),
		RxmPktRxEvent:      make(chan LacpRxLacpPdu),
		RxmKillSignalEvent: make(chan bool),
		RxmLogEnableEvent:  make(chan bool)}

	port.RxMachineFsm = rxm

	// create then stop
	rxm.CurrentWhileTimerStart()
	rxm.CurrentWhileTimerStop()

	return rxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine.
func (rxm *LacpRxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if rxm.Machine == nil {
		rxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	rxm.Machine.Rules = r
	rxm.Machine.Curr = &LacpStateEvent{
		strStateMap: RxmStateStrMap,
		logEna:      rxm.p.logEna,
		logger:      rxm.LacpRxmLog,
		owner:       RxMachineModuleStr,
	}

	return rxm.Machine
}

// LacpRxMachineInitialize function to be called after
// state transition to INITIALIZE
func (rxm *LacpRxMachine) LacpRxMachineInitialize(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	// set the agg as being unselected
	//p.aggSelected = LacpAggUnSelected
	if p.MuxMachineFsm != nil {
		p.MuxMachineFsm.MuxmEvents <- LacpMachineEvent{e: LacpMuxmEventSelectedEqualUnselected}
	}

	// Record default params
	rxm.recordDefault()

	// Actor Port Oper State Expired = False
	LacpStateClear(&p.actorOper.state, LacpStateExpiredBit)

	// set the port moved to false
	p.portMoved = false

	// next state
	return LacpRxmStateInitialize
}

// LacpRxMachineExpired function to be called after
// state transition to PORT_DISABLED
func (rxm *LacpRxMachine) LacpRxMachinePortDisabled(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	// Partner Port Oper State Sync = False
	LacpStateClear(&p.partnerOper.state, LacpStateSyncBit)

	return LacpRxmStatePortDisabled
}

// LacpRxMachineExpired function to be called after
// state transition to EXPIRED
func (rxm *LacpRxMachine) LacpRxMachineExpired(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	// Partner Port Oper State Sync = FALSE
	LacpStateClear(&p.partnerOper.state, LacpStateSyncBit)

	// Short timeout
	LacpStateSet(&p.partnerOper.state, LacpStateTimeoutBit)

	// Set the Short timeout
	rxm.CurrentWhileTimerTimeoutSet(LacpShortTimeoutTime)

	// Start the Current While timer
	rxm.CurrentWhileTimerStart()

	// Actor Port Oper State Expired = TRUE
	LacpStateSet(&p.actorOper.state, LacpStateExpiredBit)

	return LacpRxmStateExpired
}

// LacpRxMachineLacpDisabled function to be called after
// state transition to LACP_DISABLED
func (rxm *LacpRxMachine) LacpRxMachineLacpDisabled(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	// Unselect the aggregator
	//p.aggSelected = LacpAggUnSelected
	if p.MuxMachineFsm != nil {
		p.MuxMachineFsm.MuxmEvents <- LacpMachineEvent{e: LacpMuxmEventSelectedEqualUnselected}
	}

	// setup the default params
	rxm.recordDefault()

	// Partner Port Oper State Aggregation = FALSE
	LacpStateClear(&p.partnerOper.state, LacpStateAggregationBit)

	// Actor Port Oper State Expired = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateExpiredBit)

	return LacpRxmStateLacpDisabled
}

// LacpRxMachineDefaulted function to be called after
// state transition to DEFAULTED
func (rxm *LacpRxMachine) LacpRxMachineDefaulted(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	//lacpPduInfo := data.(LacpPdu)

	// Updated the default selected state
	rxm.updateDefaultSelected()

	// Record the default partner info
	rxm.recordDefault()

	// Actor Port Oper State Expired = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateExpiredBit)

	return LacpRxmStateDefaulted
}

// LacpRxMachineCurrent function to be called after
// state transition to CURRENT
func (rxm *LacpRxMachine) LacpRxMachineCurrent(m fsm.Machine, data interface{}) fsm.State {
	p := rxm.p

	// Version 1, V2 will require a serialize/deserialize routine since TLV's are involved
	lacpPduInfo := data.(*LacpPdu)

	// update selection logic
	rxm.updateSelected(lacpPduInfo)

	// update the ntt
	ntt := rxm.updateNTT(lacpPduInfo)

	// Version 2 or higher check
	if LacpActorSystemLacpVersion >= 0x2 {
		rxm.recordVersionNumber(lacpPduInfo)
	}

	// record the current packet state
	rxm.recordPDU(lacpPduInfo)

	// Current while should already be set to
	// Actors Oper value of Timeout, lets check
	// anyways
	if timeoutTime, ok := rxm.CurrentWhileTimerValid(); !ok {
		rxm.CurrentWhileTimerTimeoutSet(timeoutTime)
	}
	// lets kick off the Current While Timer
	rxm.CurrentWhileTimerStart()

	// Actor_Oper_Port_Sate.Expired = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateExpiredBit)

	if ntt == true && p.TxMachineFsm != nil {
		// update ntt, which should trigger a packet transmit
		p.TxMachineFsm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventNtt}
	}

	return LacpRxmStateCurrent
}

func LacpRxMachineFSMBuild(p *LaAggPort) *LacpRxMachine {

	RxMachineStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	rxm := NewLacpRxMachine(p)

	//BEGIN -> INIT
	rules.AddRule(LacpRxmStateNone, LacpRxmEventBegin, rxm.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventBegin, rxm.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventBegin, rxm.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateLacpDisabled, LacpRxmEventBegin, rxm.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateDefaulted, LacpRxmEventBegin, rxm.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventBegin, rxm.LacpRxMachineInitialize)
	// INIT -> PORT_DISABLE
	rules.AddRule(LacpRxmStateInitialize, LacpRxmEventUnconditionalFallthrough, rxm.LacpRxMachinePortDisabled)
	// NOT PORT ENABLED  && NOT PORT MOVED
	// All states transition to this state
	rules.AddRule(LacpRxmStateInitialize, LacpRxmEventNotPortEnabledAndNotPortMoved, rxm.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventNotPortEnabledAndNotPortMoved, rxm.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateLacpDisabled, LacpRxmEventNotPortEnabledAndNotPortMoved, rxm.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateDefaulted, LacpRxmEventNotPortEnabledAndNotPortMoved, rxm.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventNotPortEnabledAndNotPortMoved, rxm.LacpRxMachinePortDisabled)
	// PORT MOVED -> INIT
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventPortMoved, rxm.LacpRxMachineInitialize)
	// PORT ENABLED && LACP ENABLED
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventPortEnabledAndLacpEnabled, rxm.LacpRxMachineExpired)
	// PORT ENABLED && LACP DISABLED
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventPortEnabledAndLacpDisabled, rxm.LacpRxMachineLacpDisabled)
	// CURRENT WHILE TIMER EXPIRED
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventCurrentWhileTimerExpired, rxm.LacpRxMachineDefaulted)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventCurrentWhileTimerExpired, rxm.LacpRxMachineExpired)
	// LACP ENABLED
	rules.AddRule(LacpRxmStateLacpDisabled, LacpRxmEventLacpEnabled, rxm.LacpRxMachinePortDisabled)
	// PKT RX
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventLacpPktRx, rxm.LacpRxMachineCurrent)
	rules.AddRule(LacpRxmStateDefaulted, LacpRxmEventLacpPktRx, rxm.LacpRxMachineCurrent)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventLacpPktRx, rxm.LacpRxMachineCurrent)

	// Create a new FSM and apply the rules
	rxm.Apply(&rules)

	return rxm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpRxMachineMain() {

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.12 Receive Machine
	rxm := LacpRxMachineFSMBuild(p)

	// set the inital state
	rxm.Machine.Start(rxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpRxMachine) {
		m.LacpRxmLog("RXM: Machine Start")
		for {
			select {
			case <-m.RxmKillSignalEvent:
				m.LacpRxmLog("RXM: Machine End")
				return

			case <-m.currentWhileTimer.C:
				m.LacpRxmLog("RXM: Current While Timer Expired")
				m.Machine.ProcessEvent(RxMachineModuleStr, LacpRxmEventCurrentWhileTimerExpired, nil)

			case event := <-m.RxmEvents:
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv == nil {
					p := m.p
					/* continue state transition */
					if m.Machine.Curr.CurrentState() == LacpRxmStateInitialize {
						rv = m.Machine.ProcessEvent(RxMachineModuleStr, LacpRxmEventUnconditionalFallthrough, nil)
					}
					if rv == nil &&
						m.Machine.Curr.CurrentState() == LacpRxmStatePortDisabled &&
						p.lacpEnabled == true &&
						p.portEnabled == true {
						rv = m.Machine.ProcessEvent(RxMachineModuleStr, LacpRxmEventPortEnabledAndLacpEnabled, nil)
					}
					if rv == nil &&
						m.Machine.Curr.CurrentState() == LacpRxmStatePortDisabled &&
						p.lacpEnabled == false &&
						p.portEnabled == true {
						rv = m.Machine.ProcessEvent(RxMachineModuleStr, LacpRxmEventPortEnabledAndLacpDisabled, nil)
					}
				}

				if rv != nil {
					m.LacpRxmLog(strings.Join([]string{error.Error(rv), event.src, RxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.responseChan != nil {
					SendResponse(RxMachineModuleStr, event.responseChan)
				}
			case rx := <-m.RxmPktRxEvent:
				//fmt.Println(*(rx.pdu))
				// lets check if the port has moved
				if m.CheckPortMoved(&p.partnerOper, &(rx.pdu.actor.info)) {
					m.p.portMoved = true
					m.Machine.ProcessEvent(RxModuleStr, LacpRxmEventPortMoved, nil)
				} else {
					// If you rx a packet must be in one
					// of 3 states
					// Expired/Defaulted/Current. each
					// state will transition to current
					// all other states should be ignored.
					m.Machine.ProcessEvent(RxModuleStr, LacpRxmEventLacpPktRx, rx.pdu)
				}

				// respond to caller if necessary so that we don't have a deadlock
				if rx.responseChan != nil {
					SendResponse(RxMachineModuleStr, rx.responseChan)
				}

			case ena := <-m.RxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)

			}
		}
	}(rxm)
}

// handleRxFrame:
// TBD: First entry point of the raw ethernet frame
//func handleRxFrame(port int, pdu []bytes) {

// TODO
//	lacp := LacpPdu()
//	err := binary.Read(pdu, binary.BigEndian, &lacp)
//	if err != nil {
//		panic(err)
//	}
//}

// recordPDU: 802.1ax Section 6.4.9
//
// Record actor informatio from the packet
// Clear Defaulted Actor Operational state
// Determine Partner Operational Sync state
func (rxm *LacpRxMachine) recordPDU(lacpPduInfo *LacpPdu) {

	p := rxm.p

	// Record Actor info from packet - store in parter operational
	// Port Number, Port Priority, System, System Priority
	// Key, state variables
	LacpCopyLacpPortInfo(&lacpPduInfo.actor.info, &p.partnerOper)

	// Set Actor Oper port state Defaulted to FALSE
	LacpStateClear(&p.actorOper.state, LacpStateDefaultedBit)

	// Set Partner Oper port state Sync state to
	// TRUE if the (1) or (2) is true:
	//
	// 1) Rx pdu: (Partner Port, Partner Port Priority, Partner
	// System, Partner System Priority, Partner Key,
	// Partner state Aggregation) vs 	cooresponding Operational
	// parameters of the Actor and Actor state Sync is TRUE and (3)
	//
	// 2) Rx pdu: Value of Actor state aggregation is FALSE
	// (indicates individual link) and Actor state sync is TRUE
	// and (3)
	//
	// 3) Rx pdu: Actor state LACP_Activity is TRUE
	// or both Actor Oper Port state LACP_Activity and PDU Partner
	// Partner state LACP_Activity is TRUE

	// (1)
	if ((LacpLacpPortInfoIsEqual(&lacpPduInfo.partner.info, &p.actorOper, LacpStateAggregationBit) &&
		LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateSyncBit)) ||
		//(2)
		(!LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateAggregationBit) &&
			LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateSyncBit))) &&
		// (3)
		(LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateActivityBit) ||
			(LacpStateIsSet(p.actorOper.state, LacpStateActivityBit) &&
				LacpStateIsSet(lacpPduInfo.partner.info.state, LacpStateActivityBit))) {

		LacpStateSet(&p.partnerOper.state, LacpStateSyncBit)
	} else {
		LacpStateClear(&p.partnerOper.state, LacpStateSyncBit)
	}

	// Optional to validate length of the following:
	// actor, partner, collector
}

// recordDefault: 802.1ax Section 6.4.9
//
// records the default parameter values for the
// partner carried in the partner admin parameters
// (Partner Admin Port Number, Partner Admin Port Priority,
//  Partner Admin System, Partner Admin System Priority,
// Partner Admin Key, and Partner Admin Port state) as the
// current Partner operational parameter values.  Sets Actor
// Oper Port state Default to TRUE and Partner Oper Port State
// Sync to TRUE
func (rxm *LacpRxMachine) recordDefault() {

	p := rxm.p

	LacpCopyLacpPortInfo(&p.partnerAdmin, &p.partnerOper)
	LacpStateSet(&p.actorOper.state, LacpStateDefaultedBit)
	LacpStateSet(&p.partnerOper.state, LacpStateSyncBit)
}

// updateNTT: 802.1ax Section 6.4.9
//
// Compare that the newly received PDU partner
// info agrees with the local port oper state.
// If it does not agree then set the NTT flag
// such that the Tx machine generates LACPDU
func (rxm *LacpRxMachine) updateNTT(lacpPduInfo *LacpPdu) bool {

	p := rxm.p

	const nttStateCompare uint8 = (LacpStateActivityBit | LacpStateTimeoutBit |
		LacpStateAggregationBit | LacpStateSyncBit)

	if !LacpLacpPortInfoIsEqual(&lacpPduInfo.partner.info, &p.partnerOper, nttStateCompare) {

		return true
	}
	return false
}

func (rxm *LacpRxMachine) recordVersionNumber(lacpPduInfo *LacpPdu) {

	p := rxm.p

	p.partnerVersion = lacpPduInfo.version
}

// currentWhileTimerValid checks the state against
// the Actor Port Oper State Timeout
func (rxm *LacpRxMachine) CurrentWhileTimerValid() (time.Duration, bool) {

	p := rxm.p
	if rxm.currentWhileTimerTimeout == LacpShortTimeoutTime &&
		!LacpStateIsSet(p.actorOper.state, LacpStateTimeoutBit) {
		rxm.LacpRxmLog("Current While Timer invalid adjusting to LONG TIMEOUT")
		return LacpLongTimeoutTime, false
	}
	if rxm.currentWhileTimerTimeout == LacpLongTimeoutTime &&
		LacpStateIsSet(p.actorOper.state, LacpStateTimeoutBit) {
		rxm.LacpRxmLog("Current While Timer invalid adjusting to SHORT TIMEOUT")
		return LacpShortTimeoutTime, false
	}
	return 0, true
}

func (rxm *LacpRxMachine) CheckPortMoved(partnerOper *LacpPortInfo, pktActor *LacpPortInfo) bool {
	return rxm.Machine.Curr.CurrentState() == LacpRxmStatePortDisabled &&
		partnerOper.port == pktActor.port &&
		partnerOper.system.systemId == pktActor.system.systemId &&
		partnerOper.system.systemPriority == pktActor.system.systemPriority
}
