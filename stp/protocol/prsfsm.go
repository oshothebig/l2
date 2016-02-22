// 802.1D-2004 17.28 Port Role Selection State Machine
//The Port Role Selection state machine shall implement the function specified by the state diagram in Figure
//17-19, the definitions in 17.13, 17.16, 17.20, and 17.21, and the variable declarations in 17.17, 17.18, and
//17.19. It selects roles for all Bridge Ports.
//On initialization all Bridge Ports are assigned the Disabled Port Role. Whenever any Bridge Port’s reselect
//variable (17.19.34) is set by the Port Information state machine (17.27), spanning tree information including
//the designatedPriority (17.19.4) and designatedTimes (17.19.5) for each Port is recomputed and its Port
//Role (selectedRole, 17.19.37) updated by the updtRolesTree() procedure (17.21.25). The reselect variables
//are cleared before computation starts so that recomputation will take place if new information becomes
//available while the computation is in progress.
package stp

import (
	"fmt"
	//"time"
	"utils/fsm"
)

const PrsMachineModuleStr = "Port Role Selection State Machine"

const (
	PrsStateNone = iota + 1
	PrsStateInitBridge
	PrsStateRoleSelection
)

var PrsStateStrMap map[fsm.State]string

func PrsMachineStrStateMapInit() {
	PrsStateStrMap = make(map[fsm.State]string)
	PrsStateStrMap[PrsStateNone] = "None"
	PrsStateStrMap[PrsStateInitBridge] = "Init Bridge"
	PrsStateStrMap[PrsStateRoleSelection] = "Role Selection"
}

const (
	PrsEventBegin = iota + 1
	PrsEventUnconditionallFallThrough
	PrsEventReselect
)

// PrsMachine holds FSM and current State
// and event channels for State transitions
type PrsMachine struct {
	Machine *fsm.Machine

	// State transition log
	log chan string

	// Reference to StpPort
	b *Bridge

	// debug level
	debugLevel int

	// machine specific events
	PrsEvents chan MachineEvent
	// stop go routine
	PrsKillSignalEvent chan bool
	// enable logging
	PrsLogEnableEvent chan bool
}

func (m *PrsMachine) GetCurrStateStr() string {
	return PrsStateStrMap[m.Machine.Curr.CurrentState()]
}

func (m *PrsMachine) GetPrevStateStr() string {
	return PrsStateStrMap[m.Machine.Curr.PreviousState()]
}

// NewStpPimMachine will create a new instance of the LacpRxMachine
func NewStpPrsMachine(b *Bridge) *PrsMachine {
	prsm := &PrsMachine{
		b:                  b,
		debugLevel:         1,
		PrsEvents:          make(chan MachineEvent, 10),
		PrsKillSignalEvent: make(chan bool),
		PrsLogEnableEvent:  make(chan bool)}

	b.PrsMachineFsm = prsm

	return prsm
}

