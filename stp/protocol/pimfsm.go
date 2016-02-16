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
	"github.com/google/gopacket/layers"
	//"reflect"
	"utils/fsm"
)

const PimMachineModuleStr = "Port Information State Machine"

const (
	PimStateNone = iota + 1
	PimStateDisabled
	PimStateAged
	PimStateUpdate
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
	PimStateStrMap[PimStateUpdate] = "Updated"
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
	PimEventNotPortEnabledInfoIsNotEqualDisabled
	PimEventRcvdMsg
	PimEventRcvdMsgAndNotUpdtInfo
	PimEventPortEnabled
	PimEventSelectedAndUpdtInfo
	PimEventUnconditionalFallThrough
	PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg
	PimEventRcvdInfoEqualSuperiorDesignatedInfo
	PimEventRcvdInfoEqualRepeatedDesignatedInfo
	PimEventRcvdInfoEqualInferiorDesignatedInfo
	PimEventRcvdInfoEqualInferiorRootAlternateInfo
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

func (pim *PimMachine) PimLogger(s string) {
	StpMachineLogger("INFO", "PIM", pim.p.IfIndex, s)
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
		logger: pim.PimLogger,
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

// PimMachineDisabled
func (pim *PimMachine) PimMachineDisabled(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	defer p.NotifyRcvdMsgChanged(PimMachineModuleStr, p.RcvdMsg, false, data)
	p.RcvdMsg = false
	defer p.NotifyProposingChanged(PimMachineModuleStr, p.Proposing, false)
	p.Proposing = false
	defer pim.NotifyProposedChanged(p.Proposed, false)
	p.Proposed = false
	defer pim.NotifyAgreeChanged(p.Agree, false)
	p.Agree = false
	defer pim.NotifyAgreedChanged(p.Agreed, false)
	p.Agreed = false
	p.RcvdInfoWhiletimer.count = 0
	p.InfoIs = PortInfoStateDisabled
	defer p.NotifySelectedChanged(PimMachineModuleStr, p.Selected, false)
	p.Selected = false
	defer pim.NotifyReselectChanged(p.Reselect, true)
	p.Reselect = true
	return PimStateDisabled
}

// PimMachineAged
func (pim *PimMachine) PimMachineAged(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.InfoIs = PortInfoStateAged
	defer p.NotifySelectedChanged(PimMachineModuleStr, p.Selected, false)
	p.Selected = false
	defer pim.NotifyReselectChanged(p.Reselect, true)
	p.Reselect = true
	return PimStateAged
}

// PimMachineUpdate
func (pim *PimMachine) PimMachineUpdate(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	defer p.NotifyProposingChanged(PimMachineModuleStr, p.Proposing, false)
	p.Proposing = false
	defer pim.NotifyProposedChanged(p.Proposed, false)
	p.Proposed = false
	tmp := p.Agreed && pim.betterorsameinfo(p.InfoIs)
	defer pim.NotifyAgreedChanged(p.Agreed, tmp)
	p.Agreed = tmp
	tmp = p.Synced && p.Agreed
	defer pim.NotifySyncedChanged(p.Synced, tmp)
	p.Synced = tmp
	p.PortPriority = p.DesignatedPriority
	p.PortTimes = p.DesignatedTimes
	defer p.NotifyUpdtInfoChanged(PimMachineModuleStr, p.UpdtInfo, false)
	p.UpdtInfo = false
	p.InfoIs = PortInfoStateMine
	defer pim.NotifyNewInfoChange(p.NewInfo, true)
	p.NewInfo = true
	return PimStateUpdate
}

// PimMachineCurrent
func (pim *PimMachine) PimMachineCurrent(m fsm.Machine, data interface{}) fsm.State {
	return PimStateCurrent
}

// PimMachineReceive
func (pim *PimMachine) PimMachineReceive(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.RcvdInfo = pim.rcvInfo(data)
	return PimStateReceive
}

// PimMachineSuperiorDesignated
func (pim *PimMachine) PimMachineSuperiorDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	defer pim.NotifyAgreedChanged(p.Agreed, false)
	p.Agreed = false
	defer p.NotifyProposingChanged(PimMachineModuleStr, p.Proposing, false)
	p.Proposing = false
	flags := pim.getRcvdMsgFlags(data)
	pim.recordProposal(flags)
	pim.setTcFlags(flags, data)
	betterorsame := pim.recordPriority(pim.getRcvdMsgPriority(data))
	pim.recordTimes(pim.getRcvdMsgTimes(data))
	tmp := p.Agree && betterorsame
	defer pim.NotifyAgreeChanged(p.Agree, tmp)
	p.Agree = tmp
	pim.updtRcvdInfoWhile()
	p.InfoIs = PortInfoStateReceived
	defer p.NotifySelectedChanged(PimMachineModuleStr, p.Selected, false)
	p.Selected = false
	defer p.NotifyRcvdMsgChanged(PimMachineModuleStr, p.RcvdMsg, false, data)
	p.RcvdMsg = false
	defer pim.NotifyReselectChanged(p.Reselect, true)
	p.Reselect = true
	return PimStateSuperiorDesignated
}

// PimMachineRepeatedDesignated
func (pim *PimMachine) PimMachineRepeatedDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	flags := pim.getRcvdMsgFlags(data)
	pim.recordProposal(flags)
	pim.setTcFlags(flags, data)
	pim.updtRcvdInfoWhile()
	defer p.NotifyRcvdMsgChanged(PimMachineModuleStr, p.RcvdMsg, false, data)
	p.RcvdMsg = false
	return PimStateRepeatedDesignated
}

