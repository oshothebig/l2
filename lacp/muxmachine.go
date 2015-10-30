// MUX MACHINE 802.1ax-2014 Section 6.4.15
// This implementation will assume that bot state machines in Section 6.4.15 are
// implemented with an extra flag indicating the capabilities of the port
package lacp

import (
	//"fmt"
	"strconv"
	"strings"
	"time"
	"utils/fsm"
)

const MuxMachineModuleStr = "Mux Machine"

const (
	LacpMuxmStateNone = iota
	LacpMuxmStateDetached
	LacpMuxmStateWaiting
	LacpMuxmStateAttached
	LacpMuxmStateCollecting
	LacpMuxmStateDistributing
	// Coupled control - Collecting and Distributing can't be controlled independently
	LacpMuxmStateCNone
	LacpMuxmStateCDetached
	LacpMuxmStateCWaiting
	LacpMuxmStateCAttached
	LacpMuxStateCCollectingDistributing
)

var MuxmStateStrMap map[fsm.State]string

func MuxxMachineStrStateMapCreate() {

	MuxmStateStrMap = make(map[fsm.State]string)
	MuxmStateStrMap[LacpMuxmStateNone] = "None"
	MuxmStateStrMap[LacpMuxmStateDetached] = "Detached"
	MuxmStateStrMap[LacpMuxmStateWaiting] = "Waiting"
	MuxmStateStrMap[LacpMuxmStateAttached] = "Attached"
	MuxmStateStrMap[LacpMuxmStateCollecting] = "Collecting"
	MuxmStateStrMap[LacpMuxmStateDistributing] = "Distributing"
	MuxmStateStrMap[LacpMuxmStateCNone] = "LacpMuxmStateCNone"
	MuxmStateStrMap[LacpMuxmStateCDetached] = "CDetached"
	MuxmStateStrMap[LacpMuxmStateCWaiting] = "CWaiting"
	MuxmStateStrMap[LacpMuxmStateCAttached] = "CAttached"
	MuxmStateStrMap[LacpMuxStateCCollectingDistributing] = "CCollectingDistributing"
}

const (
	LacpMuxmEventBegin = iota + 1
	LacpMuxmEventSelectedEqualSelected
	LacpMuxmEventSelectedEqualStandby
	LacpMuxmEventSelectedEqualUnselected
	LacpMuxmEventSelectedEqualSelectedAndReady
	LacpMuxmEventSelectedEqualSelectedAndPartnerSync
	LacpMuxmEventNotPartnerSync
	LacpMuxmEventNotPartnerCollecting
	LacpMuxmEventSelectedEqualSelectedPartnerSyncCollecting
)

// LacpRxMachine holds FSM and current state
// and event channels for state transitions
type LacpMuxMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *LaAggPort

	// debug log
	log    chan string
	logEna bool

	collDistCoupled bool

	// timer interval
	waitWhileTimerTimeout time.Duration
	waitWhileTimerRunning bool

	// timers
	waitWhileTimer *time.Timer

	// machine specific events
	MuxmEvents          chan LacpMachineEvent
	MuxmKillSignalEvent chan bool
	MuxmLogEnableEvent  chan bool
}

func (muxm *LacpMuxMachine) Stop() {
	close(muxm.MuxmEvents)
	close(muxm.MuxmKillSignalEvent)
	close(muxm.MuxmLogEnableEvent)
}

func (muxm *LacpMuxMachine) PrevState() fsm.State { return muxm.PreviousState }

