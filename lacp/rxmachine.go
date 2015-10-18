// rxmachine
package lacp

import (
	"fmt"
	"utils/fsm"
)

// rxm states
const LacpRxmBegin = "LacpRxmBegin"
const LacpRxmInitialize = "LacpRxmInitialize"
const LacpRxmPortDisable = "LacpRxmPortDisable"
const LacpRxmExpired = "LacpRxmExpired"
const LacpRxmLacpDisabled = "LacpRxmLacpDisabled"
const LacpRxmDefaulted = "LacpRxmDefaulted"
const LacpRxmCurrent = "LacpRxmCurrent"

// rxm events
const LacpRxmPortMovedEvent = "LacpRxmPortMovedEvent"
const LacpRxmCurrentWhileTimerExpiredEvent = "LacpRxmCurrentWhileTimerExpiredEvent"
const LacpRxmLacpDisabledEvent = "LacpRxmLacpDisabledEvent"
const LacpRxmLacpEnabledEvent = "LacpRxmLacpEnabledEvent"
const LacpRxmLacpPortDisabledEvent = "LacpRxmLacpPortDisabledEvent"
const LacpRxmLacpPktRxEvent = "LacpRxmLacpPktRxEvent"

// Only valid pkt processing states
//const LacpRxmPktRxValidStates = [3]string{LacpRxmDefaulted, LacpRxmCurrent, LacpRxmExpired}

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpRxMachine struct {
	prevState fsm.State
	state     fsm.State

	machine *fsm.Machine

	// machine specific events
	rxmEvents          chan string
	rxmPktRxEvent      chan LacpPdu
	rxmKillSignalEvent chan bool
}

func (rxm *LacpRxMachine) PrevState() fsm.State    { return rxm.prevState }
func (rxm *LacpRxMachine) CurrentState() fsm.State { return rxm.state }
func (rxm *LacpRxMachine) SetState(s fsm.State)    { rxm.prevState = rxm.state; rxm.state = s }

