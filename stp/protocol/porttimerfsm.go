// 17.22 Port Timers state machine
package stp

import (
	"fmt"
	//"time"
	"strings"
	"utils/fsm"
)

const PtmMachineModuleStr = "Port Timer State Machine"

const (
	PtmStateNone = iota + 1
	PtmStateOneSecond
	PtmStateTick
)

var PtmStateStrMap map[fsm.State]string

func PtmMachineStrStateMapInit() {
	PtmStateStrMap = make(map[fsm.State]string)
	PtmStateStrMap[PtmStateNone] = "None"
	PtmStateStrMap[PtmStateOneSecond] = "OneSecond"
	PtmStateStrMap[PtmStateTick] = "Tick"
}

const (
	PtmEventBegin = iota + 1
	PtmEventTickEqualsTrue
	PtmEventUnconditionalFallthrough
)

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type PtmMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	// State transition log
	log chan string

	// timer type
	MachineTimerType TimerType
	Tick             bool

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PtmEvents chan MachineEvent
	// stop go routine
	PtmKillSignalEvent chan bool
	// enable logging
	PtmLogEnableEvent chan bool
}

func (ptm *PtmMachine) PrevState() fsm.State { return ptm.PreviousState }

// PrevStateSet will set the previous State
func (ptm *PtmMachine) PrevStateSet(s fsm.State) { ptm.PreviousState = s }

func (ptm *PtmMachine) Stop() {

	ptm.TimerDestroy()

	ptm.PtmKillSignalEvent <- true

	close(ptm.PtmEvents)
	close(ptm.PtmKillSignalEvent)
	close(ptm.PtmLogEnableEvent)
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewStpPtmMachine(p *StpPort, t TimerType) *PtmMachine {
	ptm := &PtmMachine{
		MachineTimerType:   t,
		p:                  p,
		PreviousState:      PtmStateNone,
		PtmEvents:          make(chan MachineEvent, 10),
		PtmKillSignalEvent: make(chan bool),
		PtmLogEnableEvent:  make(chan bool)}

	// start then stop
	ptm.TimerStart()
	ptm.TimerStop()

	return ptm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (ptm *PtmMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if ptm.Machine == nil {
		ptm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	ptm.Machine.Rules = r
	ptm.Machine.Curr = &StpStateEvent{
		strStateMap: PtmStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		//logger: ptxm.LacpPtxmLog,
		owner: strings.Join([]string{PtmMachineModuleStr, TimerTypeStrMap[ptm.MachineTimerType]}, ""),
	}

	return ptm.Machine
}

// LacpPtxMachineNoPeriodic stops the periodic transmission of packets
func (ptm *PtmMachine) PtmMachineOneSecond(m fsm.Machine, data interface{}) fsm.State {
	ptm.Tick = false
	return PtmStateOneSecond
}

// LacpPtxMachineFastPeriodic sets the periodic transmission time to fast
// and starts the timer
func (ptm *PtmMachine) PtmMachineTick(m fsm.Machine, data interface{}) fsm.State {
	p := ptm.p
	p.DecramentCounter(ptm.MachineTimerType)

	return PtmStateTick
}

func PtmMachineFSMBuild(p *StpPort, tt TimerType) *PtmMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new LacpPtxMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the NO PERIODIC State
	ptm := NewStpPtmMachine(p, tt)

	//BEGIN -> ONE SECOND
	rules.AddRule(PtmStateNone, PtmEventBegin, ptm.PtmMachineOneSecond)
	rules.AddRule(PtmStateOneSecond, PtmEventBegin, ptm.PtmMachineOneSecond)
	rules.AddRule(PtmStateTick, PtmEventBegin, ptm.PtmMachineOneSecond)

	// TICK EQUALS TRUE	 -> TICK
	rules.AddRule(PtmStateOneSecond, PtmEventTickEqualsTrue, ptm.PtmMachineTick)

	// PORT DISABLED -> NO PERIODIC
	rules.AddRule(PtmStateTick, PtmEventUnconditionalFallthrough, ptm.PtmMachineOneSecond)

	// Create a new FSM and apply the rules
	ptm.Apply(&rules)

	return ptm
}

// LacpRxMachineMain:  802.1ax-2014 Table 6-18
// Creation of Rx State Machine State transitions and callbacks
// and create go routine to pend on events
func (p *StpPort) PtmMachineMain(tt TimerType) {

	// Build the State machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	ptm := PtmMachineFSMBuild(p, tt)

	// set the inital State
	ptm.Machine.Start(ptm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PtmMachine, tt TimerType) {
		timerNameStr := TimerTypeStrMap[tt]
		fmt.Println(fmt.Sprintf("PTM: %s Machine Start", timerNameStr))
		//defer m.p.wg.Done()
		for {
			select {
			case <-m.PtmKillSignalEvent:
				fmt.Println(fmt.Sprintf("PTM: %s Machine End", TimerTypeStrMap[tt]))
				return
				/*
					case <-m.periodicTxTimer.C:

						//m.LacpPtxmLog("Timer expired current State")
						//m.LacpPtxmLog(PtxmStateStrMap[m.Machine.Curr.CurrentState()])
						m.Machine.ProcessEvent(PtxMachineModuleStr, PtmEventTickEqualsTrue, nil)
				*/

			case event := <-m.PtmEvents:
				m.Machine.ProcessEvent(event.src, event.e, nil)

				if event.responseChan != nil {
					SendResponse(timerNameStr, event.responseChan)
				}

			case ena := <-m.PtmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(ptm, tt)
}
