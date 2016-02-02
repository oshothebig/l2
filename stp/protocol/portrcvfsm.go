// 17.23 Port Receive state machine
package stp

import (
	"fmt"
	//"time"
	"github.com/google/gopacket/layers"
	"utils/fsm"
)

const PrxmMachineModuleStr = "Port Receive State Machine"

const (
	PrxmStateNone = iota + 1
	PrxmStateDiscard
	PrxmStateReceive
)

var PrxmStateStrMap map[fsm.State]string

func PrxmMachineStrStateMapInit() {
	PrxmStateStrMap = make(map[fsm.State]string)
	PrxmStateStrMap[PrxmStateNone] = "None"
	PrxmStateStrMap[PrxmStateDiscard] = "Discard"
	PrxmStateStrMap[PrxmStateReceive] = "Receive"
}

const (
	PrxmEventBegin = iota + 1
	PrxmEventRcvdBpduAndNotPortEnabled
	PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled
	PrxmEventRcvdBpduAndPortEnabled
	PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg
)

type RxBpduPdu struct {
	pdu          interface{}
	ptype        BPDURxType
	src          string
	responseChan chan string
}

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type PrxmMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PrxmEvents chan MachineEvent
	// rx pkt
	PrxmRxBpduPkt chan RxBpduPdu

	// stop go routine
	PrxmKillSignalEvent chan bool
	// enable logging
	PrxmLogEnableEvent chan bool
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPrxmMachine(p *StpPort) *PrxmMachine {
	prxm := &PrxmMachine{
		p:                   p,
		PrxmEvents:          make(chan MachineEvent, 10),
		PrxmRxBpduPkt:       make(chan RxBpduPdu, 20),
		PrxmKillSignalEvent: make(chan bool),
		PrxmLogEnableEvent:  make(chan bool)}

	p.PrxmMachineFsm = prxm

	return prxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (prxm *PrxmMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if prxm.Machine == nil {
		prxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	prxm.Machine.Rules = r
	prxm.Machine.Curr = &StpStateEvent{
		strStateMap: PrxmStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PrxmMachineModuleStr,
		ps:     PrxmStateNone,
		s:      PrxmStateNone,
	}

	return prxm.Machine
}

// Stop should clean up all resources
func (prxm *PrxmMachine) Stop() {

	// stop the go routine
	prxm.PrxmKillSignalEvent <- true

	close(prxm.PrxmEvents)
	close(prxm.PrxmRxBpduPkt)
	close(prxm.PrxmLogEnableEvent)
	close(prxm.PrxmKillSignalEvent)

}

// PrmMachineDiscard
func (prxm *PrxmMachine) PrxmMachineDiscard(m fsm.Machine, data interface{}) fsm.State {
	p := prxm.p
	p.RcvdBPDU = false
	p.RcvdRSTP = false
	p.RcvdSTP = false
	p.RcvdMsg = false
	// set to RSTP performance paramters Migrate Time
	p.EdgeDelayWhileTimer.count = MigrateTimeDefault
	return PrxmStateDiscard
}

// LacpPtxMachineFastPeriodic sets the periodic transmission time to fast
// and starts the timer
func (prxm *PrxmMachine) PrxmMachineReceive(m fsm.Machine, data interface{}) fsm.State {

	p := prxm.p

	// save off the bpdu info
	p.SaveMsgRcvInfo(data)

	//17.23
	// Decoding has been done as part of the Rx logic was a means of filtering
	rcvdMsg := prxm.UpdtBPDUVersion(data)
	// Figure 17-12
	prxm.NotifyRcvdMsgChange(data, rcvdMsg)
	prxm.NotifyOperEdgeChange(false)
	p.RcvdBPDU = false
	p.EdgeDelayWhileTimer.count = MigrateTimeDefault

	return PrxmStateReceive
}

func PrxmMachineFSMBuild(p *StpPort) *PrxmMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	prxm := NewStpPrxmMachine(p)

	//BEGIN -> DISCARD
	rules.AddRule(PrxmStateNone, PrxmEventBegin, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateDiscard, PrxmEventBegin, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateReceive, PrxmEventBegin, prxm.PrxmMachineDiscard)

	// RX BPDU && PORT NOT ENABLED	 -> DISCARD
	rules.AddRule(PrxmStateDiscard, PrxmEventRcvdBpduAndNotPortEnabled, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateReceive, PrxmEventRcvdBpduAndNotPortEnabled, prxm.PrxmMachineDiscard)

	// EDGEDELAYWHILE != MIGRATETIME && PORT NOT ENABLED -> DISCARD
	rules.AddRule(PrxmStateDiscard, PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled, prxm.PrxmMachineDiscard)
	rules.AddRule(PrxmStateReceive, PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled, prxm.PrxmMachineDiscard)

	// RX BPDU && PORT ENABLED -> RECEIVE
	rules.AddRule(PrxmStateDiscard, PrxmEventRcvdBpduAndPortEnabled, prxm.PrxmMachineReceive)

	// RX BPDU && PORT ENABLED && NOT RCVDMSG
	rules.AddRule(PrxmStateDiscard, PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg, prxm.PrxmMachineReceive)

	// Create a new FSM and apply the rules
	prxm.Apply(&rules)

	return prxm
}

// PrxmMachineMain:
func (p *StpPort) PrxmMachineMain() {

	// Build the State machine for STP Receive Machine according to
	// 802.1d Section 17.23
	prxm := PrxmMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	prxm.Machine.Start(prxm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PrxmMachine) {
		StpLogger("INFO", "PRXM: Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PrxmKillSignalEvent:
				StpLogger("INFO", "PRXM: Machine End")
				return

			case event := <-m.PrxmEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpLogger("INFO", fmt.Sprintf("%s\n", rv))
				}

				// post processing
				if m.Machine.Curr.CurrentState() == PrxmStateReceive &&
					m.p.RcvdMsg {
					rv := m.Machine.ProcessEvent(PrxmMachineModuleStr, PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg, nil)
					if rv != nil {
						StpLogger("INFO", fmt.Sprintf("%s\n", rv))
					}
				}

				if event.responseChan != nil {
					SendResponse(PrxmMachineModuleStr, event.responseChan)
				}
			case rx := <-m.PrxmRxBpduPkt:
				p.BpduRx++
				//fmt.Println("Event PKT Rx", rx.src, PrxmEventRcvdBpduAndPortEnabled)
				if p.PortEnabled {
					m.Machine.ProcessEvent("RX MODULE", PrxmEventRcvdBpduAndPortEnabled, rx)
				} else {

					m.Machine.ProcessEvent("RX MODULE", PrxmEventRcvdBpduAndNotPortEnabled, rx)
				}

			case ena := <-m.PrxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(prxm)
}

// UpdtBPDUVersion:  17.21.22
// This function will also inform Port Migration of
// BPDU type rcvdRSTP or rcvdSTP Figure 17-12
func (prxm *PrxmMachine) UpdtBPDUVersion(data interface{}) bool {
	validPdu := false
	p := prxm.p
	bpdumsg := data.(RxBpduPdu)
	bpduLayer := bpdumsg.pdu
	flags := uint8(0)
	switch bpduLayer.(type) {
	case *layers.RSTP:
		// 17.21.22
		// some checks a bit redundant as the layers class has already validated
		// the BPDUType, but for completness going to add the check anyways
		rstp := bpduLayer.(*layers.RSTP)
		flags = rstp.Flags
		if rstp.ProtocolVersionId == layers.RSTPProtocolVersion &&
			rstp.BPDUType == layers.BPDUTypeRSTP {
			// Inform the Port Protocol Migration STate machine
			// that we have received a RSTP packet when we were previously
			// sending non-RSTP
			if !p.RcvdRSTP &&
				!p.SendRSTP &&
				p.BridgeProtocolVersionGet() == layers.RSTPProtocolVersion {
				if p.PpmmMachineFsm != nil {
					p.PpmmMachineFsm.PpmmEvents <- MachineEvent{
						e:    PpmmEventRstpVersionAndNotSendRSTPAndRcvdRSTP,
						data: bpduLayer,
						src:  PrxmMachineModuleStr}
				}
			}
			p.RcvdRSTP = true
			validPdu = true
		}

		StpLogger("INFO", "Received RSTP packet")

		prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(StpGetBpduTopoChange(flags), false, false)
	case *layers.STP:
		stp := bpduLayer.(*layers.STP)
		flags = stp.Flags
		if stp.ProtocolVersionId == layers.STPProtocolVersion &&
			stp.BPDUType == layers.BPDUTypeSTP {
			// Inform the Port Protocol Migration State Machine
			// that we have received an STP packet when we were previously
			// sending RSTP
			if p.SendRSTP {
				if p.PpmmMachineFsm != nil {
					p.PpmmMachineFsm.PpmmEvents <- MachineEvent{
						e:    PpmmEventSendRSTPAndRcvdSTP,
						data: bpduLayer,
						src:  PrxmMachineModuleStr}
				}
			}
			p.RcvdSTP = true
			validPdu = true
		}

		StpLogger("INFO", "Received STP packet")
		prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(false, false, StpGetBpduTopoChangeAck(flags))
	case *layers.BPDUTopology:
		topo := bpduLayer.(*layers.BPDUTopology)
		if (topo.ProtocolVersionId == layers.STPProtocolVersion &&
			topo.BPDUType == layers.BPDUTypeTopoChange) ||
			(topo.ProtocolVersionId == layers.TCNProtocolVersion &&
				topo.BPDUType == layers.BPDUTypeTopoChange) {
			// Inform the Port Protocol Migration State Machine
			// that we have received an STP packet when we were previously
			// sending RSTP
			if p.SendRSTP {
				if p.PpmmMachineFsm != nil {
					p.PpmmMachineFsm.PpmmEvents <- MachineEvent{
						e:    PpmmEventSendRSTPAndRcvdSTP,
						data: bpduLayer,
						src:  PrxmMachineModuleStr}
				}
			}
			p.RcvdSTP = true
			validPdu = true
			StpLogger("INFO", "Received TCN packet")
			prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(false, true, false)

		}
	}

	return validPdu
}

// NotifyRcvdMsgChange
// Port Receive -> Port Information Figure 17-12
func (prxm *PrxmMachine) NotifyRcvdMsgChange(data interface{}, rcvdMsg bool) {
	p := prxm.p
	if p.RcvdMsg != rcvdMsg {
		bpdumsg := data.(RxBpduPdu)
		bpduLayer := bpdumsg.pdu

		p.RcvdMsg = rcvdMsg
		if p.PimMachineFsm != nil {
			if rcvdMsg {
				if p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateDisabled {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:    PimEventRcvdMsg,
						data: bpduLayer,
						src:  PrxmMachineModuleStr}
				} else if p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent &&
					!p.UpdtInfo {
					p.PimMachineFsm.PimEvents <- MachineEvent{
						e:    PimEventRcvdMsgAndNotUpdtInfo,
						data: bpduLayer,
						src:  PrxmMachineModuleStr}
				}
			} else if !rcvdMsg &&
				p.InfoIs == PortInfoStateReceived &&
				p.RcvdInfoWhiletimer.count == 0 &&
				!p.UpdtInfo {
				p.PimMachineFsm.PimEvents <- MachineEvent{
					e:    PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
					data: bpduLayer,
					src:  PrxmMachineModuleStr}

			}
		}
	}
}

func (prxm *PrxmMachine) NotifyOperEdgeChange(operedge bool) {
	p := prxm.p
	if p.OperEdge != operedge {

		if p.BdmMachineFsm.Machine.Curr.CurrentState() == BdmStateEdge &&
			!p.OperEdge {
			p.BdmMachineFsm.BdmEvents <- MachineEvent{
				e:   BdmEventNotOperEdge,
				src: PrxmMachineModuleStr,
			}
		}
	}
}

func (prxm *PrxmMachine) NotifyRcvdTcRcvdTcnRcvdTcAck(rcvdtc bool, rcvdtcn bool, rcvdtcack bool) {

	p := prxm.p

	// only care if there was a change
	if p.RcvdTc != rcvdtc ||
		p.RcvdTcn != rcvdtcn ||
		p.RcvdTcAck != rcvdtcack {

		p.RcvdTc = rcvdtc
		p.RcvdTcn = rcvdtcn
		p.RcvdTcAck = rcvdtcack

		if rcvdtc &&
			(p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning ||
				p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateActive) {
			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRcvdTc,
				src: RxModuleStr,
			}
		} else if rcvdtcn &&
			(p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning ||
				p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateActive) {

			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRcvdTcn,
				src: RxModuleStr,
			}
		} else if rcvdtcack &&
			(p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning ||
				p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateActive) {

			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRcvdTcAck,
				src: RxModuleStr,
			}
		} else if p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning &&
			p.Role != PortRoleRootPort &&
			p.Role != PortRoleDesignatedPort &&
			!(p.Learn || p.Learning) &&
			!(rcvdtc || rcvdtcn || rcvdtcack || p.TcProp) {

			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPortAndNotLearnAndNotLearningAndNotRcvdTcAndNotRcvdTcnAndNotRcvdTcAckAndNotTcProp,
				src: RxModuleStr,
			}
		}
	}
}