// Transition will have the state machine try and transition to the correct state
// if the transition is not handled then the function will return false
// successful transtion will return true and set the current and previous state
func (rxm *LacpRxMachine) Transition(state fsm.State, data interface{}) bool {
	if err := rxm.machine.Transition(state, data); err == nil {
		rxm.SetState(state)
		fmt.Println("RXM: state transition - old state", rxm.PrevState(), "new state", rxm.CurrentState())
		return true
	}
	return false
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpRxMachine() *LacpRxMachine {
	rxm := &LacpRxMachine{
		state:              LacpRxmBegin,
		rxmEvents:          make(chan string),
		rxmPktRxEvent:      make(chan LacpPdu),
		rxmKillSignalEvent: make(chan bool)}

	return rxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine. While not
// required, it's something I like to have.
func (rxm *LacpRxMachine) Apply(r *fsm.Ruleset, c *fsm.CallbackSet) *fsm.Machine {
	if rxm.machine == nil {
		rxm.machine = &fsm.Machine{Subject: rxm}
	}

	rxm.machine.Rules = r
	rxm.machine.Callbacks = c
	return rxm.machine
}

// LacpRxMachineInitialize function to be called after
// state transition to INITIALIZE
func (p *LaAggPort) LacpRxMachineInitialize(data interface{}) error {
	fmt.Println("RXM: Initialize")

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
	return nil
}

// LacpRxMachineExpired function to be called after
// state transition to PORT_DISABLED
func (p *LaAggPort) LacpRxMachinePortDisabled(data interface{}) error {
	fmt.Println("RXM: Port Disable Enter")
	LacpStateClear(p.partnerOper.state, LacpStateSyncBit)
	return nil
}

// LacpRxMachineExpired function to be called after
// state transition to EXPIRED
func (p *LaAggPort) LacpRxMachineExpired(data interface{}) error {
	fmt.Println("RXM: Expired Enter")

	// Sync set to FALSE
	LacpStateClear(p.partnerOper.state, LacpStateSyncBit)
	// Short timeout
	LacpStateSet(p.partnerOper.state, LacpStateTimeoutBit)
	// Short timeout
	p.currentWhileTimerTimeout = LacpShortTimeoutTime
	p.CurrentWhileTimerStart()
	// Actor Oper Port State Expired = TRUE
	LacpStateSet(p.actorOper.state, LacpStateExpiredBit)

	return nil
}

// LacpRxMachineLacpDisabled function to be called after
// state transition to LACP_DISABLED
func (p *LaAggPort) LacpRxMachineLacpDisabled(data interface{}) error {
	fmt.Println("RXM: Lacp Disabled Enter")
	return nil
}

// LacpRxMachineDefaulted function to be called after
// state transition to DEFAULTED
func (p *LaAggPort) LacpRxMachineDefaulted(data interface{}) error {
	fmt.Println("RXM: Defaulted Enter")
	return nil
}

// LacpRxMachineCurrent function to be called after
// state transition to CURRENT
func (p *LaAggPort) LacpRxMachineCurrent(data interface{}) error {
	fmt.Println("RXM: Current Enter")
	return nil
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpRxMachineMain() {

	// initialize the port
	p.begin = true

	rules := fsm.Ruleset{}
	callbacks := fsm.CallbackSet{}

	//BEGIN -> INIT
	rules.AddTransition(fsm.T{LacpRxmBegin, LacpRxmInitialize})
	rules.AddTransition(fsm.T{LacpRxmInitialize, LacpRxmPortDisable})
	rules.AddTransition(fsm.T{LacpRxmExpired, LacpRxmPortDisable})
	rules.AddTransition(fsm.T{LacpRxmLacpDisabled, LacpRxmPortDisable})
	rules.AddTransition(fsm.T{LacpRxmDefaulted, LacpRxmPortDisable})
	rules.AddTransition(fsm.T{LacpRxmCurrent, LacpRxmPortDisable})
	rules.AddTransition(fsm.T{LacpRxmPortDisable, LacpRxmExpired})
	rules.AddTransition(fsm.T{LacpRxmPortDisable, LacpRxmLacpDisabled})
	rules.AddTransition(fsm.T{LacpRxmPortDisable, LacpRxmInitialize})
	rules.AddTransition(fsm.T{LacpRxmLacpDisabled, LacpRxmPortDisable})
	rules.AddTransition(fsm.T{LacpRxmExpired, LacpRxmDefaulted})
	rules.AddTransition(fsm.T{LacpRxmExpired, LacpRxmCurrent})
	rules.AddTransition(fsm.T{LacpRxmDefaulted, LacpRxmCurrent})
	rules.AddTransition(fsm.T{LacpRxmCurrent, LacpRxmCurrent})
	rules.AddTransition(fsm.T{LacpRxmCurrent, LacpRxmExpired})

	callbacks.AddCallback(LacpRxmInitialize, p.LacpRxMachineInitialize)
	callbacks.AddCallback(LacpRxmPortDisable, p.LacpRxMachinePortDisabled)
	callbacks.AddCallback(LacpRxmExpired, p.LacpRxMachineExpired)
	callbacks.AddCallback(LacpRxmLacpDisabled, p.LacpRxMachineLacpDisabled)
	callbacks.AddCallback(LacpRxmDefaulted, p.LacpRxMachineDefaulted)
	callbacks.AddCallback(LacpRxmCurrent, p.LacpRxMachineCurrent)

	// Instantiate a new LacpRxMachine
	p.rxMachineFsm = NewLacpRxMachine()
	rxm := p.rxMachineFsm

	// Create a new FSM and apply the rules and callbacks
	p.rxMachineFsm.Apply(&rules, &callbacks)

	// lets create the various channels for the various events to listen on
	go func() {
		select {
		case <-rxm.rxmKillSignalEvent:
			fmt.Println("RXM: Machine Killed")
			return

		case event := <-rxm.rxmEvents:
			switch event {
			case LacpRxmLacpPortDisabledEvent:
				if p.portMoved {
					fmt.Println("RXM: Error invalid event", event, "port moved must be false")
				} else {
					rxm.Transition(LacpRxmPortDisable, nil)
				}
			case LacpRxmPortMovedEvent:
				// event only valid in PORT_DISABLE state
				// all other transitions ignored
				rxm.Transition(LacpRxmInitialize, nil)
			case LacpRxmCurrentWhileTimerExpiredEvent:
				if rxm.Transition(LacpRxmExpired, nil) ||
					rxm.Transition(LacpRxmDefaulted, nil) {
					// do something if necessary
				}
			// assumption if recevieved this event then port is also enabled
			case LacpRxmLacpDisabledEvent:
				if !p.portEnabled {
					fmt.Println("RXM: Error invalid event", event, "port must be enabled")
				} else {
					rxm.Transition(LacpRxmPortDisable, nil)
					rxm.Transition(LacpRxmExpired, nil)
				}
			// assumption if received this event then port is also enabled
			case LacpRxmLacpEnabledEvent:
				if !p.portEnabled {
					fmt.Println("RXM: Error invalid event", event, "port must be enabled")
				} else {
					rxm.Transition(LacpRxmPortDisable, nil)
					rxm.Transition(LacpRxmExpired, nil)
				}
			case LacpRxmLacpPktRxEvent:
				// TODO include packet processing logic
			}
		case pkt := <-rxm.rxmPktRxEvent:
			fmt.Println(pkt)
		}
	}()

	for {
		select {}
	}
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
func recordPDU(port *LaAggPort, lacpPduInfo *LacpPdu) {

	// Record Actor info from packet - store in parter operational
	// Port Number, Port Priority, System, System Priority
	// Key, state variables
	LacpCopyLacpPortInfo(&lacpPduInfo.actor.info, &port.partnerOper)

	// Set Actor Oper port state Defaulted to FALSE
	LacpStateClear(port.actorOper.state, LacpStateDefaultedBit)

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
	if ((LacpLacpPortInfoIsEqual(&lacpPduInfo.partner.info, &port.actorOper, LacpStateAggregationBit) &&
		LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateSyncBit)) ||
		//(2)
		(!LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateAggregationBit) &&
			LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateSyncBit))) &&
		// (3)
		(LacpStateIsSet(lacpPduInfo.actor.info.state, LacpStateActivityBit) ||
			(LacpStateIsSet(port.actorOper.state, LacpStateActivityBit) &&
				LacpStateIsSet(lacpPduInfo.partner.info.state, LacpStateActivityBit))) {

		LacpStateSet(port.partnerOper.state, LacpStateSyncBit)
	} else {
		LacpStateClear(port.partnerOper.state, LacpStateSyncBit)
	}
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
func (p *LaAggPort) recordDefault(defaultInfo *LacpPortInfo) {

	LacpCopyLacpPortInfo(defaultInfo, &p.partnerAdmin)
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
func updateSelected(port *LaAggPort, lacpPduInfo *LacpPdu) {

	if !LacpLacpPortInfoIsEqual(&lacpPduInfo.actor.info, &port.partnerOper, LacpStateAggregationBit) {

		port.aggSelected = LacpAggUnSelected
	}
}

// updateDefaultedSelected: 802.1ax Section 6.4.9
//
// Update the value of the Selected variable comparing
// the Partner admin info based with the partner
// operational info
// (port num, port priority, system, system priority,
//  key, stat.Aggregation)
func updateDefaultSelected(port *LaAggPort) {

	if !LacpLacpPortInfoIsEqual(&port.partnerAdmin, &port.partnerOper, LacpStateAggregationBit) {

		port.aggSelected = LacpAggUnSelected
	}
}

// updateNTT: 802.1ax Section 6.4.9
//
// Compare that the newly received PDU partner
// info agrees with the local port oper state.
// If it does not agree then set the NTT flag
// such that the Tx machine generates LACPDU
func updateNTT(port *LaAggPort, lacpPduInfo *LacpPdu) {

	const nttStateCompare uint8 = (LacpStateActivityBit | LacpStateTimeoutBit |
		LacpStateAggregationBit | LacpStateSyncBit)

	if !LacpLacpPortInfoIsEqual(&lacpPduInfo.partner.info, &port.partnerOper, nttStateCompare) {

		port.nttFlag = true
	}
}
