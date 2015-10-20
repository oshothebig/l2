// TX MACHINE, this is not really a state machine but going to create a sort of
// state machine to processes events
// TX Machine is described in 802.1ax-2014 6.4.16
package lacp

import (
	"fmt"
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

	// machine specific events
	TxmEvents          chan fsm.Event
	TxmKillSignalEvent chan bool
}

func (txm *LacpTxMachine) PrevState() fsm.State { return txm.PreviousState }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpTxMachine() *LacpTxMachine {
	ptxm := &LacpTxMachine{
		PreviousState:      LacpTxmStateNone,
		TxmEvents:          make(chan fsm.Event),
		TxmKillSignalEvent: make(chan bool)}

	return ptxm
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
func (p *LaAggPort) LacpTxMachineOn(m fsm.Machine, data interface{}) fsm.State {

	var state fsm.State

	state = LacpTxmStateOn

	// NTT must be set to tx
	if p.nttFlag == true {
		// Lacp must be enabled
		if LacpModeGet(p.actorOper.state, p.lacpEnabled) != LacpModeOn {
			// if more than 3 packets are being transmitted within time interval
			// delay transmission
			if p.peridicTxCnt < 3 {
				p.peridicTxCnt++
				// TODO send packet to MUX
				// Version 2 consideration if enable_long_pdu_xmit and
				// LongLACPPDUTransmit are True:
				// LACPDU will be a Long LACPDU formatted by 802.1ax-2014 Section
				// 6.4.2 and including Port Conversation Mask TLV 6.4.2.4.3
				p.nttFlag = false
			} else {
				// Delay transmission of the next pdu
				p.TxDelayTimerStart()
				state = LacpTxmStateDelayed
			}
		} else {
			state = LacpTxmStateOff
		}
	}
	return state
}

// LacpTxMachineDelayed is a state in which a packet is forced to transmit
// regardless of the ntt state
func (p *LaAggPort) LacpTxMachineDelayed(m fsm.Machine, data interface{}) fsm.State {

	var state fsm.State

	state = LacpTxmStateOn

	// reset the count
	p.peridicTxCnt = 0

	// Lacp must be enabled
	if LacpModeGet(p.actorOper.state, p.lacpEnabled) != LacpModeOn {
		// if more than 3 packets are being transmitted within time interval
		// delay transmission
		p.peridicTxCnt++
		// TODO send packet to MUX
		// Version 2 consideration if enable_long_pdu_xmit and
		// LongLACPPDUTransmit are True:
		// LACPDU will be a Long LACPDU formatted by 802.1ax-2014 Section
		// 6.4.2 and including Port Conversation Mask TLV 6.4.2.4.3
		p.nttFlag = false
	} else {
		state = LacpTxmStateOff
	}
	return state
}

func (p *LaAggPort) LacpTxMachineOff(m fsm.Machine, data interface{}) fsm.State {
	p.nttFlag = false
	return LacpTxmStateOff
}

func (p *LaAggPort) LacpTxMachineEnable(m fsm.Machine, data interface{}) fsm.State {

	return LacpTxmStateOn
}

func (p *LaAggPort) LacpTxMachineFSMBuild() *LacpTxMachine {

	rules := fsm.Ruleset{}

	//NONE -> TX ON
	rules.AddRule(LacpTxmStateNone, LacpTxmEventNtt, p.LacpTxMachineOn)
	// NTT -> TX ON
	rules.AddRule(LacpTxmStateOn, LacpTxmEventNtt, p.LacpTxMachineOn)
	// DELAY -> TX DELAY
	rules.AddRule(LacpTxmStateDelayed, LacpTxmEventDelayTx, p.LacpTxMachineDelayed)

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	p.txMachineFsm = NewLacpTxMachine()

	// Create a new FSM and apply the rules
	p.txMachineFsm.Apply(&rules)

	return p.txMachineFsm
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
		select {
		case <-m.TxmKillSignalEvent:
			fmt.Println("PTXM: Machine Killed")
			return

		case event := <-m.TxmEvents:

			if m.Machine.Curr.CurrentState() != LacpTxmStateOff {
				// special case, lets force the state to be delayed
				// so that we get different tx behavior
				if event == LacpTxmEventDelayTx {
					m.Machine.Curr.SetState(LacpTxmStateDelayed)
				}

				m.Machine.ProcessEvent(event, nil)
			} else if event == LacpTxmEventLacpEnabled {
				m.Machine.ProcessEvent(event, nil)
			}
		}
	}(txm)
}

func (p *LaAggPort) LacpTxDeferTxEventGeneration() {
	p.txMachineFsm.TxmEvents <- LacpTxmEventDelayTx
}
