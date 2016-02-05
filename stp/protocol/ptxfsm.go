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
	"utils/fsm"
)

const PtxmMachineModuleStr = "Port Transmit State Machine"

const (
	PtxmStateNone = iota + 1
	PtxmStateTransmitInit
	PtxmStateTransmitConfig
	PtxmStateTransmitTCN
	PtxmStateTransmitPeriodic
	PtxmStateTransmitRSTP
	PtxmStateIdle
)

var PtxmStateStrMap map[fsm.State]string

func PtxmMachineStrStateMapInit() {
	PtxmStateStrMap = make(map[fsm.State]string)
	PtxmStateStrMap[PpmmStateNone] = "None"
	PtxmStateStrMap[PtxmStateTransmitInit] = "Transmit Init"
	PtxmStateStrMap[PtxmStateTransmitConfig] = "Transmit Config"
	PtxmStateStrMap[PtxmStateTransmitTCN] = "Transmit TCN"
	PtxmStateStrMap[PtxmStateTransmitPeriodic] = "Transmit Periodic"
	PtxmStateStrMap[PtxmStateTransmitRSTP] = "Transmit RSTP"
	PtxmStateStrMap[PtxmStateIdle] = "Idle"
}

const (
	PtxmEventBegin = iota + 1
	PtxmEventUnconditionalFallThrough
	PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo
	PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo
	PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo
	PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo
)

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type PtxmMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PtxmEvents chan MachineEvent
	// stop go routine
	PtxmKillSignalEvent chan bool
	// enable logging
	PtxmLogEnableEvent chan bool
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPtxmMachine(p *StpPort) *PtxmMachine {
	ptxm := &PtxmMachine{
		p:                   p,
		PtxmEvents:          make(chan MachineEvent, 10),
		PtxmKillSignalEvent: make(chan bool),
		PtxmLogEnableEvent:  make(chan bool)}

	p.PtxmMachineFsm = ptxm

	return ptxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (ptxm *PtxmMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if ptxm.Machine == nil {
		ptxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	ptxm.Machine.Rules = r
	ptxm.Machine.Curr = &StpStateEvent{
		strStateMap: PtxmStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PtxmMachineModuleStr,
		ps:     PtxmStateNone,
		s:      PtxmStateNone,
	}

	return ptxm.Machine
}

// Stop should clean up all resources
func (ptxm *PtxmMachine) Stop() {

	// stop the go routine
	ptxm.PtxmKillSignalEvent <- true

	close(ptxm.PtxmEvents)
	close(ptxm.PtxmLogEnableEvent)
	close(ptxm.PtxmKillSignalEvent)

}

// PtxmMachineTransmitInit
func (ptxm *PtxmMachine) PtxmMachineTransmitInit(m fsm.Machine, data interface{}) fsm.State {
	p := ptxm.p
	p.NewInfo = true
	p.TxCount = 0
	return PtxmStateTransmitInit
}

// PtxmMachineTransmitIdle
func (ptxm *PtxmMachine) PtxmMachineTransmitIdle(m fsm.Machine, data interface{}) fsm.State {
	p := ptxm.p
	p.HelloWhenTimer.count = int32(p.b.RootTimes.HelloTime)
	return PtxmStateIdle
}

// PtxmMachineTransmitPeriodic
func (ptxm *PtxmMachine) PtxmMachineTransmitPeriodic(m fsm.Machine, data interface{}) fsm.State {
	p := ptxm.p

	p.NewInfo = p.NewInfo || (p.Role == PortRoleDesignatedPort || (p.Role == PortRoleRootPort && p.TcWhileTimer.count != 0))

	return PtxmStateTransmitPeriodic
}

func (ptxm *PtxmMachine) PtxmMachineTransmitRSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ptxm.p

	p.NewInfo = false
	p.TxRSTP()
	p.TxCount++
	p.TcAck = false

	return PtxmStateTransmitRSTP
}

func (ptxm *PtxmMachine) PtxmMachineTransmitTCN(m fsm.Machine, data interface{}) fsm.State {
	p := ptxm.p

	p.NewInfo = false
	p.TxTCN()
	p.TxCount++

	return PtxmStateTransmitTCN
}

func (ptxm *PtxmMachine) PtxmMachineTransmitConfig(m fsm.Machine, data interface{}) fsm.State {
	p := ptxm.p

	p.NewInfo = false
	p.TxConfig()
	p.TxCount++
	p.TcAck = false

	return PtxmStateTransmitConfig
}

func PtxmMachineFSMBuild(p *StpPort) *PtxmMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	ptxm := NewStpPtxmMachine(p)

	//BEGIN -> TRANSMIT INIT
	rules.AddRule(PtxmStateNone, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)
	rules.AddRule(PtxmStateTransmitInit, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)
	rules.AddRule(PtxmStateTransmitConfig, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)
	rules.AddRule(PtxmStateTransmitTCN, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)
	rules.AddRule(PtxmStateTransmitPeriodic, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)
	rules.AddRule(PtxmStateTransmitRSTP, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)
	rules.AddRule(PtxmStateIdle, PtxmEventBegin, ptxm.PtxmMachineTransmitInit)

	// UNCONDITIONAL FALL THROUGH	 -> IDLE
	rules.AddRule(PtxmStateTransmitInit, PtxmEventUnconditionalFallThrough, ptxm.PtxmMachineTransmitIdle)
	rules.AddRule(PtxmStateTransmitConfig, PtxmEventUnconditionalFallThrough, ptxm.PtxmMachineTransmitIdle)
	rules.AddRule(PtxmStateTransmitTCN, PtxmEventUnconditionalFallThrough, ptxm.PtxmMachineTransmitIdle)
	rules.AddRule(PtxmStateTransmitPeriodic, PtxmEventUnconditionalFallThrough, ptxm.PtxmMachineTransmitIdle)
	rules.AddRule(PtxmStateTransmitRSTP, PtxmEventUnconditionalFallThrough, ptxm.PtxmMachineTransmitIdle)

	// HelloWhen == 0 -> TRANSMIT PERIODIC
	rules.AddRule(PtxmStateIdle, PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo, ptxm.PtxmMachineTransmitPeriodic)

	// sendRSTP && NewInfo && TxCount < TxHoldCound && HelloWhen != 0 && Selected && !UpdtInfo -> TRANSMIT RSTP
	rules.AddRule(PtxmStateIdle, PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo, ptxm.PtxmMachineTransmitRSTP)

	// sendRSTP && NewInfo && Root Port && TxCount < TxHoldCound && HelloWhen != 0 && Selected && !UpdtInfo -> TRANSMIT RSTP
	rules.AddRule(PtxmStateIdle, PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo, ptxm.PtxmMachineTransmitTCN)

	// sendRSTP && NewInfo && Designated Port && TxCount < TxHoldCound && HelloWhen != 0 && Selected && !UpdtInfo -> TRANSMIT RSTP
	rules.AddRule(PtxmStateIdle, PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo, ptxm.PtxmMachineTransmitConfig)

	// Create a new FSM and apply the rules
	ptxm.Apply(&rules)

	return ptxm
}

