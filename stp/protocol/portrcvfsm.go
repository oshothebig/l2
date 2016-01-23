// 17.22 Port Timers state machine
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const PrxmMachineModuleStr = "Port Receive State Machine"

const (
	PrxmStateNone = iota + 1
	PrxmStateDiscard
	PrxmStateReceive
)

var PrxmStateStrMap map[fsm.State]string

func PrxmMachineStrStateMapInit() {
	PrxmStateStrMap = make(map[fsm.State]string)
	PrxmStateStrMap[PrxmStateNone] = "None"
	PrxmStateStrMap[PrxmStateDiscard] = "Discard"
	PrxmStateStrMap[PrxmStateReceive] = "Receive"
}

const (
	PrxmEventBegin = iota + 1
	PrxmEventRcvdBpduAndNotPortEnabled
	PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled
	PrxmEventRcvdBpduAndPortEnabled
	PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg
)

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type PrxmMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PrxmEvents chan MachineEvent
	// stop go routine
	PrxmKillSignalEvent chan bool
	// enable logging
	PrxmLogEnableEvent chan bool
}

func (prxm *PrxmMachine) PrevState() fsm.State { return prxm.PreviousState }

// PrevStateSet will set the previous State
func (prxm *PrxmMachine) PrevStateSet(s fsm.State) { prxm.PreviousState = s }

func (prxm *PrxmMachine) Stop() {

	prxm.PrxmKillSignalEvent <- true

	close(prxm.PrxmEvents)
	close(prxm.PrxmKillSignalEvent)
	close(prxm.PrxmLogEnableEvent)
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPrxmMachine(p *StpPort) *PrxmMachine {
	prxm := &PrxmMachine{
		p:                   p,
		PreviousState:       PrxmStateNone,
		PrxmEvents:          make(chan MachineEvent, 10),
		PrxmKillSignalEvent: make(chan bool),
		PrxmLogEnableEvent:  make(chan bool)}

	return prxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (prxm *PrxmMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if prxm.Machine == nil {
		prxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	prxm.Machine.Rules = r
	prxm.Machine.Curr = &StpStateEvent{
		strStateMap: PrxmStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		//logger: fmt.Println,
		owner: PrxmMachineModuleStr,
	}

	return prxm.Machine
}

// PrmMachineDiscard
func (prxm *PrxmMachine) PrxmMachineDiscard(m fsm.Machine, data interface{}) fsm.State {
	p := prxm.p
	p.RcvdBPDU = false
	p.RcvdRSTP = false
	p.RcvdSTP = false
	p.RcvdMsg = false
	// set to RSTP performance paramters Migrate Time
	//p.EdgeDelayWhileTimer.count =
	return PrxmStateDiscard
}

// LacpPtxMachineFastPeriodic sets the periodic transmission time to fast
// and starts the timer
func (prxm *PrxmMachine) PrxmMachineReceive(m fsm.Machine, data interface{}) fsm.State {
	return PrxmStateReceive
}

func PrxmMachineFSMBuild(p *StpPort) *PrxmMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	prxm := NewStpPrxmMachine(p)

	//BEGIN -> DISCARD
	rules.AddRule(PrxmStateNone, PrxmEventBegin, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateDiscard, PrxmEventBegin, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateReceive, PrxmEventBegin, prxm.PrxmMachineDiscard)

	// RX BPDU && PORT NOT ENABLED	 -> DISCARD
	rules.AddRule(PrxmStateDiscard, PrxmEventRcvdBpduAndNotPortEnabled, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateReceive, PrxmEventRcvdBpduAndNotPortEnabled, prxm.PrxmMachineDiscard)

	// EDGEDELAYWHILE != MIGRATETIME && PORT NOT ENABLED -> DISCARD
	rules.AddRule(PrxmStateDiscard, PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateReceive, PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled, prxm.PrxmMachineDiscard)

	// RX BPDU && PORT ENABLED -> RECEIVE
	rules.AddRule(PrxmStateDiscard, PrxmEventRcvdBpduAndPortEnabled, prxm.PrxmMachineReceive)

	// RX BPDU && PORT ENABLED && NOT RCVDMSG
	rules.AddRule(PrxmStateDiscard, PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg, prxm.PrxmMachineReceive)

	// Create a new FSM and apply the rules
	prxm.Apply(&rules)

	return prxm
}

// PrxmMachineMain:
func (p *StpPort) PrxmMachineMain() {

	// Build the State machine for STP Receive Machine according to
	// 802.1d Section 17.23
	prxm := PrxmMachineFSMBuild(p)

	// set the inital State
	prxm.Machine.Start(prxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PrxmMachine) {
		fmt.Println("PRXM: Machine Start")
		//defer m.p.wg.Done()
		for {
			select {
			case <-m.PrxmKillSignalEvent:
				fmt.Println("PRXM: Machine End")
				return
				/*
					case <-m.periodicTxTimer.C:

						//m.LacpPtxmLog("Timer expired current State")
						//m.LacpPtxmLog(PtxmStateStrMap[m.Machine.Curr.CurrentState()])
						m.Machine.ProcessEvent(PtxMachineModuleStr, PtmEventTickEqualsTrue, nil)
				*/

			case event := <-m.PrxmEvents:
				m.Machine.ProcessEvent(event.src, event.e, nil)

				if event.responseChan != nil {
					SendResponse(PrxmMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PrxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(prxm)
}
