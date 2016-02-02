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
	b *Bridge

	// machine specific events
	PrsEvents chan MachineEvent
	// stop go routine
	PrsKillSignalEvent chan bool
	// enable logging
	PrsLogEnableEvent chan bool
}

// NewStpPimMachine will create a new instance of the LacpRxMachine
func NewStpPrsMachine(b *Bridge) *PrsMachine {
	prsm := &PrsMachine{
		b:                  b,
		PrsEvents:          make(chan MachineEvent, 10),
		PrsKillSignalEvent: make(chan bool),
		PrsLogEnableEvent:  make(chan bool)}

	b.PrsMachineFsm = prsm

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

// PimMachineCheckingRSTP
func (prsm *PrsMachine) PrsMachineInitBridge(m fsm.Machine, data interface{}) fsm.State {
	b := prsm.b
	b.updtRoleDisabledTree()
	return PrsStateInitBridge
}

// PpmmMachineSelectingSTP
func (prsm *PrsMachine) PrsMachineRoleSelection(m fsm.Machine, data interface{}) fsm.State {
	b := prsm.b
	b.clearReselectTree()
	b.updtRolesTree()
	b.setSelectedTree()

	return PrsStateRoleSelection
}

func PrsMachineFSMBuild(b *Bridge) *PrsMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	prsm := NewStpPrsMachine(b)

	// BEGIN -> INIT_BRIDGE
	rules.AddRule(PrsStateNone, PrsEventBegin, prsm.PrsMachineInitBridge)

	// UNINTENTIONAL FALL THROUGH -> ROLE SELECTION
	rules.AddRule(PrsStateInitBridge, PrsEventUnconditionallFallThrough, prsm.PrsMachineRoleSelection)

	// RESLECT -> ROLE SELECTION
	rules.AddRule(PrsStateInitBridge, PrsEventReselect, prsm.PrsMachineRoleSelection)

	// Create a new FSM and apply the rules
	prsm.Apply(&rules)

	return prsm
}

// PrsMachineMain:
func (b *Bridge) PrsMachineMain() {

	// Build the State machine for STP Bridge Detection State Machine according to
	// 802.1d Section 17.25
	prsm := PrsMachineFSMBuild(b)
	b.wg.Add(1)

	// set the inital State
	prsm.Machine.Start(prsm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PrsMachine) {
		StpLogger("INFO", "PRSM: Machine Start")
		defer m.b.wg.Done()
		for {
			select {
			case <-m.PrsKillSignalEvent:
				StpLogger("INFO", "PRSM: Machine End")
				return

			case event := <-m.PrsEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("ERROR", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, event.e, PimStateStrMap[m.Machine.Curr.CurrentState()]))
				} else {
					if m.Machine.Curr.CurrentState() == PrsStateInitBridge {
						rv := m.Machine.ProcessEvent(PrsMachineModuleStr, PrsEventUnconditionallFallThrough, nil)
						if rv != nil {
							StpLogger("ERROR", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, event.e, PrsStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					}
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

// clearReselectTree: 17.21.2
func (b *Bridge) clearReselectTree() {
	var p *StpPort
	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			p.Reselect = false
		}
	}
}

func (b *Bridge) updtRoleDisabledTree() {
	var p *StpPort
	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			p.SelectedRole = PortRoleDisabledPort
		}
	}
}

// updtRolesTree: 17.21.25
func (b *Bridge) updtRolesTree() {

	// recorded from received message
	/* TOOD
	rootPathVector := PriorityVector{}
	bridgeRootPathVector := PriorityVector{}
	bridgeRootTimes := Times{}
	designatedPortVector := PriorityVector{}
	designatedTimes := Times{}
	*/
}

// setSelectedTree: 17.21.16
func (b *Bridge) setSelectedTree() {
	setAllSelectedTrue := true
	var p *StpPort
	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			if p.Reselect == true {
				setAllSelectedTrue = false
				break
			}
		}
	}
	if setAllSelectedTrue {
		for _, pId := range b.StpPorts {
			if StpFindPortById(pId, &p) {
				p.Selected = true
			}
		}
	}
}
