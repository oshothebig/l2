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

func (m *PrxmMachine) GetCurrStateStr() string {
	return PrxmStateStrMap[m.Machine.Curr.CurrentState()]
}

func (m *PrxmMachine) GetPrevStateStr() string {
	return PrxmStateStrMap[m.Machine.Curr.PreviousState()]
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPrxmMachine(p *StpPort) *PrxmMachine {
	prxm := &PrxmMachine{
		p:                   p,
		PrxmEvents:          make(chan MachineEvent, 50),
		PrxmRxBpduPkt:       make(chan RxBpduPdu, 50),
		PrxmKillSignalEvent: make(chan bool),
		PrxmLogEnableEvent:  make(chan bool)}

	p.PrxmMachineFsm = prxm

	return prxm
}

func (prxm *PrxmMachine) PrxmLogger(s string) {
	StpMachineLogger("INFO", "PRXM", prxm.p.IfIndex, prxm.p.BrgIfIndex, s)
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
		logEna:      true,
		logger:      prxm.PrxmLogger,
		owner:       PrxmMachineModuleStr,
		ps:          PrxmStateNone,
		s:           PrxmStateNone,
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
	defer p.NotifyRcvdMsgChanged(PrxmMachineModuleStr, p.RcvdMsg, false, data)
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
	defer p.NotifyRcvdMsgChanged(PrxmMachineModuleStr, p.RcvdMsg, rcvdMsg, data)
	p.RcvdMsg = rcvdMsg

	/* do not transition to NOT OperEdge if AdminEdge is set */
	if !p.AdminEdge && p.AutoEdgePort {
		defer p.NotifyOperEdgeChanged(PrxmMachineModuleStr, p.OperEdge, false)
		p.OperEdge = false
	}
	//p.OperEdge = false
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
	rules.AddRule(PrxmStateReceive, PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg, prxm.PrxmMachineReceive)

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
		StpMachineLogger("INFO", "PRXM", p.IfIndex, p.BrgIfIndex, "Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PrxmKillSignalEvent:
				StpMachineLogger("INFO", "PRXM", p.IfIndex, p.BrgIfIndex, "Machine End")
				return

			case event := <-m.PrxmEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s state[%s]event[%d]\n", rv, PrxmStateStrMap[m.Machine.Curr.CurrentState()], event.e))
				} else {
					// for faster state transitions
					m.ProcessPostStateProcessing(event.data)
				}

				if event.responseChan != nil {
					SendResponse(PrxmMachineModuleStr, event.responseChan)
				}
			case rx := <-m.PrxmRxBpduPkt:
				//fmt.Println("Event PKT Rx", rx.src, rx.ptype, p.PortEnabled)
				if m.Machine.Curr.CurrentState() == PrxmStateDiscard {
					if p.PortEnabled {
						rv := m.Machine.ProcessEvent("RX MODULE", PrxmEventRcvdBpduAndPortEnabled, rx)
						if rv != nil {
							StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s state[%s]event[%d]\n", rv, PrxmStateStrMap[m.Machine.Curr.CurrentState()], PrxmEventRcvdBpduAndPortEnabled))
						} else {
							// for faster state transitions
							m.ProcessPostStateProcessing(rx)
						}
					} else {
						rv := m.Machine.ProcessEvent("RX MODULE", PrxmEventRcvdBpduAndNotPortEnabled, rx)
						if rv != nil {
							StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s state[%s]event[%d]\n", rv, PrxmStateStrMap[m.Machine.Curr.CurrentState()], PrxmEventRcvdBpduAndPortEnabled))
						} else {
							// for faster state transitions
							m.ProcessPostStateProcessing(rx)
						}
					}
				} else {
					if p.PortEnabled &&
						!p.RcvdMsg {
						rv := m.Machine.ProcessEvent("RX MODULE", PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg, rx)
						if rv != nil {
							StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s state[%s]event[%d]\n", rv, PrxmStateStrMap[m.Machine.Curr.CurrentState()], PrxmEventRcvdBpduAndPortEnabled))
						} else {
							// for faster state transitions
							m.ProcessPostStateProcessing(rx)
						}
					} else if !p.PortEnabled {
						rv := m.Machine.ProcessEvent("RX MODULE", PrxmEventRcvdBpduAndNotPortEnabled, rx)
						if rv != nil {
							StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s state[%s]event[%d]\n", rv, PrxmStateStrMap[m.Machine.Curr.CurrentState()], PrxmEventRcvdBpduAndPortEnabled))
						} else {
							// for faster state transitions
							m.ProcessPostStateProcessing(rx)
						}
					}

				}

				p.SetRxPortCounters(rx.ptype)
			case ena := <-m.PrxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(prxm)
}

func (prxm *PrxmMachine) ProcessPostStateDiscard(data interface{}) {
	p := prxm.p
	if prxm.Machine.Curr.CurrentState() == PrxmStateDiscard &&
		p.RcvdBPDU &&
		p.PortEnabled {
		rv := prxm.Machine.ProcessEvent(PrxmMachineModuleStr, PrxmEventRcvdBpduAndPortEnabled, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s\n", rv))
		} else {
			prxm.ProcessPostStateProcessing(data)
		}
	}
}

func (prxm *PrxmMachine) ProcessPostStateReceive(data interface{}) {
	p := prxm.p
	if prxm.Machine.Curr.CurrentState() == PrxmStateReceive &&
		p.RcvdBPDU &&
		p.PortEnabled &&
		!p.RcvdMsg {
		rv := prxm.Machine.ProcessEvent(PrxmMachineModuleStr, PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg, data)
		if rv != nil {
			StpMachineLogger("ERROR", "PRXM", p.IfIndex, p.BrgIfIndex, fmt.Sprintf("%s\n", rv))
		} else {
			prxm.ProcessPostStateProcessing(data)
		}
	}
}

func (prxm *PrxmMachine) ProcessPostStateProcessing(data interface{}) {
	// post processing
	prxm.ProcessPostStateDiscard(data)
	prxm.ProcessPostStateReceive(data)
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

		//StpMachineLogger("INFO", "PRXM", p.IfIndex, "Received RSTP packet")

		defer prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(p.RcvdTc, p.RcvdTcn, p.RcvdTcAck, StpGetBpduTopoChange(flags), false, false)
		p.RcvdTc = StpGetBpduTopoChange(flags)
		p.RcvdTcn = false
		p.RcvdTcAck = false
	case *layers.PVST:
		// 17.21.22
		// some checks a bit redundant as the layers class has already validated
		// the BPDUType, but for completness going to add the check anyways
		pvst := bpduLayer.(*layers.PVST)
		flags = pvst.Flags
		if pvst.ProtocolVersionId == layers.RSTPProtocolVersion &&
			pvst.BPDUType == layers.BPDUTypeRSTP {
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

		//StpMachineLogger("INFO", "PRXM", p.IfIndex, "Received RSTP packet")

		defer prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(p.RcvdTc, p.RcvdTcn, p.RcvdTcAck, StpGetBpduTopoChange(flags), false, StpGetBpduTopoChangeAck(flags))
		p.RcvdTc = StpGetBpduTopoChange(flags)
		p.RcvdTcn = false
		p.RcvdTcAck = StpGetBpduTopoChangeAck(flags)
	case *layers.STP:
		stp := bpduLayer.(*layers.STP)
		flags = stp.Flags
		if stp.ProtocolVersionId == layers.STPProtocolVersion &&
			stp.BPDUType == layers.BPDUTypeSTP {

			// Found that Cisco send dot1d frame for tc going to
			// still interpret this as RSTP frame
			if StpGetBpduTopoChange(stp.Flags) ||
				StpGetBpduTopoChangeAck(stp.Flags) {

				p.RcvdRSTP = true
			} else {
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
			}

			validPdu = true
		}

		//StpMachineLogger("INFO", "PRXM", p.IfIndex, "Received STP packet")
		defer prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(p.RcvdTc, p.RcvdTcn, p.RcvdTcAck, StpGetBpduTopoChange(flags), false, StpGetBpduTopoChangeAck(flags))
		p.RcvdTc = StpGetBpduTopoChange(flags)
		p.RcvdTcn = false
		p.RcvdTcAck = StpGetBpduTopoChangeAck(flags)
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
			//StpMachineLogger("INFO", "PRXM", p.IfIndex, "Received TCN packet")
			defer prxm.NotifyRcvdTcRcvdTcnRcvdTcAck(p.RcvdTc, p.RcvdTcn, p.RcvdTcAck, false, true, false)
			p.RcvdTc = false
			p.RcvdTcn = true
			p.RcvdTcAck = false

		}
	}

	return validPdu
}

