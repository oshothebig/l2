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

const PimMachineModuleStr = "Port Information State Machine"

const (
	PimStateNone = iota + 1
	PimStateDisabled
	PimStateAged
	PimStateUpdated
	PimStateSuperiorDesignated
	PimStateRepeatedDesignated
	PimStateInferiorDesignated
	PimStateNotDesignated
	PimStateOther
	PimStateCurrent
	PimStateReceive
)

var PimStateStrMap map[fsm.State]string

func PimMachineStrStateMapInit() {
	PimStateStrMap = make(map[fsm.State]string)
	PimStateStrMap[PimStateNone] = "None"
	PimStateStrMap[PimStateDisabled] = "Disabled"
	PimStateStrMap[PimStateAged] = "Aged"
	PimStateStrMap[PimStateUpdated] = "Updated"
	PimStateStrMap[PimStateSuperiorDesignated] = "Superior Designated"
	PimStateStrMap[PimStateRepeatedDesignated] = "Repeated Designated"
	PimStateStrMap[PimStateInferiorDesignated] = "Inferior Designated"
	PimStateStrMap[PimStateNotDesignated] = "Not Designated"
	PimStateStrMap[PimStateOther] = "Other"
	PimStateStrMap[PimStateCurrent] = "Current"
	PimStateStrMap[PimStateReceive] = "Receive"

}

const (
	PimEventBegin = iota + 1
	PimEventNotPortEnabledInfolsNotEqualDisabled
	PimEventRcvdMsg
	PimEventRcvdMsgAndNotUpdtInfo
	PimEventPortEnabled
	PimEventSelectedAndUpdtInfo
	PimEventUnconditionalFallThrough
	PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg
	PimEventRcvdInfoEqualSuperiorDesignatedInfo
	PimEventRcvdInfoEqualRepeatedDesignatedInfo
	PimEventRcvdInfoEqualInferiorDesignatedInfo
	PimEventRcvdInfoEqualRootAlternateInfo
	PimEventRcvdInfoEqualOtherInfo
)

// PimMachine holds FSM and current State
// and event channels for State transitions
type PimMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PimEvents chan MachineEvent
	// stop go routine
	PimKillSignalEvent chan bool
	// enable logging
	PimLogEnableEvent chan bool
}

// NewStpPimMachine will create a new instance of the LacpRxMachine
func NewStpPimMachine(p *StpPort) *PimMachine {
	pim := &PimMachine{
		p:                  p,
		PimEvents:          make(chan MachineEvent, 10),
		PimKillSignalEvent: make(chan bool),
		PimLogEnableEvent:  make(chan bool)}

	p.PimMachineFsm = pim

	return pim
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (pim *PimMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if pim.Machine == nil {
		pim.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	pim.Machine.Rules = r
	pim.Machine.Curr = &StpStateEvent{
		strStateMap: PimStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PimMachineModuleStr,
		ps:     PimStateNone,
		s:      PimStateNone,
	}

	return pim.Machine
}

// Stop should clean up all resources
func (pim *PimMachine) Stop() {

	// stop the go routine
	pim.PimKillSignalEvent <- true

	close(pim.PimEvents)
	close(pim.PimLogEnableEvent)
	close(pim.PimKillSignalEvent)

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
func PimMachineFSMBuild(p *StpPort) *PimMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	pim := NewStpPimMachine(p)

	// Create a new FSM and apply the rules
	pim.Apply(&rules)

	return pim
}

// PimMachineMain:
func (p *StpPort) PimMachineMain() {

	// Build the State machine for STP Receive Machine according to
	// 802.1d Section 17.27
	pim := PimMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	pim.Machine.Start(pim.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PimMachine) {
		StpLogger("INFO", "PIM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PimKillSignalEvent:
				StpLogger("INFO", "PIM: Machine End")
				return

			case event := <-m.PimEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(PimMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PimLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(pim)
}
