//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

// pi_test.go
package stp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"testing"
	"utils/fsm"
)

func UsedForTestOnlyPimInitPortConfigTest() {

	if PortConfigMap == nil {
		PortConfigMap = make(map[int32]portConfig)
	}
	// In order to test a packet we must listen on loopback interface
	// and send on interface we expect to receive on.  In order
	// to do this a couple of things must occur the PortConfig
	// must be updated with "dummy" ifindex pointing to 'lo'
	TEST_RX_PORT_CONFIG_IFINDEX = 0x0ADDBEEF
	PortConfigMap[TEST_RX_PORT_CONFIG_IFINDEX] = portConfig{Name: "lo"}
	PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX] = portConfig{Name: "lo"}
	/*
		intfs, err := net.Interfaces()
		if err == nil {
			for _, intf := range intfs {
				if strings.Contains(intf.Name, "eth") {
					ifindex, _ := strconv.Atoi(strings.Split(intf.Name, "eth")[1])
					if ifindex == 0 {
						TEST_TX_PORT_CONFIG_IFINDEX = int32(ifindex)
					}
					PortConfigMap[int32(ifindex)] = portConfig{Name: intf.Name}
				}
			}
		}
	*/
}

func UsedForTestOnlyPimTestSetup(t *testing.T, enable bool) (p *StpPort) {
	UsedForTestOnlyPimInitPortConfigTest()

	bridgeconfig := &StpBridgeConfig{
		Address:      "00:55:55:55:55:55",
		Priority:     0x20,
		MaxAge:       BridgeMaxAgeDefault,
		HelloTime:    BridgeHelloTimeDefault,
		ForwardDelay: BridgeForwardDelayDefault,
		ForceVersion: 2,
		TxHoldCount:  TransmitHoldCountDefault,
	}

	//StpBridgeCreate
	b := NewStpBridge(bridgeconfig)
	PrsMachineFSMBuild(b)
	b.PrsMachineFsm.Machine.ProcessEvent("TEST", PrsEventBegin, nil)
	b.PrsMachineFsm.Machine.ProcessEvent("TEST", PrsEventUnconditionallFallThrough, nil)

	// configure a port
	stpconfig := &StpPortConfig{
		IfIndex:           TEST_RX_PORT_CONFIG_IFINDEX,
		Priority:          0x80,
		Enable:            enable,
		PathCost:          1,
		ProtocolMigration: 0,
		AdminPointToPoint: StpPointToPointForceFalse,
		AdminEdgePort:     false,
		AdminPathCost:     0,
		BrgIfIndex:        DEFAULT_STP_BRIDGE_VLAN,
	}

	// create a port
	p = NewStpPort(stpconfig)

	// lets only start the Port Information State Machine
	b.PrsMachineMain()
	p.PimMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	// only instanciated object not starting go routine
	PrxmMachineFSMBuild(p)
	PtxmMachineFSMBuild(p)
	BdmMachineFSMBuild(p)
	PtmMachineFSMBuild(p)
	TcMachineFSMBuild(p)
	PstMachineFSMBuild(p)
	PpmmMachineFSMBuild(p)
	PrtMachineFSMBuild(p)

	if enable {
		// previous state is this because we default to enable
		if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateDisabled {
			t.Error("Failed to Initial Port Information machine state not set correctly", p.PpmmMachineFsm.Machine.Curr.PreviousState())
			t.FailNow()
		}

		if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateAged {
			t.Error("Failed to Initial Port Information machine state not set correctly", p.PpmmMachineFsm.Machine.Curr.CurrentState())
			t.FailNow()
		}
	} else {
		// previous state is this because we default to enable
		if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateNone {
			t.Error("Failed to Initial Port Information machine state not set correctly", p.PpmmMachineFsm.Machine.Curr.PreviousState())
			t.FailNow()
		}

		if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateDisabled {
			t.Error("Failed to Initial Port Information machine state not set correctly", p.PpmmMachineFsm.Machine.Curr.CurrentState())
			t.FailNow()
		}

	}

	return p
}