// PrevStateSet will set the previous state
func (muxm *LacpMuxMachine) PrevStateSet(s fsm.State) { muxm.PreviousState = s }

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpMuxMachine(port *LaAggPort) *LacpMuxMachine {
	muxm := &LacpMuxMachine{
		p:                     port,
		log:                   port.LacpDebug.LacpLogChan,
		collDistCoupled:       false,
		waitWhileTimerTimeout: LacpAggregateWaitTime,
		PreviousState:         LacpMuxmStateNone,
		MuxmEvents:            make(chan LacpMachineEvent),
		MuxmKillSignalEvent:   make(chan bool),
		MuxmLogEnableEvent:    make(chan bool)}

	port.MuxMachineFsm = muxm

	// start then stop
	muxm.WaitWhileTimerStart()
	muxm.WaitWhileTimerStop()

	return muxm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances state machine without reallocating the machine.
func (muxm *LacpMuxMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if muxm.Machine == nil {
		muxm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	muxm.Machine.Rules = r
	muxm.Machine.Curr = &LacpStateEvent{
		strStateMap: MuxmStateStrMap,
		logEna:      muxm.p.logEna,
		logger:      muxm.LacpMuxmLog,
		owner:       MuxMachineModuleStr,
	}

	return muxm.Machine
}

func (muxm *LacpMuxMachine) SendTxMachineNtt() {

	muxm.p.TxMachineFsm.TxmEvents <- LacpMachineEvent{e: LacpTxmEventNtt,
		src: MuxMachineModuleStr}
}

// LacpMuxmDetached
func (muxm *LacpMuxMachine) LacpMuxmDetached(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p

	// DETACH MUX FROM AGGREGATOR
	muxm.DetachMuxFromAggregator()

	// Actor Oper State Sync = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateSyncBit)

	// Disable Distributing
	muxm.DisableDistributing()

	// Actor Oper State Distributing = FALSE
	// Actor Oper State Collecting = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateDistributingBit|LacpStateCollectingBit)

	// Disable Collecting
	muxm.DisableCollecting()

	// NTT = TRUE
	// TODO: is this necessary? May only want to let TxMachine
	//       set ntt to true based on NTT event
	//p.TxMachineFsm.ntt = true

	// indicate that NTT = TRUE
	if muxm.Machine.Curr.CurrentState() != LacpMuxmStateNone {
		defer muxm.SendTxMachineNtt()
	}

	return LacpMuxmStateDetached
}

// LacpMuxmWaiting
func (muxm *LacpMuxMachine) LacpMuxmWaiting(m fsm.Machine, data interface{}) fsm.State {
	var a *LaAggregator
	//var state fsm.State
	p := muxm.p

	skipWaitWhileTimer := false

	// only need to kick off the timer if ready is not true
	// ready will be true if all other ports are attached
	// or this is the the first
	// or lacp is not enabled
	if LaFindAggById(p.aggId, &a) {
		if a.ready || LacpModeGet(p.actorAdmin.state, p.lacpEnabled) == LacpModeOn {
			skipWaitWhileTimer = true
			a.ready = false
		}
	}

	//state = LacpMuxmStateWaiting
	if !skipWaitWhileTimer {
		muxm.WaitWhileTimerStart()
	} else {
		muxm.WaitWhileTimerStop()
		// force the the next state to attach
		//muxm.LacpMuxmWaitingEvaluateSelected(true)
		//muxm.Machine.Curr.CurrentState()
	}

	return LacpMuxmStateWaiting
}

// LacpMuxmAttached
func (muxm *LacpMuxMachine) LacpMuxmAttached(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p
	// Attach Mux to Aggregator
	muxm.AttachMuxToAggregator()

	// Actor Oper State Sync = TRUE
	LacpStateSet(&p.actorOper.state, LacpStateSyncBit)

	// Actor Oper State Collecting = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateCollectingBit)

	// Disable Collecting
	muxm.DisableCollecting()

	// NTT = TRUE
	defer muxm.SendTxMachineNtt()

	return LacpMuxmStateAttached
}

// LacpMuxmCollecting
func (muxm *LacpMuxMachine) LacpMuxmCollecting(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p

	// Enabled Collecting
	muxm.EnableCollecting()

	// Actor Oper State Sync == TRUE
	LacpStateSet(&p.actorOper.state, LacpStateCollectingBit)

	// Disable Distributing
	muxm.DisableDistributing()

	// Actor Oper State Distributing = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateDistributingBit)

	// indicate that NTT = TRUE
	defer muxm.SendTxMachineNtt()

	return LacpMuxmStateWaiting
}

// LacpMuxmDistributing
func (muxm *LacpMuxMachine) LacpMuxmDistributing(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p

	// Actor Oper State Sync == TRUE
	LacpStateSet(&p.actorOper.state, LacpStateDistributingBit)

	// Enabled Distributing
	muxm.EnableDistributing()

	return LacpMuxmStateWaiting
}

// LacpMuxmCDetached
func (muxm *LacpMuxMachine) LacpMuxmCDetached(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p

	// DETACH MUX FROM AGGREGATOR
	muxm.DetachMuxFromAggregator()

	// Actor Oper State Sync = FALSE
	// Actor Oper State Collecting = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateSyncBit|LacpStateCollectingBit)

	// Disable Collecting && Distributing
	muxm.DisableCollectingDistributing()

	// Actor Oper State Distributing = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateDistributingBit)

	// indicate that NTT = TRUE
	defer muxm.SendTxMachineNtt()

	return LacpMuxmStateDetached
}

// LacpMuxmCWaiting
func (muxm *LacpMuxMachine) LacpMuxmCWaiting(m fsm.Machine, data interface{}) fsm.State {
	//p := muxm.p

	muxm.WaitWhileTimerStart()

	return LacpMuxmStateWaiting
}

// LacpMuxmAttached
func (muxm *LacpMuxMachine) LacpMuxmCAttached(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p

	// Attach Mux to Aggregator
	muxm.AttachMuxToAggregator()

	// Actor Oper State Sync = TRUE
	LacpStateSet(&p.actorOper.state, LacpStateSyncBit)

	// Actor Oper State Collecting = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateCollectingBit)

	// Disable Collecting && Distributing
	muxm.DisableCollectingDistributing()

	// Actor Oper State Distributing = FALSE
	LacpStateClear(&p.actorOper.state, LacpStateDistributingBit)

	// indicate that NTT = TRUE
	defer muxm.SendTxMachineNtt()

	return LacpMuxmStateWaiting
}

// LacpMuxmCollecting
func (muxm *LacpMuxMachine) LacpMuxmCCollectingDistributing(m fsm.Machine, data interface{}) fsm.State {
	p := muxm.p

	// Actor Oper State Distributing = TRUE
	LacpStateSet(&p.actorOper.state, LacpStateDistributingBit)

	// Enable Collecting && Distributing
	muxm.EnableCollectingDistributing()

	// Actor Oper State Distributing == FALSE
	LacpStateSet(&p.actorOper.state, LacpStateDistributingBit)

	// indicate that NTT = TRUE
	defer muxm.SendTxMachineNtt()

	return LacpMuxmStateWaiting
}

// LacpMuxMachineFSMBuild:  802.1ax-2014 Figure 6-21 && 6-22
func (p *LaAggPort) LacpMuxMachineFSMBuild() *LacpMuxMachine {

	rules := fsm.Ruleset{}

	MuxxMachineStrStateMapCreate()

	// Instantiate a new LacpRxMachine
	// Initial state will be a psuedo state known as "begin" so that
	// we can transition to the initalize state
	muxm := NewLacpMuxMachine(p)

	// MUX

	//BEGIN -> DETACHED
	rules.AddRule(LacpMuxmStateNone, LacpMuxmEventBegin, muxm.LacpMuxmDetached)
	// SELECTED or STANDBY -> WAITING
	rules.AddRule(LacpMuxmStateDetached, LacpMuxmEventSelectedEqualSelected, muxm.LacpMuxmWaiting)
	rules.AddRule(LacpMuxmStateDetached, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmWaiting)
	// UNSELECTED -> DETACHED
	rules.AddRule(LacpMuxmStateWaiting, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmDetached)
	// SELECTED && READY -> ATTACHED
	rules.AddRule(LacpMuxmStateWaiting, LacpMuxmEventSelectedEqualSelectedAndReady, muxm.LacpMuxmAttached)
	// UNSELECTED or STANDBY -> DETACHED
	rules.AddRule(LacpMuxmStateAttached, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmDetached)
	rules.AddRule(LacpMuxmStateAttached, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmDetached)
	// SELECTED && PARTNER SYNC -> COLLECTING
	rules.AddRule(LacpMuxmStateAttached, LacpMuxmEventSelectedEqualSelectedAndPartnerSync, muxm.LacpMuxmCollecting)
	// UNSELECTED or STANDBY or NOT PARTNER SYNC -> ATTACHED
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmAttached)
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmAttached)
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventNotPartnerSync, muxm.LacpMuxmAttached)
	// SELECTED && PARTNER SYNC && PARTNER COLLECTING -> DISTRIBUTING
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventSelectedEqualSelectedPartnerSyncCollecting, muxm.LacpMuxmDistributing)
	// UNSELECTED or STANDBY or NOT PARTNER SYNC or NOT PARTNER COLLECTING -> COLLECTING
	rules.AddRule(LacpMuxmStateDistributing, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmCollecting)
	rules.AddRule(LacpMuxmStateDistributing, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmCollecting)
	rules.AddRule(LacpMuxmStateDistributing, LacpMuxmEventNotPartnerSync, muxm.LacpMuxmCollecting)
	rules.AddRule(LacpMuxmStateDistributing, LacpMuxmEventNotPartnerCollecting, muxm.LacpMuxmCollecting)

	// MUX Coupled
	//BEGIN -> DETACHED
	rules.AddRule(LacpMuxmStateNone, LacpMuxmEventBegin, muxm.LacpMuxmCDetached)
	// SELECTED or STANDBY -> WAITING
	rules.AddRule(LacpMuxmStateCDetached, LacpMuxmEventSelectedEqualSelected, muxm.LacpMuxmCWaiting)
	rules.AddRule(LacpMuxmStateCDetached, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmCWaiting)
	// UNSELECTED -> DETACHED
	rules.AddRule(LacpMuxmStateCWaiting, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmCDetached)
	// SELECTED && READY -> ATTACHED
	rules.AddRule(LacpMuxmStateCWaiting, LacpMuxmEventSelectedEqualSelectedAndReady, muxm.LacpMuxmAttached)
	// UNSELECTED or STANDBY -> DETACHED
	rules.AddRule(LacpMuxmStateCAttached, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmCDetached)
	rules.AddRule(LacpMuxmStateCAttached, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmCDetached)
	// SELECTED && PARTNER SYNC -> COLLECTING-DISTRIBUTING
	rules.AddRule(LacpMuxmStateCAttached, LacpMuxmEventSelectedEqualSelectedAndPartnerSync, muxm.LacpMuxmCCollectingDistributing)
	// UNSELECTED or STANDBY or NOT PARTNER SYNC -> ATTACHED
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventSelectedEqualUnselected, muxm.LacpMuxmCAttached)
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventSelectedEqualStandby, muxm.LacpMuxmCAttached)
	rules.AddRule(LacpMuxmStateCollecting, LacpMuxmEventNotPartnerSync, muxm.LacpMuxmCAttached)

	// Create a new FSM and apply the rules
	muxm.Apply(&rules)

	return muxm
}