func (prsm *PrsMachine) PrsLogger(s string) {
	StpMachineLogger("INFO", "PRSM", prsm.b.BrgIfIndex, s)
}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (prsm *PrsMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if prsm.Machine == nil {
		prsm.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	prsm.Machine.Rules = r
	prsm.Machine.Curr = &StpStateEvent{
		strStateMap: PrsStateStrMap,
		logEna:      prsm.debugLevel > 0,
		logger:      prsm.PrsLogger,
		owner:       PrsMachineModuleStr,
		ps:          PrsStateNone,
		s:           PrsStateNone,
	}

	return prsm.Machine
}

// Stop should clean up all resources
func (prsm *PrsMachine) Stop() {

	// stop the go routine
	prsm.PrsKillSignalEvent <- true

	close(prsm.PrsEvents)
	close(prsm.PrsLogEnableEvent)
	close(prsm.PrsKillSignalEvent)

}

// PrsMachineInitBridge
func (prsm *PrsMachine) PrsMachineInitBridge(m fsm.Machine, data interface{}) fsm.State {
	prsm.updtRoleDisabledTree()
	return PrsStateInitBridge
}

// PrsMachineRoleSelection
func (prsm *PrsMachine) PrsMachineRoleSelection(m fsm.Machine, data interface{}) fsm.State {
	prsm.clearReselectTree()
	prsm.updtRolesTree()
	prsm.setSelectedTree()

	return PrsStateRoleSelection
}

func PrsMachineFSMBuild(b *Bridge) *PrsMachine {

	rules := fsm.Ruleset{}

	// Instantiate a new PrxmMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the DISCARD State
	prsm := NewStpPrsMachine(b)

	// BEGIN -> INIT_BRIDGE
	rules.AddRule(PrsStateNone, PrsEventBegin, prsm.PrsMachineInitBridge)

	// UNINTENTIONAL FALL THROUGH -> ROLE SELECTION
	rules.AddRule(PrsStateInitBridge, PrsEventUnconditionallFallThrough, prsm.PrsMachineRoleSelection)

	// RESLECT -> ROLE SELECTION
	rules.AddRule(PrsStateRoleSelection, PrsEventReselect, prsm.PrsMachineRoleSelection)

	// Create a new FSM and apply the rules
	prsm.Apply(&rules)

	return prsm
}

// PrsMachineMain:
func (b *Bridge) PrsMachineMain() {

	// Build the State machine for STP Bridge Detection State Machine according to
	// 802.1d Section 17.25
	prsm := PrsMachineFSMBuild(b)
	b.wg.Add(1)

	// set the inital State
	prsm.Machine.Start(prsm.Machine.Curr.PreviousState())

	// lets create a go routing which will wait for the specific events
	// that the Port Timer State Machine should handle
	go func(m *PrsMachine) {
		StpMachineLogger("INFO", "PRSM", 0, "Machine Start")
		defer m.b.wg.Done()
		for {
			select {
			case <-m.PrsKillSignalEvent:
				StpMachineLogger("INFO", "PRSM", 0, "Machine End")
				return

			case event := <-m.PrsEvents:
				//fmt.Println("Event Rx", event.src, event.e)
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)
				if rv != nil {
					StpMachineLogger("ERROR", "PRSM", 0, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, event.e, PrsStateStrMap[m.Machine.Curr.CurrentState()]))
				} else {
					if m.Machine.Curr.CurrentState() == PrsStateInitBridge {
						rv := m.Machine.ProcessEvent(PrsMachineModuleStr, PrsEventUnconditionallFallThrough, nil)
						if rv != nil {
							StpMachineLogger("ERROR", "PRSM", 0, fmt.Sprintf("%s event[%d] currState[%s]\n", rv, event.e, PrsStateStrMap[m.Machine.Curr.CurrentState()]))
						}
					}
				}

				if event.responseChan != nil {
					SendResponse(PrsMachineModuleStr, event.responseChan)
				}

			case ena := <-m.PrsLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)
			}
		}
	}(prsm)
}

// clearReselectTree: 17.21.2
func (prsm *PrsMachine) clearReselectTree() {
	var p *StpPort
	b := prsm.b

	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			if p.PortEnabled {
				p.Reselect = false
			}
		}
	}
}

func (prsm *PrsMachine) updtRoleDisabledTree() {
	var p *StpPort
	b := prsm.b

	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleDisabledPort)
			p.SelectedRole = PortRoleDisabledPort
		}
	}
}

