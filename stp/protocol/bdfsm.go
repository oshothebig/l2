// 802.1D-2004 17.25 Bridge Detection State Machine
//The Bridge Detection state machine shall implement the function specified by the state diagram in Figure
//17-16, the definitions in 17.16, 17.13, 17.20, and 17.21, and the variable declarations in 17.17, 17.18, and
//17.19.
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const BdmMachineModuleStr = "Bridge Detection State Machine"

const (
	BdmStateNone = iota + 1
	BdmStateEdge
	BdmStateNotEdge
)

var BdmStateStrMap map[fsm.State]string

func BdmMachineStrStateMapInit() {
	BdmStateStrMap = make(map[fsm.State]string)
	BdmStateStrMap[BdmStateNone] = "None"
	BdmStateStrMap[BdmStateEdge] = "Edge"
	BdmStateStrMap[BdmStateNotEdge] = "NotEdge"
}

const (
	BdmEventBeginAdminEdge = iota + 1
	BdmEventBeginNotAdminEdge
	BdmEventNotPortEnabledAndAdminEdge
	BdmEventEdgeDelayWhileEqualZeroAndAutoEdgeAndSendRSTPAndProposing
	BdEventNotPortEnabledAndNotAdminEdge
	BdEventNotOperEdge
)

// BdmMachine holds FSM and current State
// and event channels for State transitions
type BdmMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	BdmEvents chan MachineEvent
	// stop go routine
	BdmKillSignalEvent chan bool
	// enable logging
	BdmLogEnableEvent chan bool
}

// NewStpPimMachine will create a new instance of the LacpRxMachine
func NewStpBdmMachine(p *StpPort) *BdmMachine {
	bdm := &BdmMachine{
		p:                  p,
		BdmEvents:          make(chan MachineEvent, 10),
		BdmKillSignalEvent: make(chan bool),
		BdmLogEnableEvent:  make(chan bool)}

	p.BdmMachineFsm = bdm

	return bdm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (bdm *BdmMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if bdm.Machine == nil {
		bdm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	bdm.Machine.Rules = r
	bdm.Machine.Curr = &StpStateEvent{
		strStateMap: BdmStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  BdmMachineModuleStr,
		ps:     BdmStateNone,
		s:      BdmStateNone,
	}

	return bdm.Machine
}

// Stop should clean up all resources
func (bdm *BdmMachine) Stop() {

	// stop the go routine
	bdm.BdmKillSignalEvent <- true

	close(bdm.BdmEvents)
	close(bdm.BdmLogEnableEvent)
	close(bdm.BdmKillSignalEvent)

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
func BdmMachineFSMBuild(p *StpPort) *BdmMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	bdm := NewStpBdmMachine(p)

	// Create a new FSM and apply the rules
	bdm.Apply(&rules)

	return bdm
}

// PimMachineMain:
func (p *StpPort) BdmMachineMain() {

	// Build the State machine for STP Bridge Detection State Machine according to
	// 802.1d Section 17.25
	bdm := BdmMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	bdm.Machine.Start(bdm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *BdmMachine) {
		StpLogger("INFO", "PIM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.BdmKillSignalEvent:
				StpLogger("INFO", "BDM: Machine End")
				return

			case event := <-m.BdmEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(BdmMachineModuleStr, event.responseChan)
				}

			case ena := <-m.BdmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(bdm)
}