// LacpMuxMachineMain:  802.1ax-2014 Figure 6-21 && 6-22
// Creation of Rx State Machine state transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LacpMuxMachineMain() {

	// Build the state machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.13 Periodic Transmission Machine
	muxm := p.LacpMuxMachineFSMBuild()

	// Hw only supports mux coupling
	if LacpSystemParams.muxCoupling {
		muxm.PrevStateSet(LacpMuxmStateCNone)
	}
	// set the inital state
	muxm.Machine.Start(muxm.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LacpMuxMachine) {
		m.LacpMuxmLog("Machine Start")
		for {
			select {
			case <-m.MuxmKillSignalEvent:
				m.LacpMuxmLog("Machine End")
				return

			case <-m.waitWhileTimer.C:
				m.LacpMuxmLog("MUXM: Wait While Timer Expired")
				// lets evaluate selection
				if m.Machine.Curr.CurrentState() == LacpMuxmStateWaiting ||
					m.Machine.Curr.CurrentState() == LacpMuxmStateCWaiting {
					m.LacpMuxmWaitingEvaluateSelected(false)
				}

			case event := <-m.MuxmEvents:

				p := m.p
				//m.LacpMuxmLog(strings.Join([]string{"Event received", strconv.Itoa(int(event.e))}, ":"))

				// process the event
				m.Machine.ProcessEvent(event.src, event.e, nil)

				// continuation events
				if m.Machine.Curr.CurrentState() == LacpMuxmStateDetached ||
					m.Machine.Curr.CurrentState() == LacpMuxmStateCDetached {
					if p.aggSelected == LacpAggSelected {
						//contEvt = LacpMachineEvent{e: LacpMuxmEventSelectedEqualSelected,
						//	src: MuxMachineModuleStr}
						//event.responseChan = nil
						m.Machine.ProcessEvent(MuxMachineModuleStr, LacpMuxmEventSelectedEqualSelected, nil)
					}
				} else if event.e == LacpMuxmEventSelectedEqualSelected &&
					(m.Machine.Curr.CurrentState() == LacpMuxmStateWaiting ||
						m.Machine.Curr.CurrentState() == LacpMuxmStateCWaiting) &&
					!m.waitWhileTimerRunning {
					// special case we may have a delayed event which will do a fast transition to next state
					// Attached, trigger is the fact that the timer is not running
					m.LacpMuxmWaitingEvaluateSelected(true)
				} else if (m.Machine.Curr.CurrentState() == LacpMuxmStateAttached ||
					m.Machine.Curr.CurrentState() == LacpMuxmStateCAttached) &&
					p.aggSelected == LacpAggSelected &&
					LacpStateIsSet(p.partnerOper.state, LacpStateSyncBit|LacpStateCollectingBit) {
					m.Machine.ProcessEvent(MuxMachineModuleStr, LacpMuxmEventSelectedEqualSelectedPartnerSyncCollecting, nil)
				} else if event.e == LacpMuxmEventSelectedEqualUnselected &&
					(m.Machine.Curr.CurrentState() != LacpMuxmStateDetached &&
						m.Machine.Curr.CurrentState() != LacpMuxmStateCDetached) {
					// Unselected state will cause a downward transition to detached state
					state := m.Machine.Curr.CurrentState()
					endState := fsm.State(LacpMuxmStateDetached)
					if m.Machine.Curr.CurrentState() > LacpMuxmStateDistributing {
						endState = LacpMuxmStateCDetached
					}
					for ; state > endState; state-- {
						m.Machine.ProcessEvent(MuxMachineModuleStr, LacpMuxmEventSelectedEqualUnselected, nil)
					}
				}

				if event.responseChan != nil {
					//m.LacpMuxmLog("Sending response")
					SendResponse(MuxMachineModuleStr, event.responseChan)
				}

			case ena := <-m.MuxmLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(muxm)
}

// LacpMuxmEvaluateSelected 802.1ax-2014 Section 6.4.15
// d) If Selected is SELECTED, the wait_while_timer forces a delay to allow
// for the possibility that other Aggregation Ports may be reconfiguring
// at the same time. Once the wait_while_timer expires, and once the wait_
// while_timers of all other Aggregation Ports that are ready to attach to
// the same Aggregator have expired, the process of attaching the Aggregation
// Port to the Aggregator can proceed, and the state machine enters the
// ATTACHED state. During the waiting time, changes in selection parameters
// can occur that will result in a re-evaluation of Selected. If Selected
// becomes UNSELECTED, then the state machine reenters the DETACHED state.
// If Selected becomes STANDBY, the operation is as described in item e).
//
// NOTE—This waiting period reduces the disturbance that will be visible
// to higher layers; for example, on start-up events. However, the selection
// need not wait for the entire waiting period in cases where it is known that
// no other Aggregation Ports will attach; for example, where all other
// Aggregation Ports with the same operational Key are already attached to the
// Aggregator.
//
// e) If Selected is STANDBY, the Aggregation Port is held in the WAITING
// state until such a time as the selection parameters change, resulting in a
// re-evaluation of the Selected variable. If Selected becomes UNSELECTED,
// the state machine reenters the DETACHED state. If SELECTED becomes SELECTED,
// then the operation is as described in item d). The latter case allows an
// Aggregation Port to be brought into operation from STANDBY with minimum
// delay once Selected becomes SELECTED.
func (muxm *LacpMuxMachine) LacpMuxmWaitingEvaluateSelected(sendResponse bool) {
	var a *LaAggregator
	p := muxm.p
	muxm.LacpMuxmLog(strings.Join([]string{"Selected", strconv.Itoa(LacpAggSelected), "actual", strconv.Itoa(p.aggSelected)}, "="))
	// current port should be in selected state
	if p.aggSelected == LacpAggSelected ||
		p.aggSelected == LacpAggStandby {
		p.readyN = true
		if LaFindAggById(p.aggId, &a) {
			a.LacpMuxCheckSelectionLogic(p, sendResponse)
		} else {
			muxm.LacpMuxmLog(strings.Join([]string{"MUXM: Unable to find Aggrigator", string(p.aggId)}, ":"))
		}
	}
	//else {
	//	muxm.Machine.processEvent(LacpMuxmEventSelectedEqualUnselected, nil)
	//}

}

// AttachMuxToAggregator is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregation Port’s Control Parser/Multiplexer
// to be attached to the Aggregator Parser/Multiplexer of the selected
// Aggregator, in preparation for collecting and distributing frames.
func (muxm *LacpMuxMachine) AttachMuxToAggregator() {
	// TODO send message to asic deamon  create
	p := muxm.p
	if LaFindAggById(p.aggId, &p.aggAttached) {
		muxm.LacpMuxmLog("Attach Mux To Aggregator Enter, send Add PORT to ASICD")
	}
}

// DetachMuxFromAggregator is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregation Port’s Control Parser/Multiplexer
// to be detached from the Aggregator Parser/Multiplexer of the Aggregator
// to which the Aggregation Port is currently attached.
func (muxm *LacpMuxMachine) DetachMuxFromAggregator() {
	// TODO send message to asic deamon delete
	muxm.LacpMuxmLog("Detach Mux From Aggregator Enter")
	p := muxm.p
	p.aggAttached = nil
	// should already be in unselected state
	//p.aggSelected = LacpAggUnSelected

	// Remove port from HW lag group
}

// EnableCollecting is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregator Parser of the Aggregator to which
// the Aggregation Port is attached to start collecting frames from the
// Aggregation Port.
func (muxm *LacpMuxMachine) EnableCollecting() {
	// TODO send message to asic deamon
	muxm.LacpMuxmLog("Sending Collection Enable to ASICD")
}

// DisableCollecting is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregator Parser of the Aggregator to which
// the Aggregation Port is attached to stop collecting frames from the
// Aggregation Port.
func (muxm *LacpMuxMachine) DisableCollecting() {
	// TODO send message to asic deamon
	p := muxm.p
	if LacpStateIsSet(p.actorOper.state, LacpStateCollectingBit) {
		muxm.LacpMuxmLog("Sending Collection Disable to ASICD")
	}
}

// EnableDistributing is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregator Multiplexer of the Aggregator
// to which the Aggregation Port is attached to start distributing frames
// to the Aggregation Port.
func (muxm *LacpMuxMachine) EnableDistributing() {
	// TODO send message to asic deamon
	muxm.LacpMuxmLog("Sending Distributing Enable to ASICD")
}

// DisableDistributing is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregator Multiplexer of the Aggregator
// to which the Aggregation Port is attached to stop distributing frames
// to the Aggregation Port.
func (muxm *LacpMuxMachine) DisableDistributing() {
	// TODO send message to asic deamon
	muxm.LacpMuxmLog("Sending Distributing Disable to ASICD")
}

// EnableCollectingDistributing is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregator Parser of the Aggregator to which
// the Aggregation Port is attached to start collecting frames from the
// Aggregation Port, and the Aggregator Multiplexer to start distributing
// frames to the Aggregation Port.
func (muxm *LacpMuxMachine) EnableCollectingDistributing() {
	// TODO send message to asic deamon
	muxm.LacpMuxmLog("Sending Collection-Distributing Enable to ASICD")
}

// DisableCollectingDistributing is a required function defined in 802.1ax-2014
// Section 6.4.9
// This function causes the Aggregator Parser of the Aggregator to which the
// Aggregation Port is attached to stop collecting frames from the Aggregation
// Port, and the Aggregator Multiplexer to stop distributing frames to the
// Aggregation Port.
func (muxm *LacpMuxMachine) DisableCollectingDistributing() {
	// TODO send message to asic deamon
	muxm.LacpMuxmLog("Sending Collection-Distributing Disable to ASICD")
}
