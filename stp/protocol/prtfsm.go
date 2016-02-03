// 802.1D-2004 17.29 Port Role Selection State Machine
//The Port Role Selection state machine shall implement the function specified by the state diagram in Figure
//17-19, the definitions in 17.13, 17.16, 17.20, and 17.21, and the variable declarations in 17.17, 17.18, and
//17.19. It selects roles for all Bridge Ports.
//On initialization all Bridge Ports are assigned the Disabled Port Role. Whenever any Bridge Port’s reselect
//variable (17.19.34) is set by the Port Information state machine (17.27), spanning tree information including
//the designatedPriority (17.19.4) and designatedTimes (17.19.5) for each Port is recomputed and its Port
//Role (selectedRole, 17.19.37) updated by the updtRolesTree() procedure (17.21.25). The reselect variables
//are cleared before computation starts so that recomputation will take place if new information becomes
//available while the computation is in progress.
//
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const PrtMachineModuleStr = "Port Role Transitions State Machine"

const (
	PrtStateNone = iota + 1
	// Role: Disabled
	PrtStateInitPort
	PrtStateDisablePort
	PrtStateDisabledPort
	// Role Root
	PrtStateRootPort
	PrtStateReRoot
	PrtStateRootAgreed
	PrtStateRootProposed
	PrtStateRootForward
	PrtStateRootLearn
	PrtStateReRooted
	// Role Designated
	PrtStateDesignatedPort
	PrtStateDesignatedRetired
	PrtStateDesignatedSynced
	PrtStateDesignatedPropose
	PrtStateDesignatedForward
	PrtStateDesignatedLearn
	PrtStateDesignatedDiscard
	// Role Alternate Backup
	PrtStateAlternatePort
	PrtStateAlternateAgreed
	PrtStateAlternateProposed
	PrtStateBlockPort
	PrtStateBackupPort
)

var PrtStateStrMap map[fsm.State]string

func PrtMachineStrStateMapInit() {
	PrtStateStrMap = make(map[fsm.State]string)
	PrtStateStrMap[PrtStateNone] = "None"
	PrtStateStrMap[PrtStateInitPort] = "Init Port"
	PrtStateStrMap[PrtStateDisablePort] = "Disable Port"
	PrtStateStrMap[PrtStateDisabledPort] = "Disabled Port"
	PrtStateStrMap[PrtStateRootPort] = "Root Port"
	PrtStateStrMap[PrtStateReRoot] = "Re-Root"
	PrtStateStrMap[PrtStateRootAgreed] = "Root Agreed"
	PrtStateStrMap[PrtStateRootProposed] = "Root Proposed"
	PrtStateStrMap[PrtStateRootForward] = "Root Forward"
	PrtStateStrMap[PrtStateRootLearn] = "Root Learn"
	PrtStateStrMap[PrtStateReRooted] = "Re-Rooted"
	PrtStateStrMap[PrtStateDesignatedPort] = "Designated Port"
	PrtStateStrMap[PrtStateDesignatedRetired] = "Designated Retired"
	PrtStateStrMap[PrtStateDesignatedSynced] = "Designated Synced"
	PrtStateStrMap[PrtStateDesignatedPropose] = "Designated Propose"
	PrtStateStrMap[PrtStateDesignatedForward] = "Designated Forward"
	PrtStateStrMap[PrtStateDesignatedLearn] = "Designated Learn"
	PrtStateStrMap[PrtStateDesignatedDiscard] = "Designated Discard"
	PrtStateStrMap[PrtStateAlternatePort] = "Alternate Port"
	PrtStateStrMap[PrtStateAlternateAgreed] = "Alternate Agreed"
	PrtStateStrMap[PrtStateAlternateProposed] = "Alternate Proposed"
	PrtStateStrMap[PrtStateBlockPort] = "Block Port"
	PrtStateStrMap[PrtStateBackupPort] = "Backup Port"
}

