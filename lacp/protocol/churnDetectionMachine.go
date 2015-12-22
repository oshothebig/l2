// CHURN DETECTION MACHINE 802.1ax-2014 Section 6.4.17
package lacp

import (
	"strconv"
	"strings"
	"time"
	"utils/fsm"
)

const CdMachineModuleStr = "Churn Detection Machine"

const (
	LacpCdmStateNone = iota + 1
	LacpCdmStateNoActorChurn
	LacpCdmStateActorChurnMonitor
	LacpCdmStateActorChurn
)

var CdmStateStrMap map[fsm.State]string

func CdMachineStrStateMapCreate() {
	CdmStateStrMap = make(map[fsm.State]string)
	CdmStateStrMap[LacpCdmStateNone] = "None"
	CdmStateStrMap[LacpCdmStateNoActorChurn] = "NoActorChurn"
	CdmStateStrMap[LacpCdmStateActorChurnMonitor] = "ActorChurnMonitor"
	CdmStateStrMap[LacpCdmStateActorChurn] = "ActorChurn"
}

const (
	LacpCdmEventBegin = iota + 1
	LacpCdmEventNotPortEnabled
	LacpCdmEventActorOperPortStateSyncOn
	LacpCdmEventActorOperPortStateSyncOff
	LacpCdmEventActorChurnTimerExpired
)

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type LacpCdMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *LaAggPort

	// debug log
	log chan string

	// timer intervals
	actorChurnTimerInterval   time.Duration
	partnerChurnTimerInterval time.Duration

	// Interval timers
	actorChurnTimer   *time.Timer
	partnerChurnTimer *time.Timer

	// machine specific events
	CdmEvents          chan LacpMachineEvent
	CdmKillSignalEvent chan bool
	CdmLogEnableEvent  chan bool
}

func (cdm *LacpCdMachine) Stop() {

	// stop the go routine
	cdm.CdmKillSignalEvent <- true

	close(cdm.CdmEvents)
	close(cdm.CdmKillSignalEvent)
	close(cdm.CdmLogEnableEvent)
}

func (cdm *LacpCdMachine) PrevState() fsm.State { return cdm.PreviousState }

// PrevStateSet will set the previous State
func (cdm *LacpCdMachine) PrevStateSet(s fsm.State) { cdm.PreviousState = s }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpCdMachine(port *LaAggPort) *LacpCdMachine {
	cdm := &LacpCdMachine{
		p:                         port,
		log:                       port.LacpDebug.LacpLogChan,
		PreviousState:             LacpCdmStateNone,
		actorChurnTimerInterval:   LacpChurnDetectionTime,
		partnerChurnTimerInterval: LacpChurnDetectionTime,
		CdmEvents:                 make(chan LacpMachineEvent, 10),
		CdmKillSignalEvent:        make(chan bool),
		CdmLogEnableEvent:         make(chan bool)}

	port.CdMachineFsm = cdm

	return cdm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (cdm *LacpCdMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if cdm.Machine == nil {
		cdm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	cdm.Machine.Rules = r
	cdm.Machine.Curr = &LacpStateEvent{
		strStateMap: CdmStateStrMap,
		logEna:      cdm.p.logEna,
		logger:      cdm.LacpCdmLog,
		owner:       CdMachineModuleStr,
	}

	return cdm.Machine
}

// LacpCdMachineNoActorChurn will set the churn State to false
func (cdm *LacpCdMachine) LacpCdMachineNoActorChurn(m fsm.Machine, data interface{}) fsm.State {
	p := cdm.p
	p.actorChurn = false
	return LacpCdmStateNoActorChurn
}

// LacpCdMachineActorChurn will set the churn State to true
func (cdm *LacpCdMachine) LacpCdMachineActorChurn(m fsm.Machine, data interface{}) fsm.State {
	p := cdm.p
	p.actorChurn = true
	return LacpCdmStateActorChurn
}

// LacpCdMachineActorChurnMonitor will set the churn State to true
// and kick off the churn detection timer
func (cdm *LacpCdMachine) LacpCdMachineActorChurnMonitor(m fsm.Machine, data interface{}) fsm.State {
	p := cdm.p
	p.actorChurn = true
	cdm.ChurnDetectionTimerStart()
	return LacpCdmStateActorChurnMonitor
}

func LacpCdMachineFSMBuild(p *LaAggPort) *LacpCdMachine {

	rules := fsm.Ruleset{}

	CdMachineStrStateMapCreate()

	// Instantiate a new LacpRxMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	cdm := NewLacpCdMachine(p)

	//BEGIN -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateNone, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventBegin, cdm.LacpCdMachineActorChurnMonitor)

	// PORT ENABLED -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventNotPortEnabled, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventNotPortEnabled, cdm.LacpCdMachineActorChurnMonitor)
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventNotPortEnabled, cdm.LacpCdMachineActorChurnMonitor)

	// SYNC ON -> NO CHURN
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventActorOperPortStateSyncOn, cdm.LacpCdMachineNoActorChurn)
	rules.AddRule(LacpCdmStateActorChurn, LacpCdmEventActorOperPortStateSyncOn, cdm.LacpCdMachineNoActorChurn)

	// SYNC OFF -> ACTOR CHURN MONITOR
	rules.AddRule(LacpCdmStateNoActorChurn, LacpCdmEventActorOperPortStateSyncOff, cdm.LacpCdMachineActorChurnMonitor)

	// TIMEOUT -> CHURN
	rules.AddRule(LacpCdmStateActorChurnMonitor, LacpCdmEventActorChurnTimerExpired, cdm.LacpCdMachineActorChurn)

	// Create a new FSM and apply the rules
	cdm.Apply(&rules)

	return cdm
}

// LacpCdMachineMain:  802.1ax-2014 Figure 6.23
// Creation of Rx State Machine State transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpCdMachineMain() {

	// Build the State machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	cdm := LacpCdMachineFSMBuild(p)

	// set the inital State
	cdm.Machine.Start(cdm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpCdMachine) {
		m.LacpCdmLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.CdmKillSignalEvent:
				m.LacpCdmLog("Machine End")
				return

			case event := <-m.CdmEvents:
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)

				if rv != nil {
					m.LacpCdmLog(strings.Join([]string{error.Error(rv), event.src, CdmStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
				}

				if event.responseChan != nil {
					SendResponse(CdMachineModuleStr, event.responseChan)
				}
			case ena := <-m.CdmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(cdm)
}
