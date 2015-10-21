// TX MACHINE, this is not really a state machine but going to create a sort of
// state machine to processes events
// TX Machine is described in 802.1ax-2014 6.4.16
package lacp

import (
	"time"
	"utils/fsm"
)

const (
	LacpTxmStateNone = iota + 1
	LacpTxmStateOn
	LacpTxmStateOff
	LacpTxmStateDelayed
)

const (
	LacpTxmEventNtt = iota + 1
	LacpTxmEventGuardTimer
	LacpTxmEventDelayTx
	LacpTxmEventLacpDisabled
	LacpTxmEventLacpEnabled
)

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpTxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// Port this Machine is associated with
	p *LaAggPort

	// number of frames that should be transmitted
	// after restriction logic has cleared
	txPending int

	// number of frames transmitted within guard timer interval
	txPkts int

	// ntt, this may be set by external applications
	// the state machine will only clear
	ntt bool

	// timer needed for 802.1ax-20014 section 6.4.16
	txGuardTimer *time.Timer

	// machine specific events
	TxmEvents          chan fsm.Event
	TxmKillSignalEvent chan bool
}

// PrevState will get the previous state from the state transitions
func (txm *LacpTxMachine) PrevState() fsm.State { return txm.PreviousState }

// PrevStateSet will set the previous state
func (txm *LacpTxMachine) PrevStateSet(s fsm.State) { txm.PreviousState = s }

func (txm *LacpTxMachine) Stop() {
	txm.TxGuardTimerStop()

	txm.TxmKillSignalEvent <- true

	close(txm.TxmEvents)
	close(txm.TxmKillSignalEvent)
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpTxMachine(port *LaAggPort) *LacpTxMachine {
	txm := &LacpTxMachine{
		p:                  port,
		txPending:          0,
		txPkts:             0,
		ntt:                false,
		PreviousState:      LacpTxmStateNone,
		TxmEvents:          make(chan fsm.Event),
		TxmKillSignalEvent: make(chan bool)}

	port.txMachineFsm = txm

	return txm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine.
func (txm *LacpTxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if txm.Machine == nil {
		txm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	txm.Machine.Rules = r

	return txm.Machine
}

// LacpTxMachineOn will either send a packet out or delay transmission of a
// packet
func (txm *LacpTxMachine) LacpTxMachineOn(m fsm.Machine, data interface{}) fsm.State {

	var state fsm.State

	txm.PrevStateSet(txm.Machine.Curr.CurrentState())

	state = LacpTxmStateOn

	// NTT must be set to tx
	if txm.ntt == true {
		// if more than 3 packets are being transmitted within time interval
		// delay transmission
		if txm.txPkts < 3 {
			if txm.txPkts == 0 {
				txm.TxGuardTimerStart()
			}
			txm.txPkts++
			// TODO send packet to MUX
			// Version 2 consideration if enable_long_pdu_xmit and
			// LongLACPPDUTransmit are True:
			// LACPDU will be a Long LACPDU formatted by 802.1ax-2014 Section
			// 6.4.2 and including Port Conversation Mask TLV 6.4.2.4.3
			txm.ntt = false
		} else {
			txm.txPending++
			state = LacpTxmStateDelayed
		}
	}
	return state
}

// LacpTxMachineDelayed is a state in which a packet is forced to transmit
// regardless of the ntt state
func (txm *LacpTxMachine) LacpTxMachineDelayed(m fsm.Machine, data interface{}) fsm.State {

	var state fsm.State

	txm.PrevStateSet(txm.Machine.Curr.CurrentState())

	state = LacpTxmStateOn

	// if more than 3 packets are being transmitted within time interval
	// delay transmission
	txm.txPending--
	// TODO send packet to MUX
	// Version 2 consideration if enable_long_pdu_xmit and
	// LongLACPPDUTransmit are True:
	// LACPDU will be a Long LACPDU formatted by 802.1ax-2014 Section
	// 6.4.2 and including Port Conversation Mask TLV 6.4.2.4.3
	txm.ntt = false
	if txm.txPending > 0 {
		state = LacpTxmStateDelayed
		txm.TxmEvents <- LacpTxmEventDelayTx
	}

	return state
}

// LacpTxMachineOff will ensure that no packets are transmitted, typically means that
// lacp has been disabled
func (txm *LacpTxMachine) LacpTxMachineOff(m fsm.Machine, data interface{}) fsm.State {
	txm.txPending = 0
	txm.txPkts = 0
	txm.ntt = false
	return LacpTxmStateOff
}

// LacpTxMachineGuard will clear the current transmited packet count and
// generate a new event to tx a new packet
func (txm *LacpTxMachine) LacpTxMachineGuard(m fsm.Machine, data interface{}) fsm.State {
	txm.txPkts = 0

	if txm.txPending > 0 {
		txm.TxmEvents <- LacpTxmEventDelayTx
	}
	// no state transition just need to clear the txPkts
	return txm.Machine.Curr.CurrentState()
}

// LacpTxMachineFSMBuild will build the state machine with callbacks
func (p *LaAggPort) LacpTxMachineFSMBuild() *LacpTxMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	txm := NewLacpTxMachine(p)

	//NONE -> TX ON
	rules.AddRule(LacpTxmStateNone, LacpTxmEventNtt, txm.LacpTxMachineOn)
	// NTT -> TX ON
	rules.AddRule(LacpTxmStateOn, LacpTxmEventNtt, txm.LacpTxMachineOn)
	// DELAY -> TX DELAY
	rules.AddRule(LacpTxmStateOn, LacpTxmEventNtt, txm.LacpTxMachineDelayed)
	rules.AddRule(LacpTxmStateDelayed, LacpTxmEventDelayTx, txm.LacpTxMachineDelayed)
	// LACP ON -> TX ON
	rules.AddRule(LacpTxmStateOff, LacpTxmEventLacpEnabled, txm.LacpTxMachineOn)
	// LACP DISABLED -> TX OFF
	rules.AddRule(LacpTxmStateNone, LacpTxmEventLacpDisabled, txm.LacpTxMachineOff)
	rules.AddRule(LacpTxmStateOn, LacpTxmEventLacpDisabled, txm.LacpTxMachineOff)
	rules.AddRule(LacpTxmStateDelayed, LacpTxmEventLacpDisabled, txm.LacpTxMachineOff)

	// Create a new FSM and apply the rules
	txm.Apply(&rules)

	return txm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpTxMachineMain() {

	// initialize the port
	p.begin = true

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	txm := p.LacpTxMachineFSMBuild()

	// set the inital state
	txm.Machine.Start(txm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpTxMachine) {
		m.p.LacpDebug.LacpLogStateTransitionChan <- "PTXM: Machine Start"
		select {
		case <-m.TxmKillSignalEvent:
			m.p.LacpDebug.LacpLogStateTransitionChan <- "TXM: Machine End"
			return

		case event := <-m.TxmEvents:

			// special case, another machine has a need to
			// transmit a packet
			if event == LacpTxmEventNtt {
				m.ntt = true
			}

			m.Machine.ProcessEvent(event, nil)
		}
	}(txm)
}

// LacpTxGuardGeneration will generate an event to the Tx Machine
// in order to clear the txPkts count
func (txm *LacpTxMachine) LacpTxGuardGeneration() {
	txm.TxmEvents <- LacpTxmEventGuardTimer
}