// PimMachineInferiorDesignated
func (pim *PimMachine) PimMachineInferiorDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	flags := pim.getRcvdMsgFlags(data)
	pim.recordDispute(flags)
	defer p.NotifyRcvdMsgChanged(PimMachineModuleStr, p.RcvdMsg, false, data)
	p.RcvdMsg = false
	return PimStateInferiorDesignated
}

// PimMachineNotDesignated
func (pim *PimMachine) PimMachineNotDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	flags := pim.getRcvdMsgFlags(data)
	pim.recordAgreement(flags)
	pim.setTcFlags(flags, data)
	defer p.NotifyRcvdMsgChanged(PimMachineModuleStr, p.RcvdMsg, false, data)
	p.RcvdMsg = false
	return PimStateNotDesignated
}

// PimMachineOther
func (pim *PimMachine) PimMachineOther(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	defer p.NotifyRcvdMsgChanged(PimMachineModuleStr, p.RcvdMsg, false, data)
	p.RcvdMsg = false
	return PimStateOther
}

func PimMachineFSMBuild(p *StpPort) *PimMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	pim := NewStpPimMachine(p)
	/*
		PimStateDisabled
		PimStateAged
		PimStateUpdate
		PimStateSuperiorDesignated
		PimStateRepeatedDesignated
		PimStateInferiorDesignated
		PimStateNotDesignated
		PimStateOther
		PimStateCurrent
		PimStateReceive
	*/
	// BEGIN -> DISABLED
	rules.AddRule(PimStateNone, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateDisabled, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateAged, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateUpdate, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateSuperiorDesignated, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateRepeatedDesignated, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateInferiorDesignated, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateNotDesignated, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateOther, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateCurrent, PimEventBegin, pim.PimMachineDisabled)
	rules.AddRule(PimStateReceive, PimEventBegin, pim.PimMachineDisabled)

	// RCVDMSG -> DISABLED
	rules.AddRule(PimStateDisabled, PimEventRcvdMsg, pim.PimMachineDisabled)

	// NOT PORT ENABLED and INFOLS != DISABLED -> DISABLED
	rules.AddRule(PimStateDisabled, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateAged, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateUpdate, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateSuperiorDesignated, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateRepeatedDesignated, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateInferiorDesignated, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateNotDesignated, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateOther, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateCurrent, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)
	rules.AddRule(PimStateReceive, PimEventNotPortEnabledInfoIsNotEqualDisabled, pim.PimMachineDisabled)

	// PORT ENABLED -> AGED
	rules.AddRule(PimStateDisabled, PimEventPortEnabled, pim.PimMachineAged)

	// SELECTED and  UPDTINFO -> UPDATED
	rules.AddRule(PimStateAged, PimEventSelectedAndUpdtInfo, pim.PimMachineUpdate)
	rules.AddRule(PimStateCurrent, PimEventSelectedAndUpdtInfo, pim.PimMachineUpdate)

	// UNCONDITIONAL FALL THROUGH -> CURRENT
	rules.AddRule(PimStateUpdate, PimEventUnconditionalFallThrough, pim.PimMachineCurrent)
	rules.AddRule(PimStateSuperiorDesignated, PimEventUnconditionalFallThrough, pim.PimMachineCurrent)
	rules.AddRule(PimStateRepeatedDesignated, PimEventUnconditionalFallThrough, pim.PimMachineCurrent)
	rules.AddRule(PimStateInferiorDesignated, PimEventUnconditionalFallThrough, pim.PimMachineCurrent)
	rules.AddRule(PimStateNotDesignated, PimEventUnconditionalFallThrough, pim.PimMachineCurrent)
	rules.AddRule(PimStateOther, PimEventUnconditionalFallThrough, pim.PimMachineCurrent)

	// INFOIS == RECEIVED nad RCVDINFOWHILE == 0 and NOT UPDTINFO and NOT RCVDMSG ->  AGED
	rules.AddRule(PimStateCurrent, PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg, pim.PimMachineAged)

	// RCVDMSG and NOT UPDTINFO -> RECEIVE
	rules.AddRule(PimStateCurrent, PimEventRcvdMsgAndNotUpdtInfo, pim.PimMachineReceive)

	// RCVDINFO == SUPERIORDESIGNATEDINFO -> SUPERIOR DESIGNATED
	rules.AddRule(PimStateReceive, PimEventRcvdInfoEqualSuperiorDesignatedInfo, pim.PimMachineSuperiorDesignated)

	// RCVDINFO == REPEATEDDESIGNATEDINFO -> REPEATED DESIGNATED
	rules.AddRule(PimStateReceive, PimEventRcvdInfoEqualRepeatedDesignatedInfo, pim.PimMachineRepeatedDesignated)

	// RCVDINFO == INFERIORDESIGNATEDINFO -> INFERIOR DESIGNATED
	rules.AddRule(PimStateReceive, PimEventRcvdInfoEqualInferiorDesignatedInfo, pim.PimMachineInferiorDesignated)

	// RCVDINFO == NOTDESIGNATEDINFO -> NOT DESIGNATED
	rules.AddRule(PimStateReceive, PimEventRcvdInfoEqualInferiorRootAlternateInfo, pim.PimMachineNotDesignated)

	// RCVDINFO == INFERIORROOTALTERNATEINFO -> OTHER
	rules.AddRule(PimStateReceive, PimEventRcvdInfoEqualOtherInfo, pim.PimMachineOther)

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
		StpMachineLogger("INFO", "PIM", p.IfIndex, "Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PimKillSignalEvent:
				StpMachineLogger("INFO", "PIM", p.IfIndex, "Machine End")
				return

			case event := <-m.PimEvents:
				StpMachineLogger("INFO", "PIM", p.IfIndex, fmt.Sprintf("Event Rx src[%s] event[%d] data[%#v]", event.src, event.e, event.data))
				rv := m.Machine.ProcessEvent(event.src, event.e, event.data)
				if rv != nil {
					StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, event.e, PimStateStrMap[m.Machine.Curr.CurrentState()]))
				} else {
					// POST events
					m.ProcessPostStateProcessing(event.data)
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

func (pim *PimMachine) ProcessingPostStateDisabled(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateDisabled {
		if p.PortEnabled {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventPortEnabled, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventPortEnabled, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.RcvdMsg {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdMsg, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdMsg, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		}
	}
}

func (pim *PimMachine) ProcessingPostStateAged(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateAged {
		if p.Selected &&
			p.UpdtInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventSelectedAndUpdtInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventSelectedAndUpdtInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		}
	}
}

func (pim *PimMachine) ProcessingPostStateUpdate(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateUpdate {
		rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
		} else {
			pim.ProcessPostStateProcessing(data)
		}
	}
}