func (prxm *PrxmMachine) NotifyRcvdTcRcvdTcnRcvdTcAck(oldrcvdtc bool, oldrcvdtcn bool, oldrcvdtcack bool, newrcvdtc bool, newrcvdtcn bool, newrcvdtcack bool) {

	p := prxm.p

	// only care if there was a change
	if oldrcvdtc != newrcvdtc ||
		oldrcvdtcn != newrcvdtcn ||
		oldrcvdtcack != newrcvdtcack {

		if oldrcvdtc != newrcvdtc &&
			p.RcvdTc &&
			(p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning ||
				p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateActive) {
			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRcvdTc,
				src: RxModuleStr,
			}
		}
		if oldrcvdtcn != newrcvdtcn &&
			p.RcvdTcn &&
			(p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning ||
				p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateActive) {

			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRcvdTcn,
				src: RxModuleStr,
			}
		}
		if oldrcvdtcack != newrcvdtcack &&
			p.RcvdTcAck &&
			(p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning ||
				p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateActive) {

			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRcvdTcAck,
				src: RxModuleStr,
			}
		}
		if p.TcMachineFsm.Machine.Curr.CurrentState() == TcStateLearning &&
			p.Role != PortRoleRootPort &&
			p.Role != PortRoleDesignatedPort &&
			!(p.Learn || p.Learning) &&
			!(p.RcvdTc || p.RcvdTcn || p.RcvdTcAck || p.TcProp) {

			p.TcMachineFsm.TcEvents <- MachineEvent{
				e:   TcEventRoleNotEqualRootPortAndRoleNotEqualDesignatedPortAndNotLearnAndNotLearningAndNotRcvdTcAndNotRcvdTcnAndNotRcvdTcAckAndNotTcProp,
				src: RxModuleStr,
			}
		}
	}
}