func UsedForTestOnlyPimTestTeardown(p *StpPort, t *testing.T) {
	if len(p.b.PrsMachineFsm.PrsEvents) > 0 {
		t.Error("Failed to check event sent")
	}

	if len(p.PpmmMachineFsm.PpmmEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.PimMachineFsm.PimEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.PtxmMachineFsm.PtxmEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.BdmMachineFsm.BdmEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.PtmMachineFsm.PtmEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.TcMachineFsm.TcEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.PstMachineFsm.PstEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.PpmmMachineFsm.PpmmEvents) > 0 {
		t.Error("Failed to check event sent")
	}
	if len(p.PrxmMachineFsm.PrxmEvents) > 0 {
		t.Error("Failed to check event sent")
	}

	p.PrtMachineFsm = nil
	p.PrxmMachineFsm = nil
	p.PtxmMachineFsm = nil
	p.BdmMachineFsm = nil
	p.PtmMachineFsm = nil
	p.TcMachineFsm = nil
	p.PstMachineFsm = nil
	p.PpmmMachineFsm = nil

	b := p.b
	DelStpPort(p)
	DelStpBridge(b, true)

}

func UsedForTestOnlyPimCheckDisabled(p *StpPort, t *testing.T) {

	// port receive
	if p.RcvdMsg != false {
		t.Error("Failed RcvdMsg is set")
	}
	if p.Proposed != false {
		t.Error("Failed proposed is set")
	}
	if p.Proposing != false {
		t.Error("Failed proposing is set")
	}
	if p.Agree != false {
		t.Error("Failed agree is set")
	}
	if p.Agreed != false {
		t.Error("Failed agreed is set")
	}
	if p.RcvdInfoWhiletimer.count != 0 {
		t.Error("Failed rcvdInfoWhile is not zero")
	}
	if p.InfoIs != PortInfoStateDisabled {
		t.Error("Failed Info infoIs is not Disabled")
	}
	if p.Reselect != true {
		t.Error("Failed reselect is not set")
	}
	if p.Selected != false {
		t.Error("Faild selected is set ")
	}
	// if rx machine was in receive state then rcvdMsg would have
	// equaled true
	if p.PrxmMachineFsm.Machine.Curr.CurrentState() == PrxmStateReceive &&
		p.RcvdBPDU &&
		p.PortEnabled {
		event, _ := <-p.PrxmMachineFsm.PrxmEvents
		if event.e != PrxmEventRcvdBpduAndPortEnabledAndNotRcvdMsg {
			t.Error("Failed to send event to Port Receive")
		}
	}

	// Port Role transition stae machine should receive an event based
	// not proposing
	if p.Role == PortRoleDesignatedPort &&
		p.Forward == false &&
		p.Agreed == false &&
		p.OperEdge == false &&
		p.Selected == true &&
		p.UpdtInfo == false &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Receive")
		}
	}
	// Port Role Selection machine should receive event if
	// reselected is set to true
	if p.b.PrsMachineFsm.Machine.Curr.CurrentState() == PrsStateRoleSelection {
		event, _ := <-p.b.PrsMachineFsm.PrsEvents
		if event.e != PrsEventReselect {
			t.Error("Failed to send event to Port Role Selection Machine")
		}
	}
}

func UsedForTestOnlyPimCheckAgedState(p *StpPort, t *testing.T) {
	if p.InfoIs != PortInfoStateAged {
		t.Error("Failed infoIs is not set to Aged")
	}
	if p.Reselect != true {
		t.Error("Failed reselected not set")
	}
	if p.Selected != false {
		t.Error("Failed selected is set")
	}

	if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateAged {
		t.Error(fmt.Sprintf("Failed invalid state state state %s", PimStateStrMap[p.PimMachineFsm.Machine.Curr.CurrentState()]))
	}

	// Port Role Selection machine should receive event if
	// reselected is set to true
	if p.b.PrsMachineFsm.Machine.Curr.CurrentState() == PrsStateRoleSelection {
		event, _ := <-p.b.PrsMachineFsm.PrsEvents
		if event.e != PrsEventReselect {
			t.Error("Failed to send event to Port Role Selection Machine")
		}
	}
}

func UsedForTestOnlyPimStartInAgedState(t *testing.T) *StpPort {
	testChan := make(chan string)
	p := UsedForTestOnlyPimTestSetup(t, false)

	p.PortEnabled = true

	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventPortEnabled,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan
	UsedForTestOnlyPimCheckAgedState(p, t)
	return p
}