func (pim *PimMachine) ProcessingPostStateSuperiorDesignated(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateSuperiorDesignated {
		rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
		} else {
			pim.ProcessPostStateProcessing(data)
		}
	}
}
func (pim *PimMachine) ProcessingPostStateRepeatedDesignated(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateRepeatedDesignated {
		rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
		} else {
			pim.ProcessPostStateProcessing(data)
		}
	}
}
func (pim *PimMachine) ProcessingPostStateInferiorDesignated(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateInferiorDesignated {
		rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
		} else {
			pim.ProcessPostStateProcessing(data)
		}
	}
}
func (pim *PimMachine) ProcessingPostStateNotDesignated(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateNotDesignated {
		rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
		} else {
			pim.ProcessPostStateProcessing(data)
		}
	}
}
func (pim *PimMachine) ProcessingPostStateOther(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateOther {
		rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
		} else {
			pim.ProcessPostStateProcessing(data)
		}
	}
}
func (pim *PimMachine) ProcessingPostStateCurrent(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateCurrent {
		if p.Selected &&
			p.UpdtInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventSelectedAndUpdtInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventSelectedAndUpdtInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.InfoIs == PortInfoStateReceived &&
			p.RcvdInfoWhiletimer.count == 0 &&
			!p.UpdtInfo &&
			!p.RcvdMsg {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.RcvdMsg &&
			!p.UpdtInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdMsgAndNotUpdtInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdMsgAndNotUpdtInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		}
	}
}