const (
	PrtEventBegin = iota + 1
	PrtEventUnconditionallFallThrough
	// events taken from Figure 17.20 Disabled Port role transitions
	PrtEventSelectedRoleEqualsDisabledPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo
	PrtEventNotLearningAndNotForwardingAndSelectedAndNotUpdtInfo
	PrtEventFdWhileNotEqualMaxAgeAndSelectedAndNotUpdtInfo
	PrtEventSyncAndSelectedAndNotUpdtInfo      // also applies to Alternate and Backup Port role
	PrtEventReRootAndSelectedAndNotUpdtInfo    // also applies to Alternate and Backup Port role
	PrtEventNotSyncedAndSelectedAndNotUpdtInfo // also applies to Alternate and Backup Port role
	// events taken from Figure 17.21 Root Port role transitions
	PrtEventSelectedRoleEqualRootPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo
	PrtEventProposedAndNotAgreeAndSelectedAndNotUpdtInfo  // also applies to Alternate and Backup Port role
	PrtEventAllSyncedAndNotAgreeAndSelectedAndNotUpdtInfo // also applies to Alternate and Backup Port role
	PrtEventProposedAndAgreeAndSelectedAndNotUpdtInfo     // also applies to Alternate and Backup Port role
	PrtEventNotForwardAndNotReRootAndSelectedAndNotUpdtInfo
	PrtEventRrWhileNotEqualFwdDelayAndSelectedAndNotUpdtInfo
	PrtEventReRootAndForwardAndSelectedAndNotUpdtInfo
	PrtEventFdWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventFdWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	PrtEventReRootedAndRbWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	// events take from Figure 17-22 Designated port role transitions
	PrtEventSelectedRoleEqualDesignatedPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo
	PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo
	PrtEventNotLearningAndNotForwardingAndNotSyncedAndSelectedAndNotUpdtInfo
	PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo
	PrtEventOperEdgeAndNotSyncedAndSelectedAndNotUpdtInfo
	PrtEventSyncAndSyncedAndSelectedAndNotUpdtInfo
	PrtEventRrWhileEqualZeroAndReRootAndSelectedAndNotUpdtInfo
	PrtEventSyncAndNotSyncedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo
	PrtEventSyncAndNotSyncedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo
	PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo
	PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo
	PrtEventDiputedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo
	PrtEventDisputedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo
	PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventFdWhileEqualZeroAndReRootAndNotSyncAndNotLearnSelectedAndNotUpdtInfo
	PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventOperEdgeAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo
	PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo
	PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo
	PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	PrtEventAgreedAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	PrtEventOperEdgeAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	PrtEventNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo
	// events taken from Figure 17-23 Alternate and Backup Port role transitions
	PrtEventSelectedRoleEqualAlternateAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo
	PrtEventSelectedRoleEqualBackupPortAndRoleNotEqualSelectedRoleAndSelectedAndNotUpdtInfo
	PrtEventRbWhileNotEqualTwoTimesHelloTimeAndRoleEqualsBackupPortAndSelectedAndNotUpdtInfo
	PrtEventFdWhileNotEqualForwardDelayAndSelectedAndNotUpdtInfo
)

// PrtMachine holds FSM and current State
// and event channels for State transitions
type PrtMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	p *StpPort

	// machine specific events
	PrtEvents chan MachineEvent
	// stop go routine
	PrtKillSignalEvent chan bool
	// enable logging
	PrtLogEnableEvent chan bool
}

// NewStpPrtMachine will create a new instance of the LacpRxMachine
func NewStpPrtMachine(p *StpPort) *PrtMachine {
	prtm := &PrtMachine{
		p:                  p,
		PrtEvents:          make(chan MachineEvent, 10),
		PrtKillSignalEvent: make(chan bool),
		PrtLogEnableEvent:  make(chan bool)}

	p.PrtMachineFsm = prtm

	return prtm
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (prtm *PrtMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if prtm.Machine == nil {
		prtm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	prtm.Machine.Rules = r
	prtm.Machine.Curr = &StpStateEvent{
		strStateMap: PrtStateStrMap,
		//logEna:      ptxm.p.logEna,
		logEna: false,
		logger: StpLoggerInfo,
		owner:  PrtMachineModuleStr,
		ps:     PrtStateNone,
		s:      PrtStateNone,
	}

	return prtm.Machine
}

// Stop should clean up all resources
func (prtm *PrtMachine) Stop() {

	// stop the go routine
	prtm.PrtKillSignalEvent <- true

	close(prtm.PrtEvents)
	close(prtm.PrtLogEnableEvent)
	close(prtm.PrtKillSignalEvent)

}

/*
// PimMachineCheckingRSTP
func (ppmm *PpmmMachine) PpmmMachineCheckingRSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p
	p.Mcheck = false
	p.SendRSTP = p.BridgeProtocolVersionGet() == layers.RSTPProtocolVersion
	// 17.24
	// TODO inform Port Transmit State Machine what STP version to send and which BPDU types
	// to support interoperability
	p.MdelayWhiletimer.count = MigrateTimeDefault
	return PpmmStateCheckingRSTP
}

// PpmmMachineSelectingSTP
func (ppmm *PpmmMachine) PpmmMachineSelectingSTP(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p

	p.SendRSTP = false
	// 17.24
	// TODO inform Port Transmit State Machine what STP version to send and which BPDU types
	// to support interoperability
	p.MdelayWhiletimer.count = MigrateTimeDefault

	return PpmmStateSelectingSTP
}

// PpmmMachineSelectingSTP
func (ppmm *PpmmMachine) PpmmMachineSensing(m fsm.Machine, data interface{}) fsm.State {
	p := ppmm.p

	p.RcvdRSTP = false
	p.RcvdSTP = false

	return PpmmStateSensing
}
*/
func PrtMachineFSMBuild(p *StpPort) *PrtMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrtMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	prtm := NewStpPrtMachine(p)

	// Create a new FSM and apply the rules
	prtm.Apply(&rules)

	return prtm
}

// PimMachineMain:
func (p *StpPort) PrtMachineMain() {

	// Build the State machine for STP Port Role Transitions State Machine according to
	// 802.1d Section 17.29
	prtm := PrtMachineFSMBuild(p)
	p.wg.Add(1)

	// set the inital State
	prtm.Machine.Start(prtm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	go func(m *PrtMachine) {
		StpMachineLogger("INFO", "PRTM", "Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.PrtKillSignalEvent:
				StpMachineLogger("INFO", "PRTM", "Machine End")
				return

			case event := <-m.PrtEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpMachineLogger("ERROR", "PRTM", fmt.Sprintf("%s\n", rv))
				}

				if event.responseChan != nil {
					SendResponse(PrtMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PrtLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(prtm)
}