// PrxmMachineMain:
func (p *StpPort) PtxmMachineMain() {

	// Build the State machine for STP Receive Machine according to
	// 802.1d Section 17.23
	ptxm := PtxmMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	ptxm.Machine.Start(ptxm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PtxmMachine) {
		StpMachineLogger("INFO", "PTXM", "Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PtxmKillSignalEvent:
				StpMachineLogger("INFO", "PTXM", "Machine End")
				return

			case event := <-m.PtxmEvents:
				StpMachineLogger("INFO", "PTXM", fmt.Sprintf("Event Rx", event.src, event.e))
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
				} else {
					m.ProcessPostStateProcessing()
				}

				if event.responseChan != nil {
					SendResponse(PtxmMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PtxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(ptxm)
}

func (ptxm *PtxmMachine) ProcessPostStateTransmitInit() {

	if ptxm.Machine.Curr.CurrentState() == PtxmStateTransmitInit {
		rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
		} else {
			ptxm.ProcessPostStateProcessing()
		}
	}
}

func (ptxm *PtxmMachine) ProcessPostStateTransmitPeriodic() {

	if ptxm.Machine.Curr.CurrentState() == PtxmStateTransmitPeriodic {
		rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
		} else {
			ptxm.ProcessPostStateProcessing()
		}
	}
}

func (ptxm *PtxmMachine) ProcessPostStateTransmitConfig() {

	if ptxm.Machine.Curr.CurrentState() == PtxmStateTransmitConfig {
		rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
		} else {
			ptxm.ProcessPostStateProcessing()
		}
	}
}