func UsedForTestOnlyPimCheckUpdateState(p *StpPort, t *testing.T) {
	if p.Proposing != false {
		t.Error("Failed proposing is set")
	}
	if p.Proposed != false {
		t.Error("Failed proposed is set")
	}
	/* TODO need to look into betterorsameInfo func
	if p.Agreed != true {
		t.Error("Failed agreed is not set")
	}
	TODO dependant on the result of agreed
	if p.Synced != true {
		t.Error("Failed Synced is not set")
	}
	*/
	if p.PortTimes != p.b.RootTimes {
		t.Error("Failed Port Times not equal Designated times")
	}
	if p.UpdtInfo != false {
		t.Error("Failed updtInfo is set")
	}
	if p.InfoIs != PortInfoStateMine {
		t.Error("Failed infoIs not set to Mine")
	}
	if p.NewInfo != true {
		t.Error("Failed newInfo not set")
	}

	// Port Role Transitions state machine should be notified if
	// not proposing synced agreed agree is set
	if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Forward == false &&
		p.Agreed == false &&
		p.Proposing == false &&
		p.OperEdge == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventNotForwardAndNotAgreedAndNotProposingAndNotOperEdgeAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Learning == false &&
		p.Forwarding == false &&
		p.Synced == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventNotLearningAndNotForwardingAndNotSyncedAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Agreed == true &&
		p.Synced == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndNotSyncedAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.OperEdge == true &&
		p.Synced == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventOperEdgeAndNotSyncedAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Sync == true &&
		p.Synced == true &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventSyncAndSyncedAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Sync == true &&
		p.Synced == false &&
		p.OperEdge == false &&
		p.Learn == true &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventSyncAndNotSyncedAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Sync == true &&
		p.Synced == false &&
		p.OperEdge == false &&
		p.Forward == true &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventSyncAndNotSyncedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.ReRoot == true &&
		p.RrWhileTimer.count != 0 &&
		p.OperEdge == false &&
		p.Forward == true &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.ReRoot == true &&
		p.RrWhileTimer.count != 0 &&
		p.OperEdge == false &&
		p.Learn == true &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventReRootAndRrWhileNotEqualZeroAndNotOperEdgeAndLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Disputed == true &&
		p.OperEdge == false &&
		p.Learn == true &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventDisputedAndNotOperEdgeAndForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.FdWhileTimer.count == 0 &&
		p.RrWhileTimer.count == 0 &&
		p.Sync == false &&
		p.Learn == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.FdWhileTimer.count == 0 &&
		p.ReRoot == false &&
		p.Sync == false &&
		p.Learn == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndNotLearnSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Agreed == true &&
		p.RrWhileTimer.count == 0 &&
		p.Sync == false &&
		p.Learn == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Agreed == true &&
		p.ReRoot == false &&
		p.Sync == false &&
		p.Learn == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.OperEdge == true &&
		p.RrWhileTimer.count == 0 &&
		p.Sync == false &&
		p.Learn == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.OperEdge == true &&
		p.ReRoot == false &&
		p.Sync == false &&
		p.Learn == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventOperEdgeAndNotReRootAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.FdWhileTimer.count == 0 &&
		p.RrWhileTimer.count == 0 &&
		p.Sync == false &&
		p.Learn == true &&
		p.Forward == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.FdWhileTimer.count == 0 &&
		p.ReRoot == false &&
		p.Sync == false &&
		p.Learn == true &&
		p.Forward == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventFdWhileEqualZeroAndNotReRootAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Agreed == true &&
		p.RrWhileTimer.count == 0 &&
		p.Sync == false &&
		p.Learn == true &&
		p.Forward == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.Agreed == true &&
		p.ReRoot == false &&
		p.Sync == false &&
		p.Learn == true &&
		p.Forward == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.OperEdge == true &&
		p.RrWhileTimer.count == 0 &&
		p.Sync == false &&
		p.Learn == true &&
		p.Forward == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventOperEdgeAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDesignatedPort &&
		p.PrtMachineFsm.Machine.Curr.CurrentState() == PrtStateDesignatedPort &&
		p.OperEdge == true &&
		p.ReRoot == false &&
		p.Sync == false &&
		p.Learn == true &&
		p.Forward == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventAgreedAndNotReRootAndNotSyncAndLearnAndNotForwardAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	} else if p.Role == PortRoleDisabledPort &&
		p.Synced == false &&
		p.Selected == true &&
		p.UpdtInfo == false {
		event, _ := <-p.PrtMachineFsm.PrtEvents
		if event.e != PrtEventNotSyncedAndSelectedAndNotUpdtInfo {
			t.Error("Failed to send event to Port Role Transition Machine")
		}
	}
}

