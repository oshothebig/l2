// 802.1D-2004 17.29 Port State Transition State Machine
//The Port State Transition state machine shall implement the function specified by the state diagram in Figure
//17-24, the definitions in 17.13, 17.16, 17.20, and 17.21, and variable declarations in 17.17, 17.18, and 17.19.
//This state machine models changes in the Port State (7.4). The Port Role Transitions state machine requests
//changes by setting the learn and forward variables; the Port State Transitions machine updates the learning
//and forwarding variables as the actual Port State changes. The disableLearning(), disableForwarding(),
//enableLearning(), and enableForwarding() procedures model the system-dependent actions and delays that
//take place; these procedures do not complete until the desired behavior has been achieved.
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const PstMachineModuleStr = "Port Role Transitions State Machine"

const (
	PstStateNone = iota + 1
	PstStateDiscarding
	PstStateLearning
	PstStateForwarding
)

var PstStateStrMap map[fsm.State]string

func PstMachineStrStateMapInit() {
	PstStateStrMap = make(map[fsm.State]string)
	PstStateStrMap[PstStateNone] = "None"
	PstStateStrMap[PstStateDiscarding] = "Discarding"
	PstStateStrMap[PstStateLearning] = "Learning"
	PstStateStrMap[PstStateForwarding] = "Forwarding"
}

const (
	PstEventBegin = iota + 1
	PstEventLearn
	PstEventNotLearn
	PstEventForward
	PstEventNotForward
)

// PstMachine holds FSM and current State
// and event channels for State transitions
type PstMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PstEvents chan MachineEvent
	// stop go routine
	PstKillSignalEvent chan bool
	// enable logging
	PstLogEnableEvent chan bool
}

// NewStpPrtMachine will create a new instance of the LacpRxMachine
func NewStpPstMachine(p *StpPort) *PstMachine {
	pstm := &PstMachine{
		p:                  p,
		PstEvents:          make(chan MachineEvent, 10),
		PstKillSignalEvent: make(chan bool),
		PstLogEnableEvent:  make(chan bool)}

	p.PstMachineFsm = pstm

	return pstm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (pstm *PstMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if pstm.Machine == nil {
		pstm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	pstm.Machine.Rules = r
	pstm.Machine.Curr = &StpStateEvent{
		strStateMap: PstStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PstMachineModuleStr,
		ps:     PstStateNone,
		s:      PstStateNone,
	}

	return pstm.Machine
}

// Stop should clean up all resources
func (pstm *PstMachine) Stop() {

	// stop the go routine
	pstm.PstKillSignalEvent <- true

	close(pstm.PstEvents)
	close(pstm.PstLogEnableEvent)
	close(pstm.PstKillSignalEvent)

}

/*
// PimMachineCheckingRSTP
func (ppmm *PpmmMachine) PpmmMachineCheckingRSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p
	p.Mcheck = false
	p.SendRSTP = p.BridgeProtocolVersionGet() == layers.RSTPProtocolVersion
	// 17.24
	// TODO inform Port Transmit State Machine what STP version to send and which BPDU types
	// to support interoperability
	p.MdelayWhiletimer.count = MigrateTimeDefault
	return PpmmStateCheckingRSTP
}

// PpmmMachineSelectingSTP
func (ppmm *PpmmMachine) PpmmMachineSelectingSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p

	p.SendRSTP = false
	// 17.24
	// TODO inform Port Transmit State Machine what STP version to send and which BPDU types
	// to support interoperability
	p.MdelayWhiletimer.count = MigrateTimeDefault

	return PpmmStateSelectingSTP
}

// PpmmMachineSelectingSTP
func (ppmm *PpmmMachine) PpmmMachineSensing(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p

	p.RcvdRSTP = false
	p.RcvdSTP = false

	return PpmmStateSensing
}
*/
func PstMachineFSMBuild(p *StpPort) *PstMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrtMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	pstm := NewStpPstMachine(p)

	// Create a new FSM and apply the rules
	pstm.Apply(&rules)

	return pstm
}

// PstmMachineMain:
func (p *StpPort) PstMachineMain() {

	// Build the State machine for STP Port Role Transitions State Machine according to
	// 802.1d Section 17.29
	pstm := PstMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	pstm.Machine.Start(pstm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	go func(m *PstMachine) {
		StpLogger("INFO", "PRTM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PstKillSignalEvent:
				StpLogger("INFO", "PSTM: Machine End")
				return

			case event := <-m.PstEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(PstMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PstLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(pstm)
}
