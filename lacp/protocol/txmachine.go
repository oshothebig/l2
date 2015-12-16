// TX MACHINE, this is not really a state machine but going to create a sort of
// state machine to processes events
// TX Machine is described in 802.1ax-2014 6.4.16
package lacp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"strconv"
	"strings"
	"time"
	"utils/fsm"
)

const TxMachineModuleStr = "Tx Machine"

const (
	LacpTxmStateNone = iota + 1
	LacpTxmStateOn
	LacpTxmStateOff
	LacpTxmStateDelayed
	LacpTxmStateGuardTimerExpire
)

var TxmStateStrMap map[fsm.State]string

func TxMachineStrStateMapCreate() {

	TxmStateStrMap = make(map[fsm.State]string)
	TxmStateStrMap[LacpTxmStateNone] = "LacpTxmStateNone"
	TxmStateStrMap[LacpTxmStateOn] = "LacpTxmStateOn"
	TxmStateStrMap[LacpTxmStateOff] = "LacpTxmStateOff"
	TxmStateStrMap[LacpTxmStateDelayed] = "LacpTxmStateDelayed"
	TxmStateStrMap[LacpTxmStateGuardTimerExpire] = "LacpTxmStateGuardTimerExpire"
}

const (
	LacpTxmEventBegin = iota + 1
	LacpTxmEventNtt
	LacpTxmEventGuardTimer
	LacpTxmEventDelayTx
	LacpTxmEventLacpDisabled
	LacpTxmEventLacpEnabled
)

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpTxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// Port this Machine is associated with
	p *LaAggPort

	// number of frames that should be transmitted
	// after restriction logic has cleared
	txPending int

	// number of frames transmitted within guard timer interval
	txPkts int

	// ntt, this may be set by external applications
	// the state machine will only clear
	ntt bool

	// debug log
	log chan string

	// timer needed for 802.1ax-20014 section 6.4.16
	txGuardTimer *time.Timer

	// machine specific events
	TxmEvents          chan LacpMachineEvent
	TxmKillSignalEvent chan bool
	TxmLogEnableEvent  chan bool
}

// PrevState will get the previous state from the state transitions
func (txm *LacpTxMachine) PrevState() fsm.State { return txm.PreviousState }

// PrevStateSet will set the previous state
func (txm *LacpTxMachine) PrevStateSet(s fsm.State) { txm.PreviousState = s }

