// The Periodic Transmission Machine is described in the 802.1ax-2014 Section 6.4.13
package lacp

import (
	"fmt"
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

	// machine specific events
	PtxmEvents          chan fsm.Event
	PtxmKillSignalEvent chan bool
}

func (ptxm *LacpPtxMachine) PrevState() fsm.State { return ptxm.PreviousState }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpPtxMachine() *LacpPtxMachine {
	ptxm := &LacpPtxMachine{
		PreviousState:       LacpPtxmStateNone,
		PtxmEvents:          make(chan fsm.Event),
		PtxmKillSignalEvent: make(chan bool)}

	return ptxm
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
func (p *LaAggPort) LacpPtxMachineNoPeriodic(m fsm.Machine, data interface{}) fsm.State {
	p.PeriodicTimerStop()
	return LacpPtxmStateNoPeriodic
}

// LacpPtxMachineFastPeriodic sets the periodic transmission time to fast
// and starts the timer
func (p *LaAggPort) LacpPtxMachineFastPeriodic(m fsm.Machine, data interface{}) fsm.State {
	p.PeriodicTimerIntervalSet(LacpFastPeriodicTime)
	p.PeriodicTimerStart()
	return LacpPtxmStateFastPeriodic
}

// LacpPtxMachineSlowPeriodic sets the periodic transmission time to slow
// and starts the timer
func (p *LaAggPort) LacpPtxMachineSlowPeriodic(m fsm.Machine, data interface{}) fsm.State {
	p.PeriodicTimerIntervalSet(LacpSlowPeriodicTime)
	p.PeriodicTimerStart()
	return LacpPtxmStateSlowPeriodic
}

// LacpPtxMachinePeriodicTx changes the ntt flag to true
func (p *LaAggPort) LacpPtxMachinePeriodicTx(m fsm.Machine, data interface{}) fsm.State {
	p.nttFlag = true
	// TODO SEND event to TX Machine
	return LacpPtxmStatePeriodicTx
}

func (p *LaAggPort) LacpPtxMachineFSMBuild() *LacpPtxMachine {

	rules := fsm.Ruleset{}

	//BEGIN -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventBegin, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventBegin, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventBegin, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventBegin, p.LacpPtxMachineNoPeriodic)
	// LACP DISABLED -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventLacpDisabled, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventLacpDisabled, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventLacpDisabled, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventLacpDisabled, p.LacpPtxMachineNoPeriodic)
	// PORT DISABLED -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventPortDisabled, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventPortDisabled, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventPortDisabled, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventPortDisabled, p.LacpPtxMachineNoPeriodic)
	// ACTOR/PARTNER OPER STATE ACTIVITY MODE == PASSIVE -> NO PERIODIC
	rules.AddRule(LacpPtxmStateNone, LacpPtxmEventActorPartnerOperActivityPassiveMode, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventActorPartnerOperActivityPassiveMode, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventActorPartnerOperActivityPassiveMode, p.LacpPtxMachineNoPeriodic)
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventActorPartnerOperActivityPassiveMode, p.LacpPtxMachineNoPeriodic)
	// INTENTIONAL FALL THROUGH -> FAST PERIODIC
	rules.AddRule(LacpPtxmStateNoPeriodic, LacpPtxmEventUnconditionalFallthrough, p.LacpPtxMachineFastPeriodic)
	// PARTNER OPER STAT LACP TIMEOUT == LONG -> SLOW PERIODIC
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventPartnerOperStateTimeoutLong, p.LacpPtxMachineSlowPeriodic)
	// PERIODIC TIMER EXPIRED -> PERIODIC TX
	rules.AddRule(LacpPtxmStateFastPeriodic, LacpPtxmEventPeriodicTimerExpired, p.LacpPtxMachinePeriodicTx)
	// PARTNER OPER STAT LACP TIMEOUT == SHORT ->  PERIODIC TX
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventPartnerOperStateTimeoutShort, p.LacpPtxMachinePeriodicTx)
	// PERIODIC TIMER EXPIRED -> PERIODIC TX
	rules.AddRule(LacpPtxmStateSlowPeriodic, LacpPtxmEventPeriodicTimerExpired, p.LacpPtxMachinePeriodicTx)
	// PARTNER OPER STAT LACP TIMEOUT == SHORT ->  FAST PERIODIC
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventPartnerOperStateTimeoutShort, p.LacpPtxMachineFastPeriodic)
	// PARTNER OPER STAT LACP TIMEOUT == LONG -> SLOW PERIODIC
	rules.AddRule(LacpPtxmStatePeriodicTx, LacpPtxmEventPartnerOperStateTimeoutLong, p.LacpPtxMachineSlowPeriodic)

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	p.ptxMachineFsm = NewLacpPtxMachine()

	// Create a new FSM and apply the rules
	p.ptxMachineFsm.Apply(&rules)

	return p.ptxMachineFsm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpPtxMachineMain() {

	// initialize the port
	p.begin = true

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	ptxm := p.LacpPtxMachineFSMBuild()

	// set the inital state
	ptxm.Machine.Start(ptxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpPtxMachine) {
		select {
		case <-m.PtxmKillSignalEvent:
			fmt.Println("PTXM: Machine Killed")
			return

		case event := <-m.PtxmEvents:
			m.Machine.ProcessEvent(event, nil)
			/* special case */
			if m.Machine.Curr.CurrentState() == LacpPtxmStateNoPeriodic {
				m.Machine.ProcessEvent(LacpPtxmEventUnconditionalFallthrough, nil)
			}

			//if event == LacpPtxEventPeriodicTimerExpired {
			// TODO SEND PACKET OUT
			//}
		}
	}(ptxm)
}
