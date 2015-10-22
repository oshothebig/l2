// The Periodic Transmission Machine is described in the 802.1ax-2014 Section 6.4.13
package lacp

import (
	"time"
	"utils/fsm"
)

const (
	LacpPtxmStateNone = iota + 1
	LacpPtxmStateNoPeriodic
	LacpPtxmStateFastPeriodic
	LacpPtxmStateSlowPeriodic
	LacpPtxmStatePeriodicTx
)

const (
	LacpPtxmEventBegin = iota + 1
	LacpPtxmEventLacpDisabled
	LacpPtxmEventPortDisabled
	LacpPtxmEventActorPartnerOperActivityPassiveMode
	LacpPtxmEventUnconditionalFallthrough
	LacpPtxmEventPartnerOperStateTimeoutLong
	LacpPtxmEventPeriodicTimerExpired
	LacpPtxmEventPartnerOperStateTimeoutShort
)

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpPtxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// state transition log
	log    chan string
	logEna bool

	// Reference to LaAggPort
	p *LaAggPort

	// current tx interval LONG/SHORT
	PeriodicTxTimerInterval time.Duration

	// timer
	periodicTxTimer *time.Timer

	// machine specific events
	PtxmEvents chan fsm.Event
	// stop go routine
	PtxmKillSignalEvent chan bool
	// enable logging
	PtxmLogEnableEvent chan bool
}

func (ptxm *LacpPtxMachine) PrevState() fsm.State { return ptxm.PreviousState }

// PrevStateSet will set the previous state
func (ptxm *LacpPtxMachine) PrevStateSet(s fsm.State) { ptxm.PreviousState = s }

