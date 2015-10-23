// CHURN DETECTION MACHINE 802.1ax-2014 Section 6.4.17
package lacp

import (
	"time"
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

	// debug log
	log    chan string
	logEna bool

	// timer intervals
	actorChurnTimerInterval   time.Duration
	partnerChurnTimerInterval time.Duration

	// Interval timers
	actorChurnTimer   *time.Timer
	partnerChurnTimer *time.Timer

	// machine specific events
	CdmEvents          chan fsm.Event
	CdmKillSignalEvent chan bool
	CdmLogEnableEvent  chan bool
}

func (cdm *LacpCdMachine) Stop() {
	close(cdm.CdmEvents)
	close(cdm.CdmKillSignalEvent)
	close(cdm.CdmLogEnableEvent)
}

func (cdm *LacpCdMachine) LacpCdmLog(msg string) {
	if cdm.logEna {
		cdm.log <- msg
	}
}

func (cdm *LacpCdMachine) PrevState() fsm.State { return cdm.PreviousState }

// PrevStateSet will set the previous state
func (cdm *LacpCdMachine) PrevStateSet(s fsm.State) { cdm.PreviousState = s }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpCdMachine(port *LaAggPort) *LacpCdMachine {
	cdm := &LacpCdMachine{
		p:                         port,
		log:                       port.LacpDebug.LacpLogChan,
		logEna:                    true,
		PreviousState:             LacpCdmStateNone,
		actorChurnTimerInterval:   LacpChurnDetectionTime,
		partnerChurnTimerInterval: LacpChurnDetectionTime,
		CdmEvents:                 make(chan fsm.Event),
		CdmKillSignalEvent:        make(chan bool),
		CdmLogEnableEvent:         make(chan bool)}

	port.CdMachineFsm = cdm

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
	cdm.Machine.Curr = &LacpStateEvent{}

	return cdm.Machine
}

// LacpCdMachineNoActorChurn will set the churn state to false
func (cdm *LacpCdMachine) LacpCdMachineNoActorChurn(m fsm.Machine, data interface{}) fsm.State {
	cdm.LacpCdmLog("CDM: No Actor Churn Enter")
	p := cdm.p
	p.actorChurn = false
	return LacpCdmStateNoActorChurn
}

// LacpCdMachineActorChurn will set the churn state to true
func (cdm *LacpCdMachine) LacpCdMachineActorChurn(m fsm.Machine, data interface{}) fsm.State {
	cdm.LacpCdmLog("CDM: Actor Churn Enter")
	p := cdm.p

	p.actorChurn = true
	return LacpCdmStateActorChurn
}

// LacpCdMachineActorChurnMonitor will set the churn state to true
// and kick off the churn detection timer
func (cdm *LacpCdMachine) LacpCdMachineActorChurnMonitor(m fsm.Machine, data interface{}) fsm.State {
	cdm.LacpCdmLog("CDM: Actor Churn Monitor")
	p := cdm.p
	p.actorChurn = true
	cdm.ChurnDetectionTimerStart()
	// let the port know we have initialized
	if p.begin {
		p.beginChan <- "Churn Detection Machine"
	}
	return LacpTxmStateOff
}

func LacpCdMachineFSMBuild(p *LaAggPort) *LacpCdMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	cdm := NewLacpCdMachine(p)

	//BEGIN -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateNone, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)

	// PORT ENABLED -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventPortEnabled, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventPortEnabled, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventPortEnabled, cdm.LacpCdMachineActorChurnMonitor)

	// Create a new FSM and apply the rules
	cdm.Apply(&rules)

	return cdm
}

// LacpCdMachineMain:  802.1ax-2014 Figure 6.23
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpCdMachineMain() {

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	cdm := LacpCdMachineFSMBuild(p)

	// set the inital state
	cdm.Machine.Start(cdm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpCdMachine) {
		m.LacpCdmLog("CDM: Machine Start")
		for {
			select {
			case <-m.CdmKillSignalEvent:
				m.LacpCdmLog("CDM: Machine End")
				return

			case event := <-m.CdmEvents:
				m.Machine.ProcessEvent(event, nil)

			case ena := <-m.CdmLogEnableEvent:
				m.logEna = ena
			}
		}
	}(cdm)
}
