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

// TcmMachineInactive
func (tcm *TcMachine) TcMachineInactive(m fsm.Machine, data interface{}) fsm.State {
	p := tcm.p
	tcm.NotifyFdbFlush()
	p.TcWhileTimer.count = 0
	p.TcAck = false
	return TcStateInactive
}

// TcMachineLearning
func (tcm *TcMachine) TcMachineLearning(m fsm.Machine, data interface{}) fsm.State {
	p := tcm.p

	p.RcvdTc = false
	p.RcvdTcn = false
	p.RcvdTcAck = false
	p.TcProp = false
	return TcStateLearning
}

// TcMachineDetected
func (tcm *TcMachine) TcMachineDetected(m fsm.Machine, data interface{}) fsm.State {

	newinfonotificationsent := tcm.newTcWhile()
	tcm.setTcPropTree()
	if !newinfonotificationsent {
		tcm.NotifyNewInfoChanged(true)
	}
	return TcStateDetected
}

// TcMachineActive
func (tcm *TcMachine) TcMachineActive(m fsm.Machine, data interface{}) fsm.State {

	return TcStateActive
}

// TcMachineNotifyTcn
func (tcm *TcMachine) TcMachineNotifiedTcn(m fsm.Machine, data interface{}) fsm.State {

	tcm.newTcWhile()

	return TcStateNotifiedTcn
}

// TcMachineNotifyTc
func (tcm *TcMachine) TcMachineNotifiedTc(m fsm.Machine, data interface{}) fsm.State {
	p := tcm.p

	p.RcvdTcn = false
	p.RcvdTc = false
	if p.Role == PortRoleDesignatedPort {
		p.TcAck = true
	}
	// Figure 17-25 says setTcPropBridge, but this is the only mention of this in
	// the standard, assuming it should be Tree
	tcm.setTcPropTree()

	return TcStateNotifiedTc
}

// TcMachinePropagating
func (tcm *TcMachine) TcMachinePropagating(m fsm.Machine, data interface{}) fsm.State {
	p := tcm.p

	tcm.newTcWhile()
	tcm.NotifyFdbFlush()
	p.TcProp = false

	return TcStatePropagating
}

// TcMachineAcknowledged
func (tcm *TcMachine) TcMachineAcknowledged(m fsm.Machine, data interface{}) fsm.State {
	p := tcm.p

	tcm.NotifyTcWhileChanged(0)
	p.RcvdTcAck = false

	return TcStateAcknowledged
}