func (ptxm *LacpPtxMachine) Stop() {
	ptxm.PeriodicTimerStop()

	ptxm.PtxmKillSignalEvent <- true

	close(ptxm.PtxmEvents)
	close(ptxm.PtxmKillSignalEvent)
	close(ptxm.PtxmLogEnableEvent)
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpPtxMachine(port *LaAggPort) *LacpPtxMachine {
	ptxm := &LacpPtxMachine{
		p:                       port,
		log:                     port.LacpDebug.LacpLogChan,
		logEna:                  false,
		PreviousState:           LacpPtxmStateNone,
		PeriodicTxTimerInterval: LacpSlowPeriodicTime,
		PtxmEvents:              make(chan fsm.Event),
		PtxmKillSignalEvent:     make(chan bool),
		PtxmLogEnableEvent:      make(chan bool)}

	port.ptxMachineFsm = ptxm

	return ptxm
}

func (ptxm *LacpPtxMachine) LacpPtxLog(msg string) {
	if ptxm.logEna {
		ptxm.log <- msg
	}
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine.
func (ptxm *LacpPtxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if ptxm.Machine == nil {
		ptxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	ptxm.Machine.Rules = r

	return ptxm.Machine
}

// LacpPtxMachineNoPeriodic stops the periodic transmission of packets
func (ptxm *LacpPtxMachine) LacpPtxMachineNoPeriodic(m fsm.Machine, data interface{}) fsm.State {
	ptxm.LacpPtxLog("PTXM: No Periodic enter")
	ptxm.PeriodicTimerStop()
	return LacpPtxmStateNoPeriodic
}

// LacpPtxMachineFastPeriodic sets the periodic transmission time to fast
// and starts the timer
func (ptxm *LacpPtxMachine) LacpPtxMachineFastPeriodic(m fsm.Machine, data interface{}) fsm.State {
	ptxm.LacpPtxLog("PTXM: Fast Periodic Enter")
	ptxm.PeriodicTimerIntervalSet(LacpFastPeriodicTime)
	ptxm.PeriodicTimerStart()
	return LacpPtxmStateFastPeriodic
}

// LacpPtxMachineSlowPeriodic sets the periodic transmission time to slow
// and starts the timer
func (ptxm *LacpPtxMachine) LacpPtxMachineSlowPeriodic(m fsm.Machine, data interface{}) fsm.State {
	ptxm.LacpPtxLog("PTXM: Slow Periodic Enter")
	ptxm.PeriodicTimerIntervalSet(LacpSlowPeriodicTime)
	ptxm.PeriodicTimerStart()
	return LacpPtxmStateSlowPeriodic
}

// LacpPtxMachinePeriodicTx informs the tx machine that a packet should be transmitted by setting
// ntt = true
func (ptxm *LacpPtxMachine) LacpPtxMachinePeriodicTx(m fsm.Machine, data interface{}) fsm.State {
	ptxm.LacpPtxLog("PTXM: Periodic Tx Enter")
	// inform the tx machine that ntt should change to true which should transmit a
	// packet
	ptxm.p.txMachineFsm.TxmEvents <- LacpTxmEventNtt
	return LacpPtxmStatePeriodicTx
}

func LacpPtxMachineFSMBuild(p *LaAggPort) *LacpPtxMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new LacpPtxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the NO PERIODIC state
	ptxm := NewLacpPtxMachine(p)

	//BEGIN -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventBegin, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventBegin, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventBegin, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventBegin, ptxm.LacpPtxMachineNoPeriodic)
	// LACP DISABLED -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventLacpDisabled, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventLacpDisabled, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventLacpDisabled, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventLacpDisabled, ptxm.LacpPtxMachineNoPeriodic)
	// PORT DISABLED -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventPortDisabled, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventPortDisabled, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventPortDisabled, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventPortDisabled, ptxm.LacpPtxMachineNoPeriodic)
	// ACTOR/PARTNER OPER STATE ACTIVITY MODE == PASSIVE -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventActorPartnerOperActivityPassiveMode, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventActorPartnerOperActivityPassiveMode, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventActorPartnerOperActivityPassiveMode, ptxm.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventActorPartnerOperActivityPassiveMode, ptxm.LacpPtxMachineNoPeriodic)
	// INTENTIONAL FALL THROUGH -> FAST PERIODIC
	rules.AddRule(LacpPtxmStateNoPeriodic, LacpPtxmEventUnconditionalFallthrough, ptxm.LacpPtxMachineFastPeriodic)
	// PARTNER OPER STAT LACP TIMEOUT == LONG -> SLOW PERIODIC
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventPartnerOperStateTimeoutLong, ptxm.LacpPtxMachineSlowPeriodic)
	// PERIODIC TIMER EXPIRED -> PERIODIC TX
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventPeriodicTimerExpired, ptxm.LacpPtxMachinePeriodicTx)
	// PARTNER OPER STAT LACP TIMEOUT == SHORT ->  PERIODIC TX
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventPartnerOperStateTimeoutShort, ptxm.LacpPtxMachinePeriodicTx)
	// PERIODIC TIMER EXPIRED -> PERIODIC TX
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventPeriodicTimerExpired, ptxm.LacpPtxMachinePeriodicTx)
	// PARTNER OPER STAT LACP TIMEOUT == SHORT ->  FAST PERIODIC
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventPartnerOperStateTimeoutShort, ptxm.LacpPtxMachineFastPeriodic)
	// PARTNER OPER STAT LACP TIMEOUT == LONG -> SLOW PERIODIC
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventPartnerOperStateTimeoutLong, ptxm.LacpPtxMachineSlowPeriodic)

	// Create a new FSM and apply the rules
	ptxm.Apply(&rules)

	return ptxm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpPtxMachineMain(beginWaitChan chan string) {

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	ptxm := LacpPtxMachineFSMBuild(p)

	// set the inital state
	ptxm.Machine.Start(ptxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpPtxMachine) {
		ptxm.LacpPtxLog("PTXM: Machine Start")
		beginWaitChan <- "Periodic Tx Machine"
		select {
		case <-m.PtxmKillSignalEvent:
			ptxm.LacpPtxLog("PTXM: Machine End")
			return

		case event := <-m.PtxmEvents:
			m.Machine.ProcessEvent(event, nil)
			/* special case */
			if m.Machine.Curr.CurrentState() == LacpPtxmStateNoPeriodic {
				m.Machine.ProcessEvent(LacpPtxmEventUnconditionalFallthrough, nil)
			}
		case ena := <-m.PtxmLogEnableEvent:
			m.logEna = ena
		}
	}(ptxm)
}
