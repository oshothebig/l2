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

// PimMachineDisabled
func (pim *PimMachine) PimMachineDisabled(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.RcvdMsg = false
	p.Proposing = false
	p.Proposed = false
	p.Agree = false
	p.Agreed = false
	p.RcvdInfoWhiletimer.count = 0
	p.InfoIs = PortInfoStateDisabled
	p.Selected = false
	pim.NotifyPrsMachineReselectChanged(true)
	return PimStateDisabled
}

// PimMachineAged
func (pim *PimMachine) PimMachineAged(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.InfoIs = PortInfoStateAged
	p.Selected = false
	pim.NotifyPrsMachineReselectChanged(true)
	return PimStateAged
}

// PimMachineUpdate
func (pim *PimMachine) PimMachineUpdate(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.Proposing = false
	p.Proposed = false
	p.Agreed = p.Agreed && pim.betterorsameinfo(p.InfoIs)
	p.Synced = p.Synced && p.Agreed
	p.PortPriority = p.DesignatedPriority
	p.PortTimes = p.DesignatedTimes
	p.UpdtInfo = false
	p.InfoIs = PortInfoStateMine
	pim.NotifyTxMachineNewInfoChange(true)
	return PimStateUpdate
}

// PimMachineCurrent
func (pim *PimMachine) PimMachineCurrent(m fsm.Machine, data interface{}) fsm.State {
	return PimStateCurrent
}

// PimMachineReceive
func (pim *PimMachine) PimMachineReceive(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.RcvdfInfo = pim.rcvInfo(StpGetBpduRole(pim.getRcvdMsgFlags(data)))
	return PimStateReceive
}

// PimMachineSuperiorDesignated
func (pim *PimMachine) PimMachineSuperiorDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p
	p.Agreed = false
	p.Proposing = false

	flags := pim.getRcvdMsgFlags(data)
	pim.recordProposal(flags)
	pim.setTcFlags(flags, data)
	p.Agree = p.Agree && pim.betterorsameinfo(p.InfoIs)
	pim.recordPriority(pim.getRcvdMsgPriority(data))
	pim.recordTimes(pim.getRcvdMsgTimes(data))
	pim.updtRcvdInfoWhile()
	p.InfoIs = PortInfoStateReceived

	p.Selected = false
	p.RcvdMsg = false
	pim.NotifyPrsMachineReselectChanged(true)
	return PimStateSuperiorDesignated
}

// PimMachineRepeatedDesignated
func (pim *PimMachine) PimMachineRepeatedDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	flags := pim.getRcvdMsgFlags(data)
	pim.recordProposal(flags)
	pim.setTcFlags(flags, data)
	pim.updtRcvdInfoWhile()
	p.RcvdMsg = false
	return PimStateRepeatedDesignated
}

// PimMachineInferiorDesignated
func (pim *PimMachine) PimMachineInferiorDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	flags := pim.getRcvdMsgFlags(data)
	pim.recordDispute(flags)
	p.RcvdMsg = false
	return PimStateInferiorDesignated
}

// PimMachineNotDesignated
func (pim *PimMachine) PimMachineNotDesignated(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

	flags := pim.getRcvdMsgFlags(data)
	pim.recordAgreement(flags)
	pim.setTcFlags(flags, data)
	p.RcvdMsg = false
	return PimStateNotDesignated
}

