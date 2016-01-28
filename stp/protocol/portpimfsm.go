// 802.1D-2004 17.27 Port Information State Machine
//The Port Information state machine shall implement the function specified by the state diagram in Figure 17-
//18, the definitions in 17.13, 17.16, 17.20, and 17.21, and the variable declarations in 17.17, 17.18, and 17.19.
//This state machine is responsible for updating and recording the source (infoIs, 17.19.10) of the Spanning
//Tree information (portPriority 17.19.21, portTimes 17.19.22) used to test the information conveyed
//(msgPriority, 17.19.14; msgTimes, 17.19.15) by received Configuration Messages. If new, superior,
//information arrives on the port, or the existing information is aged out, it sets the reselect variable to request
//the Port Role Selection state machine to update the spanning tree priority vectors held by the Bridge and the
//Bridge’s Port Roles.
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const PpimMachineModuleStr = "Port Information State Machine"

const (
	PpimStateNone = iota + 1
)

var PpimStateStrMap map[fsm.State]string

func PpimMachineStrStateMapInit() {
	PpimStateStrMap = make(map[fsm.State]string)

}

const (
	PpimEventBegin = iota + 1
)

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type PpimMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PpimEvents chan MachineEvent
	// stop go routine
	PpimKillSignalEvent chan bool
	// enable logging
	PpimLogEnableEvent chan bool
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPpimMachine(p *StpPort) *PpimMachine {
	ppim := &PpimMachine{
		p:                   p,
		PpimEvents:          make(chan MachineEvent, 10),
		PpimKillSignalEvent: make(chan bool),
		PpimLogEnableEvent:  make(chan bool)}

	p.PpimMachineFsm = ppim

	return ppim
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (ppim *PpimMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if ppim.Machine == nil {
		ppim.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	ppim.Machine.Rules = r
	ppim.Machine.Curr = &StpStateEvent{
		strStateMap: PpimStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PpimMachineModuleStr,
		ps:     PpimStateNone,
		s:      PpimStateNone,
	}

	return ppim.Machine
}

// Stop should clean up all resources
func (ppim *PpimMachine) Stop() {

	// stop the go routine
	ppim.PpimKillSignalEvent <- true

	close(ppim.PpimEvents)
	close(ppim.PpimLogEnableEvent)
	close(ppim.PpimKillSignalEvent)

}

/*
// PpmMachineCheckingRSTP
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
func PpimMachineFSMBuild(p *StpPort) *PpimMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	ppim := NewStpPpimMachine(p)

	// Create a new FSM and apply the rules
	ppim.Apply(&rules)

	return ppim
}

// PrxmMachineMain:
func (p *StpPort) PpimMachineMain() {

	// Build the State machine for STP Receive Machine according to
	// 802.1d Section 17.23
	ppim := PpimMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	ppim.Machine.Start(ppim.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PpimMachine) {
		StpLogger("INFO", "PPIM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PpimKillSignalEvent:
				StpLogger("INFO", "PPIM: Machine End")
				return

			case event := <-m.PpimEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(PpimMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PpimLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(ppim)
}