func UsedForTestOnlyPimStartInCurrentState(t *testing.T) *StpPort {
	testChan := make(chan string)
	p := UsedForTestOnlyPimStartInAgedState(t)

	p.Selected = true
	p.UpdtInfo = true
	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventSelectedAndUpdtInfo,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan
	if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateCurrent &&
		p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateUpdate {
		t.Error(fmt.Sprintf("Failed [Previous][Current state not correct [%s][%s]\n",
			PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()],
			PimStateStrMap[p.PimMachineFsm.Machine.Curr.CurrentState()]))
	}
	return p
}

func UsedForTestOnlyPimCheckSuperiorDesignatedState(p *StpPort, t *testing.T) {
	if p.RcvdInfo != SuperiorDesignatedInfo {
		t.Error("Failed RcvdInfo is not Superior")
	}
	if p.Agreed != false {
		t.Error("Failed agreed is set")
	}
	if p.Proposing != false {
		t.Error("Failed proposing is set")
	}
	if p.InfoIs != PortInfoStateReceived {
		t.Error("Failed InfoIs is not Received")
	}
	if p.Reselect != true {
		t.Error("Failed reselect is not set")
	}
	if p.Selected != false {
		t.Error("Failed selected is set")
	}
	if p.RcvdMsg != false {
		t.Error("Failed rcvdMsg is set")
	}

	// TODO check for events that should be sent
}