func (pim *PimMachine) ProcessingPostStateReceive(data interface{}) {
	p := pim.p
	if pim.Machine.Curr.CurrentState() == PimStateReceive {
		if p.RcvdInfo == SuperiorDesignatedInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualSuperiorDesignatedInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualSuperiorDesignatedInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.RcvdInfo == RepeatedDesignatedInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualRepeatedDesignatedInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualRepeatedDesignatedInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.RcvdInfo == InferiorDesignatedInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualInferiorDesignatedInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualRepeatedDesignatedInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.RcvdInfo == InferiorRootAlternateInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualInferiorRootAlternateInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualInferiorRootAlternateInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		} else if p.RcvdInfo == OtherInfo {
			rv := pim.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualOtherInfo, data)
			if rv != nil {
				StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualOtherInfo, PimStateStrMap[pim.Machine.Curr.CurrentState()]))
			} else {
				pim.ProcessPostStateProcessing(data)
			}
		}
	}
}

func (pim *PimMachine) ProcessPostStateProcessing(data interface{}) {

	pim.ProcessingPostStateDisabled(data)
	pim.ProcessingPostStateAged(data)
	pim.ProcessingPostStateUpdate(data)
	pim.ProcessingPostStateSuperiorDesignated(data)
	pim.ProcessingPostStateRepeatedDesignated(data)
	pim.ProcessingPostStateInferiorDesignated(data)
	pim.ProcessingPostStateNotDesignated(data)
	pim.ProcessingPostStateOther(data)
	pim.ProcessingPostStateCurrent(data)
	pim.ProcessingPostStateReceive(data)

}

func (pim *PimMachine) NotifySyncedChanged(oldsynced bool, newsynced bool) {
	p := pim.p
	if oldsynced != newsynced {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDisabledPort {
			if !p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
			if p.b.AllSynced() &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
			if !p.Learning &&
				!p.Forwarding &&
				!p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.OperEdge &&
				!p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventOperEdgeAndNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Sync &&
				p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSyncAndSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Sync &&
				!p.Synced &&
				!p.OperEdge &&
				p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Sync &&
				!p.Synced &&
				!p.OperEdge &&
				p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventSyncAndNotSyncedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateAlternatePort {
			if p.b.AllSynced() &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if !p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		}
	}
}

func (pim *PimMachine) NotifyAgreedChanged(oldagreed bool, newagreed bool) {
	p := pim.p
	if oldagreed != newagreed {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
			if !p.Forward &&
				!p.Agreed &&
				!p.Proposing &&
				!p.OperEdge &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				p.RrWhileTimer.count == 0 &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.ReRoot &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				p.RrWhileTimer.count == 0 &&
				!p.Sync &&
				p.Learn &&
				!p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.ReRoot &&
				!p.Sync &&
				p.Learn &&
				!p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}

		}
	}
}

// notify the prt machine
func (pim *PimMachine) NotifyAgreeChanged(oldagree bool, newagree bool) {
	p := pim.p
	if oldagree != newagree {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
			if p.Proposed &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.b.AllSynced() &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Proposed &&
				p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
			if !p.Forward &&
				!p.Agreed &&
				!p.Proposing &&
				!p.OperEdge &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.Synced &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				p.RrWhileTimer.count == 0 &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.ReRoot &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				p.RrWhileTimer.count == 0 &&
				!p.Sync &&
				p.Learn &&
				!p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Agreed &&
				!p.ReRoot &&
				!p.Sync &&
				!p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateAlternatePort {
			if p.Proposed &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.b.AllSynced() &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Proposed &&
				p.Agree &&
				p.Proposed &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		}
	}
}