func (ptxm *PtxmMachine) ProcessPostStateTransmitTcn() {

	if ptxm.Machine.Curr.CurrentState() == PtxmStateTransmitTCN {
		rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
		} else {
			ptxm.ProcessPostStateProcessing()
		}
	}
}

func (ptxm *PtxmMachine) ProcessPostStateTransmitRstp() {

	if ptxm.Machine.Curr.CurrentState() == PtxmStateTransmitRSTP {
		rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventUnconditionalFallThrough, nil)
		if rv != nil {
			StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
		} else {
			ptxm.ProcessPostStateProcessing()
		}
	}
}

func (ptxm *PtxmMachine) ProcessPostStateIdle() {
	p := ptxm.p
	if ptxm.Machine.Curr.CurrentState() == PtxmStateIdle {
		StpMachineLogger("INFO", "PTX", fmt.Sprintf("sendRSTP[%t] newInfo[%t] txCount[%d] hellwhen[%d] selected[%t] updtinfo[%t]\n",
			p.SendRSTP,
			p.NewInfo,
			p.TxCount,
			p.HelloWhenTimer.count,
			p.Selected,
			!p.UpdtInfo))
		if p.HelloWhenTimer.count == 0 &&
			p.Selected &&
			!p.UpdtInfo {
			rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
			} else {
				ptxm.ProcessPostStateProcessing()
			}
		} else if p.SendRSTP &&
			p.NewInfo &&
			p.TxCount < TransmitHoldCountDefault &&
			p.HelloWhenTimer.count != 0 &&
			p.Selected &&
			!p.UpdtInfo {
			rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
			} else {
				ptxm.ProcessPostStateProcessing()
			}
		} else if !p.SendRSTP &&
			p.NewInfo &&
			p.Role == PortRoleRootPort &&
			p.TxCount < TransmitHoldCountDefault &&
			p.HelloWhenTimer.count != 0 &&
			p.Selected &&
			!p.UpdtInfo {
			rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
			} else {
				ptxm.ProcessPostStateProcessing()
			}
		} else if !p.SendRSTP &&
			p.NewInfo &&
			p.Role == PortRoleDesignatedPort &&
			p.TxCount < TransmitHoldCountDefault &&
			p.HelloWhenTimer.count != 0 &&
			p.Selected &&
			!p.UpdtInfo {
			rv := ptxm.Machine.ProcessEvent(PtxmMachineModuleStr, PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo, nil)
			if rv != nil {
				StpMachineLogger("ERROR", "PTXM", fmt.Sprintf("%s\n", rv))
			} else {
				ptxm.ProcessPostStateProcessing()
			}
		}
	}
}

func (ptxm *PtxmMachine) ProcessPostStateProcessing() {

	ptxm.ProcessPostStateTransmitInit()
	ptxm.ProcessPostStateTransmitPeriodic()
	ptxm.ProcessPostStateTransmitConfig()
	ptxm.ProcessPostStateTransmitTcn()
	ptxm.ProcessPostStateTransmitRstp()
	ptxm.ProcessPostStateIdle()
}
