// rxmachine
package lacp

import (
	"fmt"
	"utils/fsm"
)

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
)

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpRxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// machine specific events
	RxmEvents          chan fsm.Event
	RxmPktRxEvent      chan LacpPdu
	RxmKillSignalEvent chan bool
}

func (rxm *LacpRxMachine) PrevState() fsm.State { return rxm.PreviousState }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpRxMachine() *LacpRxMachine {
	rxm := &LacpRxMachine{
		PreviousState:      LacpRxmStateNone,
		RxmEvents:          make(chan fsm.Event),
		RxmPktRxEvent:      make(chan LacpPdu),
		RxmKillSignalEvent: make(chan bool)}

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

	return rxm.Machine
}

// LacpRxMachineInitialize function to be called after
// state transition to INITIALIZE
func (p *LaAggPort) LacpRxMachineInitialize(m fsm.Machine, data interface{}) fsm.State {
	fmt.Println("RXM: Initialize Enter")

	// detach from aggregator
	p.aggId = 0

	// make sure no timer is running
	p.WaitWhileTimerStop()

	// set the agg as being unselected
	p.aggSelected = LacpAggUnSelected

	// clear the Expired Bit
	LacpStateClear(p.actorOper.state, LacpStateExpiredBit)

	// set the port moved to false
	p.portMoved = false

	// next state
	return LacpRxmStateInitialize
}

// LacpRxMachineExpired function to be called after
// state transition to PORT_DISABLED
func (p *LaAggPort) LacpRxMachinePortDisabled(m fsm.Machine, data interface{}) fsm.State {
	fmt.Println("RXM: Port Disable Enter")

	// Clear the Sync state bit
	LacpStateClear(p.partnerOper.state, LacpStateSyncBit)

	return LacpRxmStatePortDisabled
}

// LacpRxMachineExpired function to be called after
// state transition to EXPIRED
func (p *LaAggPort) LacpRxMachineExpired(m fsm.Machine, data interface{}) fsm.State {
	fmt.Println("RXM: Expired Enter")

	// Sync set to FALSE
	LacpStateClear(p.partnerOper.state, LacpStateSyncBit)

	// Short timeout
	LacpStateSet(p.partnerOper.state, LacpStateTimeoutBit)

	// Set the Short timeout
	p.CurrentWhileTimerTimeoutSet(LacpShortTimeoutTime)

	// Start the Current While timer
	p.CurrentWhileTimerStart()

	// Actor Oper Port State Expired = TRUE
	LacpStateSet(p.actorOper.state, LacpStateExpiredBit)

	return LacpRxmStateExpired
}

// LacpRxMachineLacpDisabled function to be called after
// state transition to LACP_DISABLED
func (p *LaAggPort) LacpRxMachineLacpDisabled(m fsm.Machine, data interface{}) fsm.State {
	fmt.Println("RXM: Lacp Disabled Enter")

	// Unselect the aggregator
	p.aggSelected = LacpAggUnSelected

	// setup the default params
	p.recordDefault()

	// clear the Aggregation bit
	LacpStateClear(p.partnerOper.state, LacpStateAggregationBit)

	// clear the expired bit
	LacpStateClear(p.actorOper.state, LacpStateExpiredBit)

	return LacpRxmStateLacpDisabled
}

// LacpRxMachineDefaulted function to be called after
// state transition to DEFAULTED
func (p *LaAggPort) LacpRxMachineDefaulted(m fsm.Machine, data interface{}) fsm.State {
	fmt.Println("RXM: Defaulted Enter")

	//lacpPduInfo := data.(LacpPdu)

	// Updated the default selected state
	p.updateDefaultSelected()

	// Record the default partner info
	p.recordDefault()

	// Clear the expired bit on the actor oper state
	LacpStateClear(p.actorOper.state, LacpStateExpiredBit)

	return LacpRxmStateDefaulted
}

// LacpRxMachineCurrent function to be called after
// state transition to CURRENT
func (p *LaAggPort) LacpRxMachineCurrent(m fsm.Machine, data interface{}) fsm.State {
	fmt.Println("RXM: Current Enter")

	// Version 1, V2 will require a serialize/deserialize routine since TLV's are involved
	lacpPduInfo := data.(LacpPdu)

	// update selection logic
	p.updateSelected(&lacpPduInfo)

	// update the ntt
	p.updateNTT(&lacpPduInfo)

	// Version 2 or higher check
	if LacpActorSystemLacpVersion >= 0x2 {
		p.recordVersionNumber(&lacpPduInfo)
	}

	// record the current packet state
	p.recordPDU(&lacpPduInfo)

	// lets kick off the Current While Timer
	p.CurrentWhileTimerStart()

	// Actor_Oper_Port_Sate.Expired = FALSE
	LacpStateClear(p.actorOper.state, LacpStateExpiredBit)

	if p.nttFlag == true {
		// TODO update to send protocol right away
	}

	return LacpRxmStateCurrent
}