// PimMachineOther
func (pim *PimMachine) PimMachineOther(m fsm.Machine, data interface{}) fsm.State {
	p := pim.p

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
		StpMachineLogger("INFO", "PIM", "Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PimKillSignalEvent:
				StpMachineLogger("INFO", "PIM", "Machine End")
				return

			case event := <-m.PimEvents:
				StpMachineLogger("INFO", "PIM", fmt.Sprintf("Event Rx src[%s] event[%d] data[%#v]", event.src, event.e, event.data))
				rv := m.Machine.ProcessEvent(event.src, event.e, event.data)
				if rv != nil {
					StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, event.e, PimStateStrMap[m.Machine.Curr.CurrentState()]))
				} else {
					// POST events
					if m.Machine.Curr.CurrentState() == PimStateReceive &&
						p.RcvdfInfo == SuperiorDesignatedInfo {
						rv := m.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualSuperiorDesignatedInfo, event.data)
						if rv != nil {
							StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualSuperiorDesignatedInfo, PimStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					} else if m.Machine.Curr.CurrentState() == PimStateReceive &&
						p.RcvdfInfo == RepeatedDesignatedInfo {
						rv := m.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualRepeatedDesignatedInfo, event.data)
						if rv != nil {
							StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualRepeatedDesignatedInfo, PimStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					} else if m.Machine.Curr.CurrentState() == PimStateReceive &&
						p.RcvdfInfo == InferiorDesignatedInfo {
						rv := m.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualInferiorDesignatedInfo, event.data)
						if rv != nil {
							StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualInferiorDesignatedInfo, PimStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					} else if m.Machine.Curr.CurrentState() == PimStateReceive &&
						p.RcvdfInfo == InferiorRootAlternateInfo {
						rv := m.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualInferiorRootAlternateInfo, event.data)
						if rv != nil {
							StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualInferiorRootAlternateInfo, PimStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					} else if m.Machine.Curr.CurrentState() == PimStateReceive &&
						p.RcvdfInfo == OtherInfo {
						rv := m.Machine.ProcessEvent(PimMachineModuleStr, PimEventRcvdInfoEqualOtherInfo, event.data)
						if rv != nil {
							StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventRcvdInfoEqualOtherInfo, PimStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					}
					if m.Machine.Curr.CurrentState() == PimStateUpdate ||
						m.Machine.Curr.CurrentState() == PimStateSuperiorDesignated ||
						m.Machine.Curr.CurrentState() == PimStateRepeatedDesignated ||
						m.Machine.Curr.CurrentState() == PimStateInferiorDesignated ||
						m.Machine.Curr.CurrentState() == PimStateNotDesignated ||
						m.Machine.Curr.CurrentState() == PimStateOther {
						rv := m.Machine.ProcessEvent(PimMachineModuleStr, PimEventUnconditionalFallThrough, event.data)
						if rv != nil {
							StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%s event[%d] currState[%s]\n", rv, PimEventUnconditionalFallThrough, PimStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					}
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

func (pim *PimMachine) ProcessPostStateProcessing() {
}

func (pim *PimMachine) NotifyPrsMachineReselectChanged(reselect bool) {
	p := pim.p
	StpMachineLogger("INFO", "PIM", fmt.Sprintf("notify prs machine reselectold[%t] reselect[%t]\n", p.Reselect, reselect))
	if p.Reselect != reselect {
		p.Reselect = reselect
		p.b.PrsMachineFsm.PrsEvents <- MachineEvent{
			e:   PrsEventReselect,
			src: PimMachineModuleStr,
		}
	}
}

func (pim *PimMachine) NotifyTxMachineNewInfoChange(newInfo bool) {
	p := pim.p
	if p.NewInfo != newInfo {
		p.NewInfo = newInfo

		if !p.SendRSTP &&
			p.NewInfo &&
			p.Role == PortRoleDesignatedPort &&
			p.TxCount < TransmitHoldCountDefault &&
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

func (pim *PimMachine) rcvInfo(role PortRole) PortDesignatedRcvInfo {
	p := pim.p

	StpMachineLogger("INFO", "PIM", fmt.Sprintf("role[%d] msgVector[%#v]\ndesignatedVector[%#v]\n", role, p.MsgPriority, p.DesignatedPriority))
	if role == PortRoleDesignatedPort &&
		(IsMsgPriorityVectorSuperiorThanPortPriorityVector(&p.MsgPriority, &p.DesignatedPriority) ||
			p.MsgTimes != p.DesignatedTimes) {
		return SuperiorDesignatedInfo
	} else if role == PortRoleDesignatedPort &&
		p.MsgPriority == p.DesignatedPriority &&
		p.MsgTimes == p.DesignatedTimes {
		return RepeatedDesignatedInfo
	} else if role == PortRoleDesignatedPort &&
		IsMsgPriorityVectorWorseThanPortPriorityVector(&p.MsgPriority, &p.DesignatedPriority) {
		return InferiorDesignatedInfo
	} else if (role == PortRoleRootPort ||
		role == PortRoleAlternatePort ||
		role == PortRoleBackupPort) &&
		IsMsgPriorityVectorTheSameOrWorseThanPortPriorityVector(&p.MsgPriority, &p.DesignatedPriority) {
		return InferiorRootAlternateInfo
	} else {
		return OtherInfo
	}
}

func (pim *PimMachine) getRcvdMsgFlags(bpduLayer interface{}) uint8 {
	//bpdumsg := data.(RxBpduPdu)
	//packet := bpdumsg.pdu.(gopacket.Packet)
	//bpduLayer := packet.Layer(layers.LayerTypeBPDU)

	var flags uint8
	switch bpduLayer.(type) {
	case *layers.STP:
		StpMachineLogger("INFO", "PIM", "Found STP frame getting flags")
		stp := bpduLayer.(*layers.STP)
		flags = stp.Flags
	case *layers.RSTP:
		StpMachineLogger("INFO", "PIM", "Found RSTP frame getting flags")
		rstp := bpduLayer.(*layers.RSTP)
		flags = rstp.Flags
	case *layers.PVST:

		StpMachineLogger("INFO", "PIM", "Found PVST frame getting flags")
		pvst := bpduLayer.(*layers.STP)
		flags = pvst.Flags
	default:
		StpMachineLogger("ERROR", "PIM", fmt.Sprintf("%T\n", bpduLayer))
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
// TODO add logic
func (pim *PimMachine) betterorsameinfo(newInfoIs PortInfoState) bool {
	p := pim.p

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
func (pim *PimMachine) recordPriority(rcvdMsgPriority *PriorityVector) {
	p := pim.p
	p.MsgPriority = *rcvdMsgPriority
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
	if StpGetBpduLearning(rcvdMsgFlags) {
		p.Agreed = true
		p.Proposing = false
	}
}

func (pim *PimMachine) recordAgreement(rcvdMsgFlags uint8) {
	p := pim.p
	if p.RstpVersion &&
		p.OperPointToPointMAC &&
		StpGetBpduAgreement(rcvdMsgFlags) {
		p.Agreed = true
		p.Proposing = false
	} else {
		p.Agreed = false
	}

}