func (pim *PimMachine) NotifyProposedChanged(oldproposed bool, newproposed bool) {
	p := pim.p
	if oldproposed != newproposed {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateRootPort {
			if p.Proposed &&
				p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Proposed &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		} else if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateAlternatePort {
			if p.Proposed &&
				!p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Proposed &&
				p.Agree &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		}
	}
}

func (pim *PimMachine) NotifyReselectChanged(oldreselect bool, newreselect bool) {
	p := pim.p
	if newreselect {
		if p.b.PrsMachineFsm.Machine.Curr.CurrentState() == PrsStateRoleSelection {
			p.b.PrsMachineFsm.PrsEvents <- MachineEvent{
				e:   PrsEventReselect,
				src: PimMachineModuleStr,
			}
		}
	}
}

func (pim *PimMachine) NotifyNewInfoChange(oldnewinfo bool, newnewinfo bool) {
	p := pim.p
	if oldnewinfo != newnewinfo {
		if p.PtxmMachineFsm.Machine.Curr.CurrentState() == PtxmStateIdle {
			if !p.SendRSTP &&
				p.NewInfo &&
				p.Role == PortRoleDesignatedPort &&
				p.TxCount < p.b.TxHoldCount &&
				p.HelloWhenTimer.count != 0 &&
				p.Selected &&
				!p.UpdtInfo {
				p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
					e:   PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if !p.SendRSTP &&
				p.NewInfo &&
				p.Role == PortRoleRootPort &&
				p.HelloWhenTimer.count != 0 &&
				p.Selected &&
				!p.UpdtInfo {
				p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
					e:   PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			}
		}
	}
}

func (pim *PimMachine) NotifyDisputedChanged(olddisputed bool, newdisputed bool) {
	p := pim.p
	if olddisputed != newdisputed {
		if p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
			if p.Disputed &&
				!p.OperEdge &&
				p.Learn &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventDisputedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo,
					src: PimMachineModuleStr,
				}
			} else if p.Disputed &&
				!p.OperEdge &&
				p.Forward &&
				p.Selected &&
				!p.UpdtInfo {
				p.PrtMachineFsm.PrtEvents <- MachineEvent{
					e:   PrtEventDisputedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo,
					src: PrsMachineModuleStr,
				}
			}
		}
	}
}

func (pim *PimMachine) rcvInfo(data interface{}) PortDesignatedRcvInfo {
	p := pim.p
	msgRole := StpGetBpduRole(pim.getRcvdMsgFlags(data))
	msgpriority := pim.getRcvdMsgPriority(data)
	msgtimes := pim.getRcvdMsgTimes(data)

	StpMachineLogger("INFO", "PIM", p.IfIndex, fmt.Sprintf("role[%d] msgVector[%#v] portVector[%#v] msgTimes[%#v] designatedTimes[%#v]", msgRole, msgpriority, p.PortPriority, msgtimes, p.DesignatedTimes))
	if msgRole == PortRoleDesignatedPort &&
		(IsMsgPriorityVectorSuperiorThanPortPriorityVector(msgpriority, &p.PortPriority) ||
			(*msgpriority == p.PortPriority &&
				*msgtimes != p.PortTimes)) {
		return SuperiorDesignatedInfo
	} else if msgRole == PortRoleDesignatedPort &&
		*msgpriority == p.PortPriority &&
		*msgtimes == p.PortTimes {
		return RepeatedDesignatedInfo
	} else if msgRole == PortRoleDesignatedPort &&
		IsMsgPriorityVectorWorseThanPortPriorityVector(msgpriority, &p.PortPriority) {
		return InferiorDesignatedInfo
	} else if (msgRole == PortRoleRootPort ||
		msgRole == PortRoleAlternatePort ||
		msgRole == PortRoleBackupPort) &&
		IsMsgPriorityVectorTheSameOrWorseThanPortPriorityVector(msgpriority, &p.PortPriority) {
		return InferiorRootAlternateInfo
	} else {
		return OtherInfo
	}
}