func TestPimBEGIN(t *testing.T) {
	// NOTE setup starts with BEGIN
	p := UsedForTestOnlyPimTestSetup(t, false)

	UsedForTestOnlyPimCheckDisabled(p, t)
	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimDisabledStateInvalidStateTransitions(t *testing.T) {
	testChan := make(chan string)
	p := UsedForTestOnlyPimTestSetup(t, false)

	invalidStateMap := [9]fsm.Event{
		PimEventRcvdMsgAndNotUpdtInfo,
		PimEventSelectedAndUpdtInfo,
		PimEventUnconditionalFallThrough,
		PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
		PimEventRcvdInfoEqualSuperiorDesignatedInfo,
		PimEventRcvdInfoEqualRepeatedDesignatedInfo,
		PimEventRcvdInfoEqualInferiorDesignatedInfo,
		PimEventRcvdInfoEqualInferiorRootAlternateInfo,
		PimEventRcvdInfoEqualOtherInfo,
	}

	// test the invalid states
	for _, e := range invalidStateMap {
		// one way to validate that a transition did not occur is to make sure
		// that mdelayWhiletimer is not set to MigrateTime
		p.MdelayWhiletimer.count = 1

		p.PimMachineFsm.PimEvents <- MachineEvent{e: e,
			src:          "TEST",
			responseChan: testChan}

		<-testChan
		if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateDisabled {
			t.Error(fmt.Sprintf("Failed e[%d] transitioned PIM to a new state state %d", e, p.PimMachineFsm.Machine.Curr.CurrentState()))
			t.FailNow()
		}
		if p.MdelayWhiletimer.count != 1 {
			t.Error(fmt.Sprintf("Failed e[%d] transitioned PPM to a back to same state %d", e, p.PimMachineFsm.Machine.Curr.CurrentState()))
			t.FailNow()
		}
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimDisabledStateRcvdMsg(t *testing.T) {
	testChan := make(chan string)
	p := UsedForTestOnlyPimTestSetup(t, false)

	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventRcvdMsg,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	UsedForTestOnlyPimCheckDisabled(p, t)

	if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateDisabled {
		t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimDisabledStateNotPortEnabledandInfoIsNotDisabled(t *testing.T) {
	testChan := make(chan string)

	p := UsedForTestOnlyPimTestSetup(t, false)

	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventNotPortEnabledInfoIsNotEqualDisabled,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	UsedForTestOnlyPimCheckDisabled(p, t)

	if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateDisabled {
		t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimDisabledStatePortEnabled(t *testing.T) {
	testChan := make(chan string)
	p := UsedForTestOnlyPimTestSetup(t, false)

	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventPortEnabled,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	UsedForTestOnlyPimCheckAgedState(p, t)

	if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateDisabled {
		t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimAgedStateInvalidStateTransitions(t *testing.T) {
	testChan := make(chan string)

	p := UsedForTestOnlyPimStartInAgedState(t)

	invalidStateMap := [11]fsm.Event{
		PimEventRcvdMsg,
		PimEventRcvdMsgAndNotUpdtInfo,
		PimEventPortEnabled,
		PimEventUnconditionalFallThrough,
		PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
		PimEventRcvdInfoEqualSuperiorDesignatedInfo,
		PimEventRcvdInfoEqualRepeatedDesignatedInfo,
		PimEventRcvdInfoEqualInferiorDesignatedInfo,
		PimEventRcvdInfoEqualInferiorRootAlternateInfo,
		PimEventRcvdInfoEqualOtherInfo,
	}

	// test the invalid states
	for _, e := range invalidStateMap {

		p.PimMachineFsm.PimEvents <- MachineEvent{e: e,
			src:          "TEST",
			responseChan: testChan}

		<-testChan
		if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateAged {
			t.Error(fmt.Sprintf("Failed e[%d] transitioned PIM to a new state state %s", e, PimStateStrMap[p.PimMachineFsm.Machine.Curr.CurrentState()]))
			t.FailNow()
		}
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimAgedStateSelectedAndUpdtInfo(t *testing.T) {
	testChan := make(chan string)

	p := UsedForTestOnlyPimStartInAgedState(t)
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDisabledPort)
	p.InfoIs = PortInfoStateReceived
	p.Selected = true
	p.UpdtInfo = true
	p.Synced = true
	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventSelectedAndUpdtInfo,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	UsedForTestOnlyPimCheckUpdateState(p, t)

	UsedForTestOnlyPimTestTeardown(p, t)
}

// Invalid test as we never stay in updated state, as it is a transition state
/*func xTestPimUpdateStateInvalidStateTransitions(t *testing.T) {
	testChan := make(chan string)

	p := UsedForTestOnlyPimStartInUpdateState(t)

	invalidStateMap := [11]fsm.Event{
		PimEventRcvdMsg,
		PimEventRcvdMsgAndNotUpdtInfo,
		PimEventPortEnabled,
		PimEventSelectedAndUpdtInfo,
		PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
		PimEventRcvdInfoEqualSuperiorDesignatedInfo,
		PimEventRcvdInfoEqualRepeatedDesignatedInfo,
		PimEventRcvdInfoEqualInferiorDesignatedInfo,
		PimEventRcvdInfoEqualInferiorRootAlternateInfo,
		PimEventRcvdInfoEqualOtherInfo,
	}

	// test the invalid states
	for _, e := range invalidStateMap {

		p.PimMachineFsm.PimEvents <- MachineEvent{e: e,
			src:          "TEST",
			responseChan: testChan}

		<-testChan
		if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateUpdate {
			t.Error(fmt.Sprintf("Failed e[%d] transitioned PIM to a new state state %d", e, p.PimMachineFsm.Machine.Curr.CurrentState()))
			t.FailNow()
		}
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

// Invalid test as we never stay in updated state, as it is a transition state
func xTestPimUpdateStateUnconditionalFallThrough(t *testing.T) {
	testChan := make(chan string)

	p := UsedForTestOnlyPimStartInUpdateState(t)

	p.PimMachineFsm.PimEvents <- MachineEvent{
		e:            PimEventUnconditionalFallThrough,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	if p.PimMachineFsm.Machine.Curr.PreviousState() == PimStateUpdate &&
		p.PimMachineFsm.Machine.Curr.CurrentState() == PimStateCurrent {
		t.Error("Failed to transition states appropiately")
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}
*/
func TestPimCurrentStateInvalidStateTransitions(t *testing.T) {
	testChan := make(chan string)

	p := UsedForTestOnlyPimStartInCurrentState(t)

	invalidStateMap := [9]fsm.Event{
		PimEventRcvdMsg,
		PimEventPortEnabled,
		PimEventUnconditionalFallThrough,
		PimEventRcvdInfoEqualSuperiorDesignatedInfo,
		PimEventRcvdInfoEqualRepeatedDesignatedInfo,
		PimEventRcvdInfoEqualInferiorDesignatedInfo,
		PimEventRcvdInfoEqualInferiorRootAlternateInfo,
		PimEventRcvdInfoEqualOtherInfo,
	}

	// test the invalid states
	for _, e := range invalidStateMap {
		// one way to validate that a transition did not occur is to make sure
		// that mdelayWhiletimer is not set to MigrateTime
		p.MdelayWhiletimer.count = 1

		p.PimMachineFsm.PimEvents <- MachineEvent{e: e,
			src:          "TEST",
			responseChan: testChan}

		<-testChan
		if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateCurrent {
			t.Error(fmt.Sprintf("Failed e[%d] transitioned PIM to a new state state %d", e, p.PimMachineFsm.Machine.Curr.CurrentState()))
			t.FailNow()
		}
		if p.MdelayWhiletimer.count != 1 {
			t.Error(fmt.Sprintf("Failed e[%d] transitioned PPM to a back to same state %d", e, p.PimMachineFsm.Machine.Curr.CurrentState()))
			t.FailNow()
		}
	}

	UsedForTestOnlyPimTestTeardown(p, t)
}

// TODO need to understand the vector lists better in order to test
func xTestPimCurrentStateRcvdMsgAndNotUpdtInfo(t *testing.T) {
	testChan := make(chan string)

	//Lets set the various msg and port priority vectors to determine which rcvdInfo should be set
	MsgVectorList := [4]PriorityVector{
		// less than
		PriorityVector{RootBridgeId: BridgeId{0x03, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			RootPathCost:       PortPathCost10Gb,
			DesignatedBridgeId: BridgeId{0x01, 0x00, 0x00, 0x22, 0x33, 0x44, 0x55, 0x66},
			DesignatedPortId:   200,
			BridgePortId:       200},
		// equal
		PriorityVector{RootBridgeId: BridgeId{0x02, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			RootPathCost:       PortPathCost10Gb,
			DesignatedBridgeId: BridgeId{0x01, 0x00, 0x00, 0x22, 0x33, 0x44, 0x55, 0x66},
			DesignatedPortId:   200,
			BridgePortId:       200},
		// greater than
		PriorityVector{RootBridgeId: BridgeId{0x01, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			RootPathCost:       PortPathCost10Gb,
			DesignatedBridgeId: BridgeId{0x01, 0x00, 0x00, 0x22, 0x33, 0x44, 0x55, 0x66},
			DesignatedPortId:   200,
			BridgePortId:       200},
	}
	PortVectorList := [4]PriorityVector{
		// less than
		PriorityVector{RootBridgeId: BridgeId{0x04, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			RootPathCost:       PortPathCost10Gb,
			DesignatedBridgeId: BridgeId{0x01, 0x00, 0x00, 0x22, 0x33, 0x44, 0x55, 0x66},
			DesignatedPortId:   200,
			BridgePortId:       200},
		// equal
		PriorityVector{RootBridgeId: BridgeId{0x02, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			RootPathCost:       PortPathCost10Gb,
			DesignatedBridgeId: BridgeId{0x01, 0x00, 0x00, 0x22, 0x33, 0x44, 0x55, 0x66},
			DesignatedPortId:   300,
			BridgePortId:       200},
		// greater than
		PriorityVector{RootBridgeId: BridgeId{0x00, 0x00, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			RootPathCost:       PortPathCost10Gb,
			DesignatedBridgeId: BridgeId{0x01, 0x00, 0x00, 0x22, 0x33, 0x44, 0x55, 0x66},
			DesignatedPortId:   200,
			BridgePortId:       200},
	}
	RoleList := [4]PortRole{
		PortRoleDesignatedPort, // less than
		PortRoleRootPort,       // less than
		PortRoleAlternatePort,
		PortRoleBackupPort,
	}

	// lets setup some pre-data for this test
	for _, r := range RoleList {
		for i, v := range MsgVectorList {
			p := UsedForTestOnlyPimStartInCurrentState(t)
			p.MsgPriority = v
			p.PortPriority = PortVectorList[i]
			p.RcvdMsg = true
			p.UpdtInfo = false
			p.Role = r

			rstp := layers.RSTP{
				ProtocolId:        layers.RSTPProtocolIdentifier,
				ProtocolVersionId: p.BridgeProtocolVersionGet(),
				BPDUType:          layers.BPDUTypeRSTP,
				Flags:             0,
				RootId:            p.PortPriority.RootBridgeId,
				RootPathCost:      uint32(p.PortPriority.RootPathCost),
				BridgeId:          p.PortPriority.DesignatedBridgeId,
				PortId:            uint16(p.PortPriority.DesignatedPortId),
				MsgAge:            uint16(p.PortTimes.MessageAge),
				MaxAge:            uint16(p.PortTimes.MaxAge),
				HelloTime:         uint16(p.PortTimes.HelloTime),
				FwdDelay:          uint16(p.PortTimes.ForwardingDelay),
				Version1Length:    0,
			}

			var flags uint8
			StpSetBpduFlags(ConvertBoolToUint8(false),
				ConvertBoolToUint8(p.Agree),
				ConvertBoolToUint8(p.Forward),
				ConvertBoolToUint8(p.Learning),
				ConvertRoleToPktRole(r),
				ConvertBoolToUint8(p.Proposed),
				ConvertBoolToUint8(p.TcWhileTimer.count != 0),
				&flags)

			rstp.Flags = layers.StpFlags(flags)

			p.PimMachineFsm.PimEvents <- MachineEvent{e: PimEventRcvdMsgAndNotUpdtInfo,
				src:          "TEST",
				data:         &rstp,
				responseChan: testChan}

			<-testChan
			if r == PortRoleDesignatedPort {
				switch i {
				case 0: // superior
					if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateSuperiorDesignated {
						t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
						t.FailNow()
					}
					if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateCurrent {
						t.Error("Failed Designated Port should have produced Current state")
						t.FailNow()
					}

					if p.Agreed != false {
						t.Error("Failed agreed is set")
					}
					if p.Proposing != false {
						t.Error("Failed proposiing is set")
					}
					if p.Agree != false {
						t.Error("Failed agree is set")
					}
					if p.InfoIs != PortInfoStateReceived {
						t.Error("Failed infoIs is not set to Received")
					}
					if p.Reselect != true {
						t.Error("Failed reselect is not set")
					}
					if p.Selected != false {
						t.Error("Failed selectd is set")
					}
					if p.RcvdMsg != false {
						t.Error("Failed rcvdMsg is set")
					}
				case 1: // repeated
					if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateRepeatedDesignated {
						t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))

						t.FailNow()
					}
					if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateCurrent {
						t.Error("Failed Designated Port should have produced Current state")
					}
					if p.RcvdMsg != false {
						t.Error("Failed rcvdMsg is set")
					}
				case 2: // inferior
					if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateInferiorDesignated {
						t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
					}
					if p.RcvdMsg != false {
						t.Error("Failed rcvdMsg is set")
					}
				}
			} else {
				switch i {
				case 0:
					if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateOther {
						t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
					}
					if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateCurrent {
						t.Error("Failed Designated Port should have produced Current state")
					}
					if p.RcvdMsg != false {
						t.Error("Failed rcvdMsg is set")
					}
				case 1, 2:
					if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateNotDesignated {
						t.Error(fmt.Sprintf("Failed Previous state not correct %s\n", PimStateStrMap[p.PimMachineFsm.Machine.Curr.PreviousState()]))
					}
					if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateCurrent {
						t.Error("Failed Designated Port should have produced Current state")
					}
					if p.RcvdMsg != false {
						t.Error("Failed rcvdMsg is set")
					}
				}
			}
			UsedForTestOnlyPimTestTeardown(p, t)
		}
	}
}

func TestPimCurrentStateSelectedAndUpdtInfo(t *testing.T) {
	testChan := make(chan string)
	p := UsedForTestOnlyPimStartInCurrentState(t)

	p.InfoIs = PortInfoStateReceived
	p.Synced = true
	p.Agreed = true
	p.Selected = true
	p.UpdtInfo = true
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDisabledPort)
	p.PimMachineFsm.PimEvents <- MachineEvent{e: PimEventSelectedAndUpdtInfo,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	UsedForTestOnlyPimCheckUpdateState(p, t)
	UsedForTestOnlyPimTestTeardown(p, t)
}

func TestPimCurrentStateInfoIsEqualReceivedAndrcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg(t *testing.T) {
	testChan := make(chan string)
	p := UsedForTestOnlyPimStartInCurrentState(t)

	// likely rcvdInfoWhile == 0 triggers this event
	p.InfoIs = PortInfoStateReceived
	p.RcvdInfoWhiletimer.count = 0
	p.UpdtInfo = false
	p.RcvdMsg = false
	p.PimMachineFsm.PimEvents <- MachineEvent{e: PimEventInflsEqualReceivedAndRcvdInfoWhileEqualZeroAndNotUpdtInfoAndNotRcvdMsg,
		src:          "TEST",
		responseChan: testChan,
	}
	<-testChan

	UsedForTestOnlyPimCheckAgedState(p, t)
	UsedForTestOnlyPimTestTeardown(p, t)
}

// TODO test Superior Designated
//func TestPimCurrentStateRcvdMsgAndNotUpdtInfo()
