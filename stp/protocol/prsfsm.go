// 802.1D-2004 17.28 Port Role Selection State Machine
//The Port Role Selection state machine shall implement the function specified by the state diagram in Figure
//17-19, the definitions in 17.13, 17.16, 17.20, and 17.21, and the variable declarations in 17.17, 17.18, and
//17.19. It selects roles for all Bridge Ports.
//On initialization all Bridge Ports are assigned the Disabled Port Role. Whenever any Bridge Port’s reselect
//variable (17.19.34) is set by the Port Information state machine (17.27), spanning tree information including
//the designatedPriority (17.19.4) and designatedTimes (17.19.5) for each Port is recomputed and its Port
//Role (selectedRole, 17.19.37) updated by the updtRolesTree() procedure (17.21.25). The reselect variables
//are cleared before computation starts so that recomputation will take place if new information becomes
//available while the computation is in progress.
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const PrsMachineModuleStr = "Port Role Selection State Machine"

const (
	PrsStateNone = iota + 1
	PrsStateInitBridge
	PrsStateRoleSelection
)

var PrsStateStrMap map[fsm.State]string

func PrsMachineStrStateMapInit() {
	PrsStateStrMap = make(map[fsm.State]string)
	PrsStateStrMap[PrsStateNone] = "None"
	PrsStateStrMap[PrsStateInitBridge] = "Init Bridge"
	PrsStateStrMap[PrsStateRoleSelection] = "Role Selection"
}

const (
	PrsEventBegin = iota + 1
	PrsEventUnconditionallFallThrough
	PrsEventReselect
)

// PrsMachine holds FSM and current State
// and event channels for State transitions
type PrsMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PrsEvents chan MachineEvent
	// stop go routine
	PrsKillSignalEvent chan bool
	// enable logging
	PrsLogEnableEvent chan bool
}

// NewStpPimMachine will create a new instance of the LacpRxMachine
func NewStpPrsMachine(p *StpPort) *PrsMachine {
	prsm := &PrsMachine{
		p:                  p,
		PrsEvents:          make(chan MachineEvent, 10),
		PrsKillSignalEvent: make(chan bool),
		PrsLogEnableEvent:  make(chan bool)}

	p.PrsMachineFsm = prsm

	return prsm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (prsm *PrsMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if prsm.Machine == nil {
		prsm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	prsm.Machine.Rules = r
	prsm.Machine.Curr = &StpStateEvent{
		strStateMap: PrsStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PrsMachineModuleStr,
		ps:     PrsStateNone,
		s:      PrsStateNone,
	}

	return prsm.Machine
}

// Stop should clean up all resources
func (prsm *PrsMachine) Stop() {

	// stop the go routine
	prsm.PrsKillSignalEvent <- true

	close(prsm.PrsEvents)
	close(prsm.PrsLogEnableEvent)
	close(prsm.PrsKillSignalEvent)

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
func PrsMachineFSMBuild(p *StpPort) *PrsMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	prsm := NewStpPrsMachine(p)

	// Create a new FSM and apply the rules
	prsm.Apply(&rules)

	return prsm
}

// PimMachineMain:
func (p *StpPort) PrsMachineMain() {

	// Build the State machine for STP Bridge Detection State Machine according to
	// 802.1d Section 17.25
	prsm := PrsMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	prsm.Machine.Start(prsm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PrsMachine) {
		StpLogger("INFO", "PRSM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PrsKillSignalEvent:
				StpLogger("INFO", "PRSM: Machine End")
				return

			case event := <-m.PrsEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(PrsMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PrsLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(prsm)
}