func (pim *PimMachine) getRcvdMsgFlags(bpduLayer interface{}) uint8 {
	p := pim.p
	//bpdumsg := data.(RxBpduPdu)
	//packet := bpdumsg.pdu.(gopacket.Packet)
	//bpduLayer := packet.Layer(layers.LayerTypeBPDU)

	var flags uint8
	switch bpduLayer.(type) {
	case *layers.STP:
		StpMachineLogger("INFO", "PIM", p.IfIndex, "Found STP frame getting flags")
		stp := bpduLayer.(*layers.STP)
		flags = stp.Flags
	case *layers.RSTP:
		StpMachineLogger("INFO", "PIM", p.IfIndex, "Found RSTP frame getting flags")
		rstp := bpduLayer.(*layers.RSTP)
		flags = rstp.Flags
	case *layers.PVST:

		StpMachineLogger("INFO", "PIM", p.IfIndex, "Found PVST frame getting flags")
		pvst := bpduLayer.(*layers.STP)
		flags = pvst.Flags
	default:
		StpMachineLogger("ERROR", "PIM", p.IfIndex, fmt.Sprintf("%T\n", bpduLayer))
	}
	return flags
}

func (pim *PimMachine) getRcvdMsgPriority(bpduLayer interface{}) (msgpriority *PriorityVector) {
	msgpriority = &PriorityVector{}

	switch bpduLayer.(type) {
	case *layers.STP:
		stp := bpduLayer.(*layers.STP)
		msgpriority.RootBridgeId = stp.RootId
		msgpriority.RootPathCost = stp.RootPathCost
		msgpriority.DesignatedBridgeId = stp.BridgeId
		msgpriority.DesignatedPortId = stp.PortId
		msgpriority.BridgePortId = stp.PortId
	case *layers.RSTP:
		rstp := bpduLayer.(*layers.RSTP)
		msgpriority.RootBridgeId = rstp.RootId
		msgpriority.RootPathCost = rstp.RootPathCost
		msgpriority.DesignatedBridgeId = rstp.BridgeId
		msgpriority.DesignatedPortId = rstp.PortId
		msgpriority.BridgePortId = rstp.PortId
	case *layers.PVST:
		pvst := bpduLayer.(*layers.STP)
		msgpriority.RootBridgeId = pvst.RootId
		msgpriority.RootPathCost = pvst.RootPathCost
		msgpriority.DesignatedBridgeId = pvst.BridgeId
		msgpriority.DesignatedPortId = pvst.PortId
		msgpriority.BridgePortId = pvst.PortId
	}
	return msgpriority
}

func (pim *PimMachine) getRcvdMsgTimes(bpduLayer interface{}) (msgtimes *Times) {
	msgtimes = &Times{}

	switch bpduLayer.(type) {
	case *layers.STP:
		stp := bpduLayer.(*layers.STP)
		msgtimes.MessageAge = stp.MsgAge
		msgtimes.MaxAge = stp.MaxAge
		msgtimes.HelloTime = stp.HelloTime
		msgtimes.ForwardingDelay = stp.FwdDelay
	case *layers.RSTP:
		rstp := bpduLayer.(*layers.RSTP)
		msgtimes.MessageAge = rstp.MsgAge
		msgtimes.MaxAge = rstp.MaxAge
		msgtimes.HelloTime = rstp.HelloTime
		msgtimes.ForwardingDelay = rstp.FwdDelay
	case *layers.PVST:
		pvst := bpduLayer.(*layers.STP)
		msgtimes.MessageAge = pvst.MsgAge
		msgtimes.MaxAge = pvst.MaxAge
		msgtimes.HelloTime = pvst.HelloTime
		msgtimes.ForwardingDelay = pvst.FwdDelay
	}
	return msgtimes
}

// recordProposal(): 17.21.11
func (pim *PimMachine) recordProposal(rcvdMsgFlags uint8) {
	p := pim.p
	if StpGetBpduRole(rcvdMsgFlags) == PortRoleDesignatedPort &&
		StpGetBpduProposal(rcvdMsgFlags) {
		StpMachineLogger("INFO", "PIM", p.IfIndex, "recording proposal set")
		defer pim.NotifyProposedChanged(p.Proposed, true)
		p.Proposed = true
	}
}