func (p *LaAggPort) LacpRxMachineFSMBuild() *LacpRxMachine {

	rules := fsm.Ruleset{}

	//BEGIN -> INIT
	rules.AddRule(LacpRxmStateNone, LacpRxmEventBegin, p.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventBegin, p.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateLacpDisabled, LacpRxmEventBegin, p.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateDefaulted, LacpRxmEventBegin, p.LacpRxMachineInitialize)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventBegin, p.LacpRxMachineInitialize)
	// INIT -> PORT_DISABLE
	rules.AddRule(LacpRxmStateInitialize, LacpRxmEventUnconditionalFallthrough, p.LacpRxMachinePortDisabled)
	// NOT PORT ENABLED  && NOT PORT MOVED
	// All states transition to this state
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventNotPortEnabledAndNotPortMoved, p.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateLacpDisabled, LacpRxmEventNotPortEnabledAndNotPortMoved, p.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateDefaulted, LacpRxmEventNotPortEnabledAndNotPortMoved, p.LacpRxMachinePortDisabled)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventNotPortEnabledAndNotPortMoved, p.LacpRxMachinePortDisabled)
	// PORT MOVED
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventPortMoved, p.LacpRxMachineInitialize)
	// PORT ENABLED && LACP ENABLED
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventPortEnabledAndLacpEnabled, p.LacpRxMachineExpired)
	// PORT ENABLED && LACP DISABLED
	rules.AddRule(LacpRxmStatePortDisabled, LacpRxmEventPortEnabledAndLacpDisabled, p.LacpRxMachineLacpDisabled)
	// CURRENT WHILE TIMER EXPIRED
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventCurrentWhileTimerExpired, p.LacpRxMachineDefaulted)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventCurrentWhileTimerExpired, p.LacpRxMachineExpired)
	// LACP ENABLED
	rules.AddRule(LacpRxmStateLacpDisabled, LacpRxmEventLacpEnabled, p.LacpRxMachinePortDisabled)
	// PKT RX
	rules.AddRule(LacpRxmStateExpired, LacpRxmEventLacpPktRx, p.LacpRxMachineCurrent)
	rules.AddRule(LacpRxmStateDefaulted, LacpRxmEventLacpPktRx, p.LacpRxMachineCurrent)
	rules.AddRule(LacpRxmStateCurrent, LacpRxmEventLacpPktRx, p.LacpRxMachineCurrent)

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	p.rxMachineFsm = NewLacpRxMachine()

	// Create a new FSM and apply the rules
	p.rxMachineFsm.Apply(&rules)

	return p.rxMachineFsm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpRxMachineMain() {

	// initialize the port
	p.begin = true

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.12 Receive Machine
	rxm := p.LacpRxMachineFSMBuild()

	// set the inital state
	rxm.Machine.Start(rxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpRxMachine) {
		select {
		case <-m.RxmKillSignalEvent:
			fmt.Println("RXM: Machine Killed")
			return

		case event := <-m.RxmEvents:
			m.Machine.ProcessEvent(event, nil)
			/* special case */
			if m.Machine.Curr.CurrentState() == LacpRxmStateInitialize {
				m.Machine.ProcessEvent(LacpRxmEventUnconditionalFallthrough, nil)
			}

		case pkt := <-rxm.RxmPktRxEvent:
			fmt.Println(pkt)
			// If you rx a packet must be in one
			// of 3 states
			// Expired/Defaulted/Current. each
			// state will transition to current
			// all other states should be ignored.

			m.Machine.ProcessEvent(LacpRxmEventLacpPktRx, pkt)

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
func (p *LaAggPort) recordPDU(lacpPduInfo *LacpPdu) {

	// Record Actor info from packet - store in parter operational
	// Port Number, Port Priority, System, System Priority
	// Key, state variables
	LacpCopyLacpPortInfo(&lacpPduInfo.actor.info, &p.partnerOper)

	// Set Actor Oper port state Defaulted to FALSE
	LacpStateClear(p.actorOper.state, LacpStateDefaultedBit)

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

		LacpStateSet(p.partnerOper.state, LacpStateSyncBit)
	} else {
		LacpStateClear(p.partnerOper.state, LacpStateSyncBit)
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
func (p *LaAggPort) recordDefault() {

	LacpCopyLacpPortInfo(&p.partnerAdmin, &p.partnerOper)
	LacpStateSet(p.actorOper.state, LacpStateDefaultedBit)
	LacpStateSet(p.partnerOper.state, LacpStateSyncBit)
}

// updateSelected:  802.1ax Section 6.4.9
// Sets the value of the Selected variable based on the following:
//
// Rx pdu: (Actor: Port, Priority, System, System Priority, Key
// and State Aggregation) vs (Partner Oper: Port, Priority,
// System, System Priority, Key, State Aggregation).  If values
// have changed then Selected is set to UNSELECTED, othewise
// SELECTED
func (p *LaAggPort) updateSelected(lacpPduInfo *LacpPdu) {

	if !LacpLacpPortInfoIsEqual(&lacpPduInfo.actor.info, &p.partnerOper, LacpStateAggregationBit) {

		p.aggSelected = LacpAggUnSelected
	}
}

// updateDefaultedSelected: 802.1ax Section 6.4.9
//
// Update the value of the Selected variable comparing
// the Partner admin info based with the partner
// operational info
// (port num, port priority, system, system priority,
//  key, stat.Aggregation)
func (p *LaAggPort) updateDefaultSelected() {

	if !LacpLacpPortInfoIsEqual(&p.partnerAdmin, &p.partnerOper, LacpStateAggregationBit) {

		p.aggSelected = LacpAggUnSelected
	}
}

// updateNTT: 802.1ax Section 6.4.9
//
// Compare that the newly received PDU partner
// info agrees with the local port oper state.
// If it does not agree then set the NTT flag
// such that the Tx machine generates LACPDU
func (p *LaAggPort) updateNTT(lacpPduInfo *LacpPdu) {

	const nttStateCompare uint8 = (LacpStateActivityBit | LacpStateTimeoutBit |
		LacpStateAggregationBit | LacpStateSyncBit)

	if !LacpLacpPortInfoIsEqual(&lacpPduInfo.partner.info, &p.partnerOper, nttStateCompare) {

		p.nttFlag = true
	}
}

func (p *LaAggPort) recordVersionNumber(lacpPduInfo *LacpPdu) {
	p.partnerVersion = lacpPduInfo.version
}