func TcMachineFSMBuild(p *StpPort) *TcMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	tcm := NewStpTcMachine(p)

	/*
	   TcStateInactive
	   	TcStateLearning
	   	TcStateDetected
	   	TcStateActive
	   	TcStateNotifiedTcn
	   	TcStateNotifiedTc
	   	TcStatePropagating
	   	TcStateAcknowledged
	*/
	// BEGIN -> INACTIVE
	rules.AddRule(TcStateNone, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateInactive, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateLearning, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateDetected, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateActive, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateNotifiedTcn, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateNotifiedTc, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStatePropagating, TcEventBegin, tcm.TcMachineInactive)
	rules.AddRule(TcStateAcknowledged, TcEventBegin, tcm.TcMachineInactive)

	// ROLE != ROOTPORT && ROLE != DESIGNATEDPORT and !(LEARN || LEARNING) and !(RCVDTC || RCVDTCN || RCVDTCACK || TCPROP) -> INACTIVE
	rules.AddRule(TcStateLearning, TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPortAndNotLearnAndNotLearningAndNotRcvdTcAndNotRcvdTcnAndNotRcvdTcAckAndNotTcProp, tcm.TcMachineInactive)

	// LEARN and NOTFLUSH -> LEARNING
	rules.AddRule(TcStateInactive, TcEventLearnAndNotFdbFlush, tcm.TcMachineLearning)

	// RCVDTC -> LEARNING or NOTIFIED_TC
	rules.AddRule(TcStateLearning, TcEventRcvdTc, tcm.TcMachineLearning)
	rules.AddRule(TcStateActive, TcEventRcvdTc, tcm.TcMachineNotifiedTc)

	// RCVDTCN -> LEARNING or NOTIFIED_TCN
	rules.AddRule(TcStateLearning, TcEventRcvdTcn, tcm.TcMachineLearning)
	rules.AddRule(TcStateActive, TcEventRcvdTcn, tcm.TcMachineNotifiedTcn)

	// RCVDTCACK -> LEARNING or ACKNOWLEDGED
	rules.AddRule(TcStateLearning, TcEventRcvdTcAck, tcm.TcMachineLearning)
	rules.AddRule(TcStateActive, TcEventRcvdTcAck, tcm.TcMachineAcknowledged)

	// TCPROP -> LEARNING
	rules.AddRule(TcStateLearning, TcEventTcProp, tcm.TcMachineLearning)

	// (ROLE == ROOTPORT or ROLE == DESIGNATEDPORT) and FORWARD and NOT OPEREDGE -> DETECTED
	rules.AddRule(TcStateLearning, TcEventRoleEqualRootPortAndForwardAndNotOperEdge, tcm.TcMachineDetected)
	rules.AddRule(TcStateLearning, TcEventRoleEqualDesignatedPortAndForwardAndNotOperEdge, tcm.TcMachineDetected)

	// UNCONDITIONAL FALL THROUGH -> ACTIVE
	rules.AddRule(TcStateDetected, TcEventUnconditionalFallThrough, tcm.TcMachineActive)
	rules.AddRule(TcStateNotifiedTcn, TcEventUnconditionalFallThrough, tcm.TcMachineActive)
	rules.AddRule(TcStateNotifiedTc, TcEventUnconditionalFallThrough, tcm.TcMachineActive)
	rules.AddRule(TcStatePropagating, TcEventUnconditionalFallThrough, tcm.TcMachineActive)
	rules.AddRule(TcStateAcknowledged, TcEventUnconditionalFallThrough, tcm.TcMachineActive)

	// ROLE != ROOT PORT and ROLE != DESIGNATEDPORT -> LEARNING
	rules.AddRule(TcStateActive, TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPort, tcm.TcMachineLearning)

	// OPEREDGE -> LEARNING
	rules.AddRule(TcStateActive, TcEventOperEdge, tcm.TcMachineLearning)

	// TCPROP and NOT OPEREDGE -> PROPAGATING
	rules.AddRule(TcStateActive, TcEventTcPropAndNotOperEdge, tcm.TcMachinePropagating)

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
		StpMachineLogger("INFO", "TCM", "Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.TcKillSignalEvent:
				StpMachineLogger("INFO", "TCM", "Machine End")
				return

			case event := <-m.TcEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
				} else {
					// for faster transitions lets check all state events
					m.ProcessPostStateProcessing()
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

func (tcm *TcMachine) ProcessPostStateInactive() {
	p := tcm.p
	if tcm.Machine.Curr.CurrentState() == TcStateInactive &&
		p.Learn &&
		!p.FdbFlush {
		rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventLearnAndNotFdbFlush, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
		}

	}
}

func (tcm *TcMachine) ProcessPostStateLearning() {
	p := tcm.p
	if tcm.Machine.Curr.CurrentState() == TcStateLearning &&
		(p.Role == PortRoleRootPort || p.Role == PortRoleDesignatedPort) &&
		p.Forward &&
		!p.OperEdge {
		if p.Role == PortRoleRootPort {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventRoleEqualRootPortAndForwardAndNotOperEdge, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		} else {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventRoleEqualDesignatedPortAndForwardAndNotOperEdge, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		}
	}
}