// Stop will stop all timers and close all channels
func (txm *LacpTxMachine) Stop() {
	txm.TxGuardTimerStop()

	txm.TxmKillSignalEvent <- true

	close(txm.TxmEvents)
	close(txm.TxmKillSignalEvent)
	close(txm.TxmLogEnableEvent)
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpTxMachine(port *LaAggPort) *LacpTxMachine {
	txm := &LacpTxMachine{
		p:                  port,
		log:                port.LacpDebug.LacpLogChan,
		txPending:          0,
		txPkts:             0,
		ntt:                false,
		PreviousState:      LacpTxmStateNone,
		TxmEvents:          make(chan LacpMachineEvent, 10),
		TxmKillSignalEvent: make(chan bool),
		TxmLogEnableEvent:  make(chan bool)}

	port.TxMachineFsm = txm

	// start then stop
	txm.TxGuardTimerStart()
	txm.TxGuardTimerStop()

	return txm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine.
func (txm *LacpTxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if txm.Machine == nil {
		txm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	txm.Machine.Rules = r
	txm.Machine.Curr = &LacpStateEvent{
		strStateMap: TxmStateStrMap,
		logEna:      false,
		logger:      txm.LacpTxmLog,
		owner:       TxMachineModuleStr,
	}

	return txm.Machine
}

// LacpTxMachineOn will either send a packet out or delay transmission of a
// packet
func (txm *LacpTxMachine) LacpTxMachineOn(m fsm.Machine, data interface{}) fsm.State {

	var nextState fsm.State
	p := txm.p

	txm.PrevStateSet(txm.Machine.Curr.CurrentState())

	nextState = LacpTxmStateOn

	// NTT must be set to tx
	if txm.ntt == true {
		// if more than 3 packets are being transmitted within time interval
		// delay transmission
		if txm.txPkts < 3 {
			if txm.txPkts == 0 {
				txm.TxGuardTimerStart()
			}
			txm.txPkts++

			lacp := &layers.LACP{
				Version: layers.LACPVersion1,
				Actor: layers.LACPInfoTlv{TlvType: layers.LACPTLVActorInfo,
					Length: layers.LACPActorTlvLength,
					Info: layers.LACPPortInfo{
						System: layers.LACPSystem{SystemId: p.actorOper.system.actor_system,
							SystemPriority: p.actorOper.system.actor_system_priority,
						},
						Key:     p.actorOper.key,
						PortPri: p.actorOper.port_pri,
						Port:    p.actorOper.port,
						State:   p.actorOper.state,
					},
				},
				Partner: layers.LACPInfoTlv{TlvType: layers.LACPTLVPartnerInfo,
					Length: layers.LACPActorTlvLength,
					Info: layers.LACPPortInfo{
						System: layers.LACPSystem{SystemId: p.partnerOper.system.actor_system,
							SystemPriority: p.partnerOper.system.actor_system_priority,
						},
						Key:     p.partnerOper.key,
						PortPri: p.partnerOper.port_pri,
						Port:    p.partnerOper.port,
						State:   p.partnerOper.state,
					},
				},
				Collector: layers.LACPCollectorInfoTlv{
					TlvType:  layers.LACPTLVCollectorInfo,
					Length:   layers.LACPCollectorTlvLength,
					MaxDelay: 0,
				},
			}

			// transmit the packet
			for _, ftx := range LaSysGlobalTxCallbackListGet(p) {
				//txm.LacpTxmLog(fmt.Sprintf("Sending Tx packet %d", txm.txPkts))
				ftx(p.portNum, lacp)
			}
			// Version 2 consideration if enable_long_pdu_xmit and
			// LongLACPPDUTransmit are True:
			// LACPDU will be a Long LACPDU formatted by 802.1ax-2014 Section
			// 6.4.2 and including Port Conversation Mask TLV 6.4.2.4.3
			txm.ntt = false
		} else {
			txm.txPending++
			txm.LacpTxmLog(fmt.Sprintf("ON: Delay packets %d", txm.txPending))
			nextState = LacpTxmStateDelayed
		}
	}
	return nextState
}

// LacpTxMachineDelayed is a state in which a packet is forced to transmit
// regardless of the ntt state
func (txm *LacpTxMachine) LacpTxMachineDelayed(m fsm.Machine, data interface{}) fsm.State {
	var state fsm.State

	txm.PrevStateSet(txm.Machine.Curr.CurrentState())

	state = LacpTxmStateOn

	// if more than 3 packets are being transmitted within time interval
	// TODO send packet to MUX
	// Version 2 consideration if enable_long_pdu_xmit and
	// LongLACPPDUTransmit are True:
	// LACPDU will be a Long LACPDU formatted by 802.1ax-2014 Section
	// 6.4.2 and including Port Conversation Mask TLV 6.4.2.4.3
	txm.LacpTxmLog(fmt.Sprintf("Delayed: txPending %d txPkts %d delaying tx", txm.txPending, txm.txPkts))
	if txm.txPending > 0 && txm.txPkts > 3 {
		state = LacpTxmStateDelayed
		txm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventDelayTx,
			src: TxMachineModuleStr}
	} else {
		// transmit packet
		txm.txPending--
		txm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventNtt,
			src: TxMachineModuleStr}
	}

	return state
}

// LacpTxMachineOff will ensure that no packets are transmitted, typically means that
// lacp has been disabled
func (txm *LacpTxMachine) LacpTxMachineOff(m fsm.Machine, data interface{}) fsm.State {
	txm.txPending = 0
	txm.txPkts = 0
	txm.ntt = false
	txm.TxGuardTimerStop()
	return LacpTxmStateOff
}

// LacpTxMachineGuard will clear the current transmited packet count and
// generate a new event to tx a new packet
func (txm *LacpTxMachine) LacpTxMachineGuard(m fsm.Machine, data interface{}) fsm.State {
	txm.txPkts = 0
	var state fsm.State

	state = LacpTxmStateOn
	if txm.txPending > 0 {
		state = LacpTxmStateGuardTimerExpire
	}

	// no state transition just need to clear the txPkts
	return state
}

// LacpTxMachineFSMBuild will build the state machine with callbacks
func LacpTxMachineFSMBuild(p *LaAggPort) *LacpTxMachine {

	rules := fsm.Ruleset{}

	TxMachineStrStateMapCreate()

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	txm := NewLacpTxMachine(p)

	//BEGIN -> TX OFF
	rules.AddRule(LacpTxmStateNone, LacpTxmEventBegin, txm.LacpTxMachineOff)
	// NTT -> TX ON
	rules.AddRule(LacpTxmStateOn, LacpTxmEventNtt, txm.LacpTxMachineOn)
	rules.AddRule(LacpTxmStateGuardTimerExpire, LacpTxmEventNtt, txm.LacpTxMachineOn)
	// DELAY -> TX DELAY
	rules.AddRule(LacpTxmStateOn, LacpTxmEventNtt, txm.LacpTxMachineDelayed)
	rules.AddRule(LacpTxmStateDelayed, LacpTxmEventDelayTx, txm.LacpTxMachineDelayed)
	// LACP ON -> TX ON
	rules.AddRule(LacpTxmStateOff, LacpTxmEventLacpEnabled, txm.LacpTxMachineOn)
	// LACP DISABLED -> TX OFF
	rules.AddRule(LacpTxmStateNone, LacpTxmEventLacpDisabled, txm.LacpTxMachineOff)
	rules.AddRule(LacpTxmStateOn, LacpTxmEventLacpDisabled, txm.LacpTxMachineOff)
	rules.AddRule(LacpTxmStateDelayed, LacpTxmEventLacpDisabled, txm.LacpTxMachineOff)
	// GUARD TIMER -> TX ON
	rules.AddRule(LacpTxmStateOn, LacpTxmEventGuardTimer, txm.LacpTxMachineGuard)
	rules.AddRule(LacpTxmStateDelayed, LacpTxmEventGuardTimer, txm.LacpTxMachineGuard)

	// Create a new FSM and apply the rules
	txm.Apply(&rules)

	return txm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpTxMachineMain() {

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	txm := LacpTxMachineFSMBuild(p)

	// set the inital state
	txm.Machine.Start(txm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpTxMachine) {
		m.LacpTxmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.TxmKillSignalEvent:
				m.LacpTxmLog("Machine End")
				return

			case event := <-m.TxmEvents:

				//m.LacpTxmLog(fmt.Sprintf("Event rx %d %s %s", event.e, event.src, TxmStateStrMap[m.Machine.Curr.CurrentState()]))
				// special case, another machine has a need to
				// transmit a packet
				if event.e == LacpTxmEventNtt {
					m.ntt = true
				}

				rv := m.Machine.ProcessEvent(event.src, event.e, nil)

				if rv != nil {
					m.LacpTxmLog(strings.Join([]string{error.Error(rv), event.src, TxmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
				} else {
					if m.Machine.Curr.CurrentState() == LacpTxmStateGuardTimerExpire &&
						m.txPending > 0 && m.txPkts == 0 {
						m.txPending--
						m.ntt = true

						m.LacpTxmLog("Forcing NTT processing from expire")
						m.Machine.ProcessEvent(TxMachineModuleStr, LacpTxmEventNtt, nil)
					}
				}

				if event.responseChan != nil {
					SendResponse(TxMachineModuleStr, event.responseChan)
				}
			case ena := <-m.TxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(txm)
}

// LacpTxGuardGeneration will generate an event to the Tx Machine
// in order to clear the txPkts count
func (txm *LacpTxMachine) LacpTxGuardGeneration() {
	//txm.LacpTxmLog("LacpTxGuardGeneration")
	txm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventGuardTimer,
		src: TxMachineModuleStr}
}
