// CHURN DETECTION MACHINE 802.1ax-2014 Section 6.4.17
package lacp

import (
	"utils/fsm"
)

const (
	LacpCdmStateNone = iota + 1
	LacpCdmStateNoActorChurn
	LacpCdmStateActorChurnMonitor
	LacpCdmStateActorChurn
)

const (
	LacpCdmEventBegin = iota + 1
	LacpCdmEventPortEnabled
	LacpCdmEventActorOperPortStateSyncOn
	LacpCdmEventActorOperPortStateSyncOff
	LacpCdmEventActorChurnTimerExpired
)

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpCdMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *LaAggPort

	// machine specific events
	CdmEvents          chan fsm.Event
	CdmKillSignalEvent chan bool
}

func (cdm *LacpCdMachine) Stop() {
	close(cdm.CdmEvents)
	close(cdm.CdmKillSignalEvent)
}

func (cdm *LacpCdMachine) PrevState() fsm.State { return cdm.PreviousState }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpCdMachine(port *LaAggPort) *LacpCdMachine {
	cdm := &LacpCdMachine{
		p:                  port,
		PreviousState:      LacpCdmStateNone,
		CdmEvents:          make(chan fsm.Event),
		CdmKillSignalEvent: make(chan bool)}

	port.cdMachineFsm = cdm

	return cdm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine.
func (cdm *LacpCdMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if cdm.Machine == nil {
		cdm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	cdm.Machine.Rules = r

	return cdm.Machine
}

// LacpCdMachineNoActorChurn will set the churn state to false
func (p *LaAggPort) LacpCdMachineNoActorChurn(m fsm.Machine, data interface{}) fsm.State {
	p.actorChurn = false
	return LacpCdmStateNoActorChurn
}

// LacpCdMachineActorChurn will set the churn state to true
func (p *LaAggPort) LacpCdMachineActorChurn(m fsm.Machine, data interface{}) fsm.State {
	p.actorChurn = true
	return LacpCdmStateActorChurn
}

// LacpCdMachineActorChurnMonitor will set the churn state to true
// and kick off the churn detection timer
func (p *LaAggPort) LacpCdMachineActorChurnMonitor(m fsm.Machine, data interface{}) fsm.State {
	p.actorChurn = true
	p.ChurnDetectionTimerStart()
	return LacpTxmStateOff
}

func (p *LaAggPort) LacpCdMachineFSMBuild() *LacpCdMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	cdm := NewLacpCdMachine(p)

	//BEGIN -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateNone, LacpCdmEventBegin, p.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventBegin, p.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventBegin, p.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventBegin, p.LacpCdMachineActorChurnMonitor)

	// PORT ENABLED -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventPortEnabled, p.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventPortEnabled, p.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventPortEnabled, p.LacpCdMachineActorChurnMonitor)

	// Create a new FSM and apply the rules
	cdm.Apply(&rules)

	return cdm
}

// LacpCdMachineMain:  802.1ax-2014 Figure 6.23
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpCdMachineMain() {

	// initialize the port
	p.begin = true

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	cdm := p.LacpCdMachineFSMBuild()

	// set the inital state
	cdm.Machine.Start(cdm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpCdMachine) {
		m.p.LacpDebug.LacpLogStateTransitionChan <- "RXM: Machine Start"
		select {
		case <-m.CdmKillSignalEvent:
			m.p.LacpDebug.LacpLogStateTransitionChan <- "CDM: Machine End"
			return

		case event := <-m.CdmEvents:
			m.Machine.ProcessEvent(event, nil)
		}
	}(cdm)
}
