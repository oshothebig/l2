// 802.1D-2004 17.24 Port Protocol Migration state machine
// The Port Protocol Migration state machine shall implement the function specified by the state diagram in
// Figure 17-15, the definitions in 17.16, 17.20, and 17.21, and the variable declarations in 17.17, 17.18, and
// 17.19. It updates sendRSTP (17.19.38) to tell the Port Transmit state machine (17.26) which BPDU types
// (9.3) to transmit, to support interoperability (17.4) with the Spanning Tree Algorithm and Protocol specified
// in previous revisions of this standard.
package stp

import (
	"fmt"
	//"time"
	"github.com/google/gopacket/layers"
	"utils/fsm"
)

const PpmmMachineModuleStr = "Port Receive State Machine"

const (
	PpmmStateNone = iota + 1
	PpmmStateCheckingRSTP
	PpmmStateSelectingSTP
	PpmmStateSensing
)

var PpmmStateStrMap map[fsm.State]string

func PpmmMachineStrStateMapInit() {
	PpmmStateStrMap = make(map[fsm.State]string)
	PpmmStateStrMap[PpmmStateNone] = "None"
	PpmmStateStrMap[PpmmStateCheckingRSTP] = "Checking RSTP"
	PpmmStateStrMap[PpmmStateSelectingSTP] = "Selecting STP"
	PpmmStateStrMap[PpmmStateSensing] = "Sensing"
}

const (
	PpmmEventBegin = iota + 1
	PpmmEventMdelayNotEqualMigrateTimeAndNotPortEnabled
	PpmmEventNotPortEnabled
	PpmmEventMcheck
	PpmmEventRstpVersionAndNotSendRSTPAndRcvdRSTP
	PpmmEventMdelayWhileEqualZero
	PpmmEventSendRSTPAndRcvdSTP
)

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type PpmmMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PpmmEvents chan MachineEvent
	// stop go routine
	PpmmKillSignalEvent chan bool
	// enable logging
	PpmmLogEnableEvent chan bool
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPpmmMachine(p *StpPort) *PpmmMachine {
	ppmm := &PpmmMachine{
		p:                   p,
		PpmmEvents:          make(chan MachineEvent, 10),
		PpmmKillSignalEvent: make(chan bool),
		PpmmLogEnableEvent:  make(chan bool)}

	p.PpmmMachineFsm = ppmm

	return ppmm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (ppmm *PpmmMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if ppmm.Machine == nil {
		ppmm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	ppmm.Machine.Rules = r
	ppmm.Machine.Curr = &StpStateEvent{
		strStateMap: PpmmStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PpmmMachineModuleStr,
		ps:     PpmmStateNone,
		s:      PpmmStateNone,
	}

	return ppmm.Machine
}

// Stop should clean up all resources
func (ppmm *PpmmMachine) Stop() {

	// stop the go routine
	ppmm.PpmmKillSignalEvent <- true

	close(ppmm.PpmmEvents)
	close(ppmm.PpmmLogEnableEvent)
	close(ppmm.PpmmKillSignalEvent)

}

func (ppmm *PpmmMachine) InformPtxMachineSendRSTPChanged() {
	p := ppmm.p
	if p.PtxmMachineFsm != nil &&
		p.PtxmMachineFsm.Machine.Curr.CurrentState() == PtxmStateIdle {
		/*fmt.Println(p.SendRSTP,
		p.NewInfo,
		p.TxCount < TransmitHoldCountDefault,
		p.HelloWhenTimer.count != 0,
		p.Selected,
		p.UpdtInfo)*/
		if p.SendRSTP == true &&
			p.NewInfo == true &&
			p.TxCount < TransmitHoldCountDefault &&
			p.HelloWhenTimer.count != 0 &&
			p.Selected == true &&
			p.UpdtInfo == false {
			p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
				e:   PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
				src: PpmmMachineModuleStr}
		} else if p.SendRSTP == false &&
			p.NewInfo == true &&
			p.Role == PortRoleRootPort &&
			p.TxCount < TransmitHoldCountDefault &&
			p.HelloWhenTimer.count != 0 &&
			p.Selected == true &&
			p.UpdtInfo == false {
			p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
				e:   PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
				src: PpmmMachineModuleStr}
		} else if p.SendRSTP == false &&
			p.NewInfo == true &&
			p.Role == PortRoleDesignatedPort &&
			p.TxCount < TransmitHoldCountDefault &&
			p.HelloWhenTimer.count != 0 &&
			p.Selected == true &&
			p.UpdtInfo == false {
			p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
				e:   PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
				src: PpmmMachineModuleStr}
		}
	}
}

// PpmMachineCheckingRSTP
func (ppmm *PpmmMachine) PpmmMachineCheckingRSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p
	p.Mcheck = false

	sendRSTPchanged := p.SendRSTP != (p.BridgeProtocolVersionGet() == layers.RSTPProtocolVersion)
	p.MdelayWhiletimer.count = MigrateTimeDefault

	if sendRSTPchanged {
		p.SendRSTP = p.BridgeProtocolVersionGet() == layers.RSTPProtocolVersion
		// 17.24
		// Inform Port Transmit State Machine what STP version to send and which BPDU types
		// to support interoperability
		ppmm.InformPtxMachineSendRSTPChanged()
	}

	return PpmmStateCheckingRSTP
}