// updtRolesTree: 17.21.25
//STP_VECT_create (OUT PRIO_VECTOR_T* t,
//                 IN BRIDGE_ID* root_br,
//                IN unsigned long root_path_cost,
//                 IN BRIDGE_ID* design_bridge,
//                 IN PORT_ID design_port,
//                 IN PORT_ID bridge_port)
func (prsm *PrsMachine) updtRolesTree() {

	b := prsm.b

	var p *StpPort
	var rootPortId int32
	rootPathVector := PriorityVector{
		RootBridgeId:       b.BridgePriority.DesignatedBridgeId,
		DesignatedBridgeId: b.BridgePriority.DesignatedBridgeId,
	}

	rootTimes := Times{
		ForwardingDelay: b.BridgeTimes.ForwardingDelay,
		HelloTime:       b.BridgeTimes.HelloTime,
		MaxAge:          b.BridgeTimes.MaxAge,
		MessageAge:      b.BridgeTimes.MessageAge,
	}

	tmpVector := rootPathVector

	// lets consider each port a root to begin with
	myBridgeId := rootPathVector.RootBridgeId

	// lets find the root port
	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			if prsm.debugLevel > 0 {
				StpMachineLogger("INFO", "PRSM", p.IfIndex, fmt.Sprintf("updtRolesTree: InfoIs %d", p.InfoIs))
			}
			if p.InfoIs == PortInfoStateReceived {

				if CompareBridgeAddr(GetBridgeAddrFromBridgeId(myBridgeId),
					GetBridgeAddrFromBridgeId(p.PortPriority.DesignatedBridgeId)) == 0 {
					continue
				}

				compare := CompareBridgeId(p.PortPriority.RootBridgeId, tmpVector.RootBridgeId)
				switch compare {
				case -1:
					if prsm.debugLevel > 0 {
						StpMachineLogger("INFO", "PRSM", p.IfIndex, fmt.Sprintf("updtRolesTree: Root Bridge Received is SUPERIOR port Priority %#v", p.PortPriority))
					}
					tmpVector.RootBridgeId = p.PortPriority.RootBridgeId
					tmpVector.RootPathCost = p.PortPriority.RootPathCost + p.PortPathCost
					tmpVector.DesignatedBridgeId = p.PortPriority.DesignatedBridgeId
					tmpVector.DesignatedPortId = p.PortPriority.DesignatedPortId
					rootPortId = int32(p.Priority<<8 | p.PortId)
					rootTimes = p.PortTimes
				case 0:
					if prsm.debugLevel > 0 {
						StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: Root Bridge Received by port SAME")
					}
					tmpCost := p.PortPriority.RootPathCost + p.PortPathCost
					if prsm.debugLevel > 0 {
						StpMachineLogger("INFO", "PRSM", p.IfIndex, fmt.Sprintf("updtRolesTree: rx+txCost[%d] bridgeCost[%d]", tmpCost, tmpVector.RootPathCost))
					}
					if tmpCost < tmpVector.RootPathCost {
						if prsm.debugLevel > 0 {
							StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: DesignatedBridgeId received by port is SUPERIOR")
						}
						tmpVector.RootPathCost = tmpCost
						tmpVector.DesignatedBridgeId = p.PortPriority.DesignatedBridgeId
						tmpVector.DesignatedPortId = p.PortPriority.DesignatedPortId
						rootPortId = int32(p.Priority<<8 | p.PortId)
					} else if tmpCost == tmpVector.RootPathCost {
						if prsm.debugLevel > 0 {
							StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: DesignatedBridgeId received by port is SAME")
						}
						if p.PortPriority.DesignatedPortId <
							tmpVector.DesignatedPortId {
							if prsm.debugLevel > 0 {
								StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: DesignatedPortId received by port is SUPPERIOR")
							}
							tmpVector.DesignatedPortId = p.PortPriority.DesignatedPortId
							rootPortId = int32(p.Priority<<8 | p.PortId)
						} else if p.PortPriority.DesignatedPortId ==
							tmpVector.DesignatedPortId {
							var rp *StpPort
							var localPortId int32
							if StpFindPortById(rootPortId, &rp) {
								rootPortId = int32((rp.Priority << 8) | p.PortId)
								localPortId = int32((p.Priority << 8) | p.PortId)
								if localPortId < rootPortId {
									if prsm.debugLevel > 0 {
										StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: received portId is SUPPERIOR")
									}
									rootPortId = int32(p.Priority<<8 | p.PortId)
								}
							}
						}
					}
				}
			}
		}
	}

	// lets copy over the tmpVector over to the rootPathVector
	if rootPortId != 0 {
		if prsm.debugLevel > 0 {
			StpMachineLogger("INFO", "PRSM", -1, fmt.Sprintf("updtRolesTree: Port %d selected as the root port", rootPortId))
		}
		compare := CompareBridgeAddr(GetBridgeAddrFromBridgeId(b.BridgePriority.RootBridgeId),
			GetBridgeAddrFromBridgeId(tmpVector.RootBridgeId))
		if compare != 0 {
			b.OldRootBridgeIdentifier = b.BridgePriority.RootBridgeId
		}

		b.BridgePriority.RootBridgeId = tmpVector.RootBridgeId
		b.BridgePriority.BridgePortId = tmpVector.BridgePortId
		b.BridgePriority.RootPathCost = tmpVector.RootPathCost
		b.RootTimes = rootTimes
		b.RootTimes.MessageAge += 1
		b.RootPortId = rootPortId
	} else {
		if prsm.debugLevel > 0 {
			StpMachineLogger("INFO", "PRSM", 0, "updtRolesTree: This bridge is the root bridge")
		}
		compare := CompareBridgeAddr(GetBridgeAddrFromBridgeId(b.BridgeIdentifier),
			GetBridgeAddrFromBridgeId(tmpVector.RootBridgeId))
		if compare != 0 {
			b.OldRootBridgeIdentifier = b.BridgePriority.RootBridgeId
		}

		b.BridgePriority.RootBridgeId = tmpVector.RootBridgeId
		b.BridgePriority.BridgePortId = tmpVector.BridgePortId
		b.BridgePriority.RootPathCost = tmpVector.RootPathCost
		b.RootTimes = rootTimes
		b.RootPortId = rootPortId

	}
	if prsm.debugLevel > 0 {
		StpMachineLogger("INFO", "PRSM", -1, fmt.Sprintf("BridgePriority: %#v", b.BridgePriority))
	}
	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {

			// 17.21.25 (e)
			p.PortTimes = b.RootTimes
			p.PortTimes.HelloTime = b.BridgeTimes.HelloTime

			if prsm.debugLevel > 0 {
				StpMachineLogger("INFO", "PRSM", p.IfIndex, fmt.Sprintf("updtRolesTree: portEnabled %t, infoIs %d\n", p.PortEnabled, p.InfoIs))
			}
			// Assign the port roles
			if !p.PortEnabled || p.InfoIs == PortInfoStateDisabled {
				// 17.21.25 (f) if port is disabled
				defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleDisabledPort)
				p.SelectedRole = PortRoleDisabledPort

				if prsm.debugLevel > 0 {
					StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree:1 port role selected DISABLED")
				}
			} else if p.InfoIs == PortInfoStateAged {
				// 17.21.25 (g)
				defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, true)
				p.UpdtInfo = true
				defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleDesignatedPort)
				p.SelectedRole = PortRoleDesignatedPort
				if prsm.debugLevel > 0 {
					StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree:1 port role selected DESIGNATED")
				}
			} else if p.InfoIs == PortInfoStateMine {
				// 17.21.25 (h)
				defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleDesignatedPort)
				p.SelectedRole = PortRoleDesignatedPort
				if p.b.BridgePriority == p.PortPriority ||
					p.PortTimes != b.RootTimes {
					defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, true)
					p.UpdtInfo = true
				}
				if prsm.debugLevel > 0 {
					StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree:2 port role selected DESIGNATED")
				}
			} else if p.InfoIs == PortInfoStateReceived {
				// 17.21.25 (i)
				if rootPortId == int32(p.Priority<<8|p.PortId) {
					defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleRootPort)
					p.SelectedRole = PortRoleRootPort
					defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, false)
					p.UpdtInfo = false
					if prsm.debugLevel > 0 {
						StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: port role selected ROOT")
					}
				} else {
					// 17.21.25 (j), (k), (l)
					// designated not higher than port priority
					if IsDesignatedPriorytVectorNotHigherThanPortPriorityVector(&p.b.BridgePriority, &p.PortPriority) {
						if CompareBridgeAddr(GetBridgeAddrFromBridgeId(p.PortPriority.DesignatedBridgeId),
							GetBridgeAddrFromBridgeId(myBridgeId)) != 0 {
							defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleAlternatePort)
							p.SelectedRole = PortRoleAlternatePort
							defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, false)
							p.UpdtInfo = false
							if prsm.debugLevel > 0 {
								StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: port role selected ALTERNATE")
							}
						} else {

							if (p.Priority<<8 | p.PortId) != p.PortPriority.DesignatedPortId {
								defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleBackupPort)
								p.SelectedRole = PortRoleBackupPort
								defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, false)
								p.UpdtInfo = false
								if prsm.debugLevel > 0 {
									StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree: port role selected BACKUP")
								}
							} else {
								//if p.SelectedRole != PortRoleDesignatedPort {
								defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleDesignatedPort)
								p.SelectedRole = PortRoleDesignatedPort
								defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, true)
								p.UpdtInfo = true
								//}
								//StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree:3 port role selected DESIGNATED")
							}
						}
					} else {
						//if p.SelectedRole != PortRoleDesignatedPort {
						defer p.NotifySelectedRoleChanged(PrsMachineModuleStr, p.SelectedRole, PortRoleDesignatedPort)
						p.SelectedRole = PortRoleDesignatedPort
						defer p.NotifyUpdtInfoChanged(PrsMachineModuleStr, p.UpdtInfo, true)
						p.UpdtInfo = true
						//}
						if prsm.debugLevel > 0 {
							StpMachineLogger("INFO", "PRSM", p.IfIndex, "updtRolesTree:4 port role selected DESIGNATED")
						}
					}
				}
			}
		}
	}
}

// setSelectedTree: 17.21.16
func (prsm *PrsMachine) setSelectedTree() {
	var p *StpPort
	b := prsm.b
	setAllSelectedTrue := true

	for _, pId := range b.StpPorts {
		if StpFindPortById(pId, &p) {
			if p.Reselect == true {
				if prsm.debugLevel > 0 {
					StpMachineLogger("INFO", "PRSM", p.IfIndex, "setSelectedTree: is in reselet mode")
				}
				setAllSelectedTrue = false
				break
			}
		}
	}
	if setAllSelectedTrue {
		if prsm.debugLevel > 0 {
			StpMachineLogger("INFO", "PRSM", -1, "setSelectedTree: setting all ports as selected")
		}
		for _, pId := range b.StpPorts {
			if StpFindPortById(pId, &p) {
				if prsm.debugLevel > 0 {
					StpMachineLogger("INFO", "PRSM", p.IfIndex, fmt.Sprintf("setSelectedTree: setting selected prev selected state %t", p.Selected))
				}
				defer p.NotifySelectedChanged(PrsMachineModuleStr, p.Selected, true)
				p.Selected = true
			}
		}
	}
}
