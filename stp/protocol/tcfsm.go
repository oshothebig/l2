// 802.1D-2004 17.30 Topology Change State Machine
//The Topology Change state machine shall implement the function specified by the state diagram in Figure
//17-25, the definitions in 17.13, 17.16, 17.20, and 17.21, and variable declarations in 17.17, 17.18, and 17.19
//This state machine is responsible for topology change detection, notification, and propagation, and for
//instructing the Filtering Database to remove Dynamic Filtering Entries for certain ports (17.11).
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const TcMachineModuleStr = "Topology Change State Machine"

const (
	TcStateNone = iota + 1
	TcStateInactive
	TcStateLearning
	TcStateDetected
	TcStateActive
	TcStateNotifiedTcn
	TcStateNotifiedTc
	TcStatePropagating
	TcStateAcknowledged
)

var TcStateStrMap map[fsm.State]string

func TcMachineStrStateMapInit() {
	TcStateStrMap = make(map[fsm.State]string)
	TcStateStrMap[PrsStateNone] = "None"
	TcStateStrMap[TcStateInactive] = "Inactive"
	TcStateStrMap[TcStateLearning] = "Learning"
	TcStateStrMap[TcStateDetected] = "Detected"
	TcStateStrMap[TcStateActive] = "Active"
	TcStateStrMap[TcStateNotifiedTcn] = "NotifiedTcn"
	TcStateStrMap[TcStateNotifiedTc] = "NotifiedTc"
	TcStateStrMap[TcStatePropagating] = "Propagating"
	TcStateStrMap[TcStateAcknowledged] = "Acknowledged"
}

const (
	TcEventBegin = iota + 1
	TcEventUnconditionalFallThrough
	TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPortAndNotLearnAndNotLearningAndNotRcvdTcAndNotRcvdTcnAndNotRcvdTcAckAndNotTcProp
	TcEventLearnAndNotFdbFlush
	TcEventRcvdTc
	TcEventRcvdTcn
	TcEventRcvdTcAck
	TcEventTcProp
	TcEventRoleEqualRootPortAndForwardAndNotOperEdge
	TcEventRoleEqualDesignatedPortAndForwardAndNotOperEdge
	TcEventTcPropAndNotOperEdge
	TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPort
	TcEventOperEdge
)

// TcMachine holds FSM and current State
// and event channels for State transitions
type TcMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	TcEvents chan MachineEvent
	// stop go routine
	TcKillSignalEvent chan bool
	// enable logging
	TcLogEnableEvent chan bool
}

// NewStpTcMachine will create a new instance of the LacpRxMachine
func NewStpTcMachine(p *StpPort) *TcMachine {
	tcm := &TcMachine{
		p:                 p,
		TcEvents:          make(chan MachineEvent, 10),
		TcKillSignalEvent: make(chan bool),
		TcLogEnableEvent:  make(chan bool)}

	p.TcMachineFsm = tcm

	return tcm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (tcm *TcMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if tcm.Machine == nil {
		tcm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	tcm.Machine.Rules = r
	tcm.Machine.Curr = &StpStateEvent{
		strStateMap: TcStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  TcMachineModuleStr,
		ps:     TcStateNone,
		s:      TcStateNone,
	}

	return tcm.Machine
}

// Stop should clean up all resources
func (tcm *TcMachine) Stop() {

	// stop the go routine
	tcm.TcKillSignalEvent <- true

	close(tcm.TcEvents)
	close(tcm.TcLogEnableEvent)
	close(tcm.TcKillSignalEvent)

}

/*
// TcMachineCheckingRSTP
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
func TcMachineFSMBuild(p *StpPort) *TcMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	tcm := NewStpTcMachine(p)

	// Create a new FSM and apply the rules
	tcm.Apply(&rules)

	return tcm
}

// PimMachineMain:
func (p *StpPort) TcMachineMain() {

	// Build the State machine for STP Bridge Detection State Machine according to
	// 802.1d Section 17.25
	tcm := TcMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	tcm.Machine.Start(tcm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *TcMachine) {
		StpLogger("INFO", "TCM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.TcKillSignalEvent:
				StpLogger("INFO", "TCM: Machine End")
				return

			case event := <-m.TcEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(TcMachineModuleStr, event.responseChan)
				}

			case ena := <-m.TcLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(tcm)
}