func (tcm *TcMachine) ProcessPostStateActive() {
	p := tcm.p
	if tcm.Machine.Curr.CurrentState() == TcStateActive {
		if p.Role != PortRoleRootPort &&
			p.Role != PortRoleDesignatedPort {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPort, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		} else if p.OperEdge {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventOperEdge, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		} else if p.RcvdTcn {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventRcvdTcn, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		} else if p.RcvdTc {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventRcvdTc, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		} else if p.TcProp && !p.OperEdge {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventTcPropAndNotOperEdge, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		} else if p.RcvdTcAck {
			rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventRcvdTcAck, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
			}
		}
	}
}

// ProcessPostStateProcessing:
// advance states after a given event for faster state transitions
func (tcm *TcMachine) ProcessPostStateProcessing() {

	tcm.ProcessPostStateInactive()
	tcm.ProcessPostStateLearning()
	if tcm.Machine.Curr.CurrentState() == TcStateDetected {
		rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
		}
	}
	tcm.ProcessPostStateActive()

	if tcm.Machine.Curr.CurrentState() == TcStateNotifiedTcn ||
		tcm.Machine.Curr.CurrentState() == TcStateNotifiedTc ||
		tcm.Machine.Curr.CurrentState() == TcStatePropagating ||
		tcm.Machine.Curr.CurrentState() == TcStateAcknowledged {
		rv := tcm.Machine.ProcessEvent(TcMachineModuleStr, TcEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "TCM", fmt.Sprintf("%s\n", rv))
		}
	}

}

func (tcm *TcMachine) NotifyFdbFlush() {
	p := tcm.p
	p.FdbFlush = true
	// TODO send flush to asic d to flush macs
	// spawn go routine to flush and wait for response
	// but allow processing to continue
}

func (tcm *TcMachine) NotifyTcWhileChanged(val int32) {
	p := tcm.p

	p.TcWhileTimer.count = val
	if val == 0 {
		// this should stop transmit of tcn messages
	}
}

func (tcm *TcMachine) NotifyNewInfoChanged(newinfo bool) {
	p := tcm.p
	if p.NewInfo != newinfo {
		p.NewInfo = newinfo

		if p.PtxmMachineFsm.Machine.Curr.CurrentState() == PtxmStateIdle {
			if p.SendRSTP &&
				p.NewInfo &&
				p.TxCount < TransmitHoldCountDefault &&
				p.HelloWhenTimer.count != 0 {
				p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
					e:   PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
					src: TcMachineModuleStr,
				}
			} else if !p.SendRSTP &&
				p.NewInfo && p.Role == PortRoleRootPort &&
				p.TxCount < TransmitHoldCountDefault &&
				p.HelloWhenTimer.count != 0 {
				p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
					e:   PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
					src: TcMachineModuleStr,
				}
			} else if !p.SendRSTP &&
				p.NewInfo && p.Role == PortRoleDesignatedPort &&
				p.TxCount < TransmitHoldCountDefault &&
				p.HelloWhenTimer.count != 0 {
				p.PtxmMachineFsm.PtxmEvents <- MachineEvent{
					e:   PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo,
					src: TcMachineModuleStr,
				}
			}
		}
	}

}

// newTcWhile: 17.21.7
func (tcm *TcMachine) newTcWhile() (newinfonotificationsent bool) {
	p := tcm.p
	newinfonotificationsent = false
	if p.TcWhileTimer.count == 0 {
		if p.SendRSTP {
			p.TcWhileTimer.count = BridgeHelloTimeDefault + 2
			tcm.NotifyNewInfoChanged(true)
			newinfonotificationsent = true
		} else {
			p.TcWhileTimer.count = int32(p.b.RootTimes.MaxAge + p.b.RootTimes.ForwardingDelay)
		}
	}
	return newinfonotificationsent
}

// setTcPropTree: 17.21.18
func (tcm *TcMachine) setTcPropTree() {
	p := tcm.p
	b := p.b

	var port *StpPort
	for _, pId := range b.StpPorts {
		if pId != p.IfIndex &&
			StpFindPortById(pId, &port) {
			port.TcProp = true
		}
	}
}