// PpmmMachineSelectingSTP
func (ppmm *PpmmMachine) PpmmMachineSelectingSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p

	sendRSTPchanged := p.SendRSTP != false
	p.MdelayWhiletimer.count = MigrateTimeDefault
	if sendRSTPchanged {
		p.SendRSTP = false
		// 17.24
		// Inform Port Transmit State Machine what STP version to send and which BPDU types
		// to support interoperability
		ppmm.InformPtxMachineSendRSTPChanged()
	}

	return PpmmStateSelectingSTP
}

// PpmmMachineSelectingSTP
func (ppmm *PpmmMachine) PpmmMachineSensing(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p

	p.RcvdRSTP = false
	p.RcvdSTP = false

	return PpmmStateSensing
}

func PpmmMachineFSMBuild(p *StpPort) *PpmmMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	ppmm := NewStpPpmmMachine(p)

	//BEGIN -> CHECKING RSTP
	rules.AddRule(PpmmStateNone, PpmmEventBegin, ppmm.PpmmMachineCheckingRSTP)
	rules.AddRule(PpmmStateCheckingRSTP, PpmmEventBegin, ppmm.PpmmMachineCheckingRSTP)
	rules.AddRule(PpmmStateSelectingSTP, PpmmEventBegin, ppmm.PpmmMachineCheckingRSTP)
	rules.AddRule(PpmmStateSensing, PpmmEventBegin, ppmm.PpmmMachineCheckingRSTP)

	// mdelayWhile != MigrateTime and not portEnable	 -> CHECKING RSTP
	rules.AddRule(PpmmStateCheckingRSTP, PpmmEventMdelayNotEqualMigrateTimeAndNotPortEnabled, ppmm.PpmmMachineCheckingRSTP)

	// NOT PORT ENABLED -> SENSING/CHECKING RSTP
	rules.AddRule(PpmmStateSelectingSTP, PpmmEventNotPortEnabled, ppmm.PpmmMachineSensing)
	rules.AddRule(PpmmStateSensing, PpmmEventNotPortEnabled, ppmm.PpmmMachineCheckingRSTP)

	// MCHECK ->  SENSING/CHECKING RSTP
	rules.AddRule(PpmmStateSelectingSTP, PpmmEventMcheck, ppmm.PpmmMachineSensing)
	rules.AddRule(PpmmStateSensing, PpmmEventMcheck, ppmm.PpmmMachineCheckingRSTP)

	// RSTP VERSION and NOT SENDING RSTP AND RCVD RSTP -> CHECKING RSTP
	rules.AddRule(PpmmStateSensing, PpmmEventRstpVersionAndNotSendRSTPAndRcvdRSTP, ppmm.PpmmMachineCheckingRSTP)

	//MDELAYWHILE EQUALS ZERO -> SENSING
	rules.AddRule(PpmmStateCheckingRSTP, PpmmEventMdelayWhileEqualZero, ppmm.PpmmMachineSensing)
	rules.AddRule(PpmmStateSelectingSTP, PpmmEventMdelayWhileEqualZero, ppmm.PpmmMachineSensing)

	// SEND RSTP and RCVD STP
	rules.AddRule(PpmmStateSensing, PpmmEventSendRSTPAndRcvdSTP, ppmm.PpmmMachineSelectingSTP)

	// Create a new FSM and apply the rules
	ppmm.Apply(&rules)

	return ppmm
}

// PrxmMachineMain:
func (p *StpPort) PpmmMachineMain() {

	// Build the State machine for STP Receive Machine according to
	// 802.1d Section 17.23
	ppmm := PpmmMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	ppmm.Machine.Start(ppmm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PpmmMachine) {
		StpLogger("INFO", "PPMM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PpmmKillSignalEvent:
				StpLogger("INFO", "PPMM: Machine End")
				return

			case event := <-m.PpmmEvents:
				//fmt.Println("Event Rx", event.src, event.e, PpmmStateStrMap[m.Machine.Curr.CurrentState()])
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				} else {

					// post processing
					if m.Machine.Curr.CurrentState() == PpmmStateSensing {
						if !m.p.PortEnabled {
							rv := m.Machine.ProcessEvent(PpmmMachineModuleStr, PpmmEventNotPortEnabled, nil)
							if rv != nil {
								StpLogger("INFO", fmt.Sprintf("%s\n", rv))
							}
						} else if m.p.Mcheck {
							rv := m.Machine.ProcessEvent(PpmmMachineModuleStr, PpmmEventMcheck, nil)
							if rv != nil {
								StpLogger("INFO", fmt.Sprintf("%s\n", rv))
							}
						} else if p.BridgeProtocolVersionGet() == layers.RSTPProtocolVersion &&
							!p.SendRSTP &&
							p.RcvdRSTP {
							rv := m.Machine.ProcessEvent(PpmmMachineModuleStr, PpmmEventRstpVersionAndNotSendRSTPAndRcvdRSTP, nil)
							if rv != nil {
								StpLogger("INFO", fmt.Sprintf("%s\n", rv))
							}
						}
					}
				}

				if event.responseChan != nil {
					SendResponse(PpmmMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PpmmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(ppmm)
}