// setTcFlags 17,21,17
func (pim *PimMachine) setTcFlags(rcvdMsgFlags uint8, bpduLayer interface{}) {
	p := pim.p
	//bpdumsg := data.(RxBpduPdu)
	//packet := bpdumsg.pdu.(gopacket.Packet)
	//bpduLayer := packet.Layer(layers.LayerTypeBPDU)

	p.RcvdTcAck = StpGetBpduTopoChangeAck(rcvdMsgFlags)

	p.RcvdTc = StpGetBpduTopoChange(rcvdMsgFlags)

	switch bpduLayer.(type) {
	case *layers.BPDUTopology:
		p.RcvdTcn = true
	}
}

// betterorsameinfo 17.21.1
func (pim *PimMachine) betterorsameinfo(newInfoIs PortInfoState) bool {
	p := pim.p

	// recordPriority should be called when this is called from superior designated
	// this way we don't need to pass the message around
	if (newInfoIs == PortInfoStateReceived &&
		p.InfoIs == PortInfoStateReceived &&
		IsMsgPriorityVectorSuperiorThanPortPriorityVector(&p.MsgPriority, &p.PortPriority)) ||
		(newInfoIs == PortInfoStateMine &&
			p.InfoIs == PortInfoStateMine &&
			IsMsgPriorityVectorSuperiorThanPortPriorityVector(&p.DesignatedPriority, &p.PortPriority)) {
		return true
	}
	return false
}

// recordPriority  17.21.12
func (pim *PimMachine) recordPriority(rcvdMsgPriority *PriorityVector) bool {
	p := pim.p
	p.MsgPriority = *rcvdMsgPriority
	// 17.6
	//This message priority vector is superior to the port priority vector and will replace it if, and only if, the
	//message priority vector is better than the port priority vector, or the message has been transmitted from the
	//same Designated Bridge and Designated Port as the port priority vector, i.e., if the following is true
	betterorsame := pim.betterorsameinfo(p.InfoIs)
	if betterorsame {
		p.PortPriority = *rcvdMsgPriority
	}
	return betterorsame
}

// recordTimes 17.21.13
func (pim *PimMachine) recordTimes(rcvdMsgTimes *Times) {
	p := pim.p
	p.PortTimes.ForwardingDelay = rcvdMsgTimes.ForwardingDelay
	if rcvdMsgTimes.HelloTime > BridgeHelloTimeMin {
		p.PortTimes.HelloTime = rcvdMsgTimes.HelloTime
	} else {
		p.PortTimes.HelloTime = BridgeHelloTimeMin
	}
	p.PortTimes.MaxAge = rcvdMsgTimes.MaxAge
	p.PortTimes.MessageAge = rcvdMsgTimes.MessageAge
}

// updtRcvdInfoWhile 17.21.23
func (pim *PimMachine) updtRcvdInfoWhile() {
	p := pim.p
	if p.PortTimes.MessageAge+1 <= p.PortTimes.MaxAge {
		p.RcvdInfoWhiletimer.count = 3 * int32(p.PortTimes.HelloTime)
	} else {
		p.RcvdInfoWhiletimer.count = 0
		// TODO what happens when this is set
	}
}

func (pim *PimMachine) recordDispute(rcvdMsgFlags uint8) {
	p := pim.p
	msgRole := StpGetBpduRole(rcvdMsgFlags)

	if msgRole == PortRoleDesignatedPort &&
		(StpGetBpduLearning(rcvdMsgFlags) ||
			StpGetBpduForwarding(rcvdMsgFlags)) {
		pim.NotifyDisputedChanged(p.Disputed, true)
		p.Disputed = true
		defer pim.NotifyAgreedChanged(p.Agreed, false)
		p.Agreed = false

		//defer pim.NotifyAgreedChanged(p.Agreed, true)
		//p.Agreed = true
		//defer p.NotifyProposingChanged(PimMachineModuleStr, p.Proposing, false)
		//p.Proposing = false
	}
}

func (pim *PimMachine) recordAgreement(rcvdMsgFlags uint8) {
	p := pim.p
	if p.RstpVersion &&
		p.OperPointToPointMAC &&
		StpGetBpduAgreement(rcvdMsgFlags) {
		defer pim.NotifyAgreedChanged(p.Agreed, true)
		p.Agreed = true
		defer p.NotifyProposingChanged(PimMachineModuleStr, p.Proposing, false)
		p.Proposing = false
	} else {
		defer pim.NotifyAgreedChanged(p.Agreed, false)
		p.Agreed = false
	}

}
