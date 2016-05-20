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

// lacp tests
// go test
// go test -coverageprofile lacpcov.out
// go tool cover -html=lacpcov.out
package lacp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"testing"
	"time"
	"utils/fsm"
)

func InvalidStateCheck(p *LaAggPort, invalidStates []fsm.Event, prevState fsm.State, currState fsm.State) (string, bool) {

	var s string
	rc := true

	portchan := p.PortChannelGet()

	// force what State transition should have been
	p.RxMachineFsm.Machine.Curr.SetState(prevState)
	p.RxMachineFsm.Machine.Curr.SetState(currState)

	for _, e := range invalidStates {
		// send PORT MOVED event to Rx Machine
		p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
			e:            e,
			responseChan: portchan,
			src:          "TEST"}

		// wait for response
		if msg := <-portchan; msg != RxMachineModuleStr {
			s = fmt.Sprintf("Expected response from", RxMachineModuleStr)
			rc = false
			return s, rc
		}

		// PORT MOVED
		if p.RxMachineFsm.Machine.Curr.PreviousState() != prevState &&
			p.RxMachineFsm.Machine.Curr.CurrentState() != currState {
			s = fmt.Sprintf("ERROR RX Machine State incorrect expected (prev/curr)",
				prevState,
				currState,
				"actual",
				p.RxMachineFsm.Machine.Curr.PreviousState(),
				p.RxMachineFsm.Machine.Curr.CurrentState())
			rc = false
			return s, rc
		}
	}

	return "", rc
}

func TestLaAggPortCreateAndBeginEvent(t *testing.T) {

	var p *LaAggPort

	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		Key:    100,
		AggId:  2000,
		Enable: false,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   "SIMeth1.1",
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {

		//	fmt.Println("Rx:", p.RxMachineFsm.Machine.Curr.CurrentState(),
		//		"Ptx:", p.PtxMachineFsm.Machine.Curr.CurrentState(),
		//		"Cd:", p.CdMachineFsm.Machine.Curr.CurrentState(),
		//		"Mux:", p.MuxMachineFsm.Machine.Curr.CurrentState(),
		//		"Tx:", p.TxMachineFsm.Machine.Curr.CurrentState())

		// lets test the States, after initialization port moves to Disabled State
		// Rx Machine
		if p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
			t.Error("ERROR RX Machine State incorrect expected",
				LacpRxmStatePortDisabled, "actual",
				p.RxMachineFsm.Machine.Curr.CurrentState())
		}
		// Periodic Tx Machine
		if p.PtxMachineFsm.Machine.Curr.CurrentState() != LacpPtxmStateNoPeriodic {
			t.Error("ERROR PTX Machine State incorrect expected",
				LacpPtxmStateNoPeriodic, "actual",
				p.PtxMachineFsm.Machine.Curr.CurrentState())
		}
		// Churn Detection Machine
		if p.CdMachineFsm.Machine.Curr.CurrentState() != LacpCdmStateActorChurnMonitor {
			t.Error("ERROR CD Machine State incorrect expected",
				LacpCdmStateActorChurnMonitor, "actual",
				p.CdMachineFsm.Machine.Curr.CurrentState())
		}
		// Mux Machine
		if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDetached {
			t.Error("ERROR MUX Machine State incorrect expected",
				LacpMuxmStateDetached, "actual",
				p.MuxMachineFsm.Machine.Curr.CurrentState())
		}
		// Tx Machine
		if p.TxMachineFsm.Machine.Curr.CurrentState() != LacpTxmStateOff {
			t.Error("ERROR TX Machine State incorrect expected",
				LacpTxmStateOff, "actual",
				p.TxMachineFsm.Machine.Curr.CurrentState())
		}
	}
	DeleteLaAggPort(pconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

func TestLaAggPortCreateWithInvalidKeySetWithAgg(t *testing.T) {
	var p *LaAggPort

	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}

	LacpSysGlobalInfoInit(sysId)

	aconf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:  2000,
		Key: 50,
		Lacp: LacpConfigInfo{Interval: LacpSlowPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:01:02:03:04:05",
			SystemPriority: 128},
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	pconf := &LaAggPortConfig{
		Id:     2,
		Prio:   0x80,
		Key:    100, // INVALID
		AggId:  2000,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, 0x02, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   "SIMeth1.1",
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	// Delete the port and agg
	DeleteLaAggPort(pconf.Id)
	DeleteLaAgg(aconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.SysKey, sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.SysKey, sgi.PortList, sgi.PortMap)
		}
	}
}

func TestLaAggPortCreateWithoutKeySetNoAgg(t *testing.T) {

	var p *LaAggPort
	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     3,
		Prio:   0x80,
		Key:    100,
		AggId:  2000,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   "SIMeth1.1",
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	// Delete port
	DeleteLaAggPort(pconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

func TestLaAggPortCreateThenCorrectAggCreate(t *testing.T) {

	var p *LaAggPort
	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     3,
		Prio:   0x80,
		Key:    100,
		AggId:  2000,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   "SIMeth1.1",
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode should be UNSELECTED")
		}
	}

	aconf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:  2000,
		Key: 100,
		Lacp: LacpConfigInfo{Interval: LacpSlowPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:01:02:03:04:05",
			SystemPriority: 128},
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if p.aggSelected != LacpAggSelected {
		t.Error("Port is in SELECTED mode (2)")
	}

	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateAttached {
		t.Error("Mux State expected", LacpMuxmStateAttached, "actual", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}

	// TODO Check States of other State machines

	// Delete agg
	DeleteLaAggPort(pconf.Id)
	DeleteLaAgg(aconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

// TestLaAggPortCreateThenCorrectAggCreateThenDetach:
// - create port
// - create lag
// - attach port
// - enable port
func TestLaAggPortCreateThenCorrectAggCreateThenDetach(t *testing.T) {

	var p *LaAggPort
	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:    3,
		Prio:  0x80,
		Key:   100,
		AggId: 2000,
		Mode:  LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   "SIMeth1.1",
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode 1")
		}
	}

	aconf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:  2000,
		Key: 100,
		Lacp: LacpConfigInfo{Interval: LacpSlowPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:01:02:03:04:05",
			SystemPriority: 128},
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create should be disabled since this
	// is the initial state of port config
	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode 2 mux state", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}

	EnableLaAggPort(pconf.Id)

	if p.aggSelected != LacpAggSelected {
		t.Error("Port is NOT in SELECTED mode 3")
	}

	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateAttached {
		t.Error("Mux State expected", LacpMuxmStateAttached, "actual", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}
	// Delete port
	DeleteLaAggPortFromAgg(pconf.Key, pconf.Id)
	DeleteLaAggPort(pconf.Id)
	DeleteLaAgg(aconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

// Enable port post creation
func TestLaAggPortEnable(t *testing.T) {
	var p *LaAggPort
	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:    3,
		Prio:  0x80,
		Key:   100,
		AggId: 2000,
		Mode:  LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   "SIMeth1.1",
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial State after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	aconf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:  2000,
		Key: 100,
		Lacp: LacpConfigInfo{Interval: LacpSlowPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:01:02:03:04:05",
			SystemPriority: 128},
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	EnableLaAggPort(pconf.Id)

	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateAttached {
		t.Error("Mux State expected", LacpMuxmStateAttached, "actual", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}

	if p.aggSelected != LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	// Delete port
	DeleteLaAggPort(pconf.Id)
	DeleteLaAgg(aconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

func TestLaAggPortRxMachineStateTransitions(t *testing.T) {

	var msg string
	var portchan chan string
	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		IntfId: "SIMeth1.1",
		Key:    100,
	}

	// not calling Create because we don't want to launch all State machines
	p := NewLaAggPort(pconf)

	// lets start the Rx Machine only
	p.LacpRxMachineMain()

	// Rx Machine
	if p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateNone {
		t.Error("ERROR RX Machine State incorrect expected",
			LacpRxmStateNone, "actual",
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.BEGIN(false)
	portchan = p.PortChannelGet()

	// port is initally disabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateInitialize &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateInitialize,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// check State info
	if p.aggSelected != LacpAggUnSelected {
		t.Error("expected UNSELECTED", LacpAggUnSelected, "actual", p.aggSelected)
	}
	if LacpStateIsSet(p.ActorOper.State, LacpStateExpiredBit) {
		t.Error("expected State Expired to be cleared")
	}
	if p.portMoved != false {
		t.Error("expected port moved to be false")
	}
	// TODO check actor oper State

	p.portMoved = true
	// send PORT MOVED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// PORT MOVED
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateInitialize &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateInitialize,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.aggSelected = LacpAggSelected
	p.portMoved = false
	p.PortEnabled = true
	p.lacpEnabled = false
	// send PORT ENABLED && LACP DISABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpDisabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port is initally disabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStatePortDisabled &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateLacpDisabled {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStatePortDisabled,
			LacpRxmStateLacpDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.lacpEnabled = true
	// send LACP ENABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventLacpEnabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was lacp disabled, but then transitioned to port disabled
	// then expired
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStatePortDisabled &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateExpired {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStatePortDisabled,
			LacpRxmStateExpired,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	if LacpStateIsSet(p.PartnerOper.State, LacpStateSyncBit) {
		t.Error("Expected partner Sync Bit to not be set")
	}
	if !LacpStateIsSet(p.PartnerOper.State, LacpStateTimeoutBit) {
		t.Error("Expected partner Timeout bit to be set since we are in short timeout")
	}
	if p.RxMachineFsm.currentWhileTimerTimeout != LacpShortTimeoutTime {
		t.Error("Expected timer to be set to short timeout")
	}
	if !LacpStateIsSet(p.ActorOper.State, LacpStateExpiredBit) {
		t.Error("Expected actor expired bit to be set")
	}

	p.PortEnabled = false
	// send NOT ENABLED AND NOT MOVED event to Rx Machine from Expired State
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventNotPortEnabledAndNotPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	if LacpStateIsSet(p.PartnerOper.State, LacpStateSyncBit) {
		t.Error("Expected partner Sync Bit to not be set")
	}

	p.PortEnabled = true
	p.lacpEnabled = false
	// send NOT ENABLED AND NOT MOVED event to Rx Machine from Expired State
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpDisabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	p.PortEnabled = false
	// send NOT ENABLED AND NOT MOVED event to Rx Machine from LACP DISABLED
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventNotPortEnabledAndNotPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateLacpDisabled &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateLacpDisabled,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.PortEnabled = true
	p.lacpEnabled = true
	// send PORT ENABLE LACP ENABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpEnabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// send CURRENT WHILE TIMER event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventCurrentWhileTimerExpired,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateDefaulted {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateDefaulted,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// TODO check default selected, record default, expired == false

	// LETS GET THE State BACK TO EXPIRED

	p.PortEnabled = false
	// send NOT PORT ENABLE NOT PORT MOVED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventNotPortEnabledAndNotPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	p.PortEnabled = true
	// send PORT ENABLE LACP ENABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpEnabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// lets adjust the ActorOper timeout State
	// TODO Assume a method was called to adjust this
	LacpStateSet(&p.actorAdmin.State, LacpStateTimeoutBit)
	LacpStateSet(&p.ActorOper.State, LacpStateTimeoutBit)

	//slow := &layers.SlowProtocol{
	//	SubType: layers.SlowProtocolTypeLACP,
	//}
	// send valid pdu
	lacppdu := &layers.LACP{
		Version: layers.LACPVersion2,
		Actor: layers.LACPInfoTlv{TlvType: layers.LACPTLVActorInfo,
			Length: layers.LACPActorTlvLength,
			Info: layers.LACPPortInfo{
				System: layers.LACPSystem{SystemId: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					SystemPriority: 1},
				Key:     100,
				PortPri: 0x80,
				Port:    10,
				State:   LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		Partner: layers.LACPInfoTlv{TlvType: layers.LACPTLVPartnerInfo,
			Length: layers.LACPPartnerTlvLength,
			Info: layers.LACPPortInfo{
				System: layers.LACPSystem{SystemId: p.ActorOper.System.actor_System,
					SystemPriority: p.ActorOper.System.Actor_System_priority},
				Key:     p.Key,
				PortPri: p.portPriority,
				Port:    p.PortNum,
				State:   p.ActorOper.State},
		},
	}

	rx := LacpRxLacpPdu{
		pdu:          lacppdu,
		responseChan: portchan,
		src:          "TEST"}
	p.RxMachineFsm.RxmPktRxEvent <- rx

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateCurrent {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// allow for current while timer to expire
	time.Sleep(time.Second * 4)

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateCurrent &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateExpired {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateCurrent,
			LacpRxmStateExpired,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// allow for current while timer to expire
	time.Sleep(time.Second * 4)

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateDefaulted {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateDefaulted,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	lacppdu = &layers.LACP{
		Version: layers.LACPVersion2,
		Actor: layers.LACPInfoTlv{TlvType: layers.LACPTLVActorInfo,
			Length: layers.LACPActorTlvLength,
			Info: layers.LACPPortInfo{
				System: layers.LACPSystem{SystemId: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					SystemPriority: 1},
				Key:     100,
				PortPri: 0x80,
				Port:    10,
				State:   LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		Partner: layers.LACPInfoTlv{TlvType: layers.LACPTLVPartnerInfo,
			Length: layers.LACPPartnerTlvLength,
			Info: layers.LACPPortInfo{
				System: layers.LACPSystem{SystemId: p.ActorOper.System.actor_System,
					SystemPriority: p.ActorOper.System.Actor_System_priority},
				Key:     p.Key,
				PortPri: p.portPriority,
				Port:    p.PortNum,
				State:   p.ActorOper.State},
		},
	}

	rx = LacpRxLacpPdu{
		pdu:          lacppdu,
		responseChan: portchan,
		src:          "TEST"}
	p.RxMachineFsm.RxmPktRxEvent <- rx

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateDefaulted &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateCurrent {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	lacppdu = &layers.LACP{
		Version: layers.LACPVersion2,
		Actor: layers.LACPInfoTlv{TlvType: layers.LACPTLVActorInfo,
			Length: layers.LACPActorTlvLength,
			Info: layers.LACPPortInfo{
				System: layers.LACPSystem{SystemId: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					SystemPriority: 1},
				Key:     100,
				PortPri: 0x80,
				Port:    10,
				State:   LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		Partner: layers.LACPInfoTlv{TlvType: layers.LACPTLVPartnerInfo,
			Length: layers.LACPPartnerTlvLength,
			Info: layers.LACPPortInfo{
				System: layers.LACPSystem{SystemId: p.ActorOper.System.actor_System,
					SystemPriority: p.ActorOper.System.Actor_System_priority},
				Key:     p.Key,
				PortPri: p.portPriority,
				Port:    p.PortNum,
				State:   p.ActorOper.State},
		},
	}

	rx = LacpRxLacpPdu{
		pdu:          lacppdu,
		responseChan: portchan,
		src:          "TEST"}
	p.RxMachineFsm.RxmPktRxEvent <- rx

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateCurrent &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateCurrent {
		t.Error("ERROR RX Machine State incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	DeleteLaAggPort(pconf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

func TestLaAggPortRxMachineInvalidStateTransitions(t *testing.T) {

	// must be called to initialize the global
	// must be called to initialize the global
	sysId := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		IntfId: "SIMeth1.1",
		Key:    100,
	}

	// not calling Create because we don't want to launch all State machines
	p := NewLaAggPort(pconf)

	// lets start the Rx Machine only
	p.LacpRxMachineMain()

	p.BEGIN(false)

	// turn timer off so that we do not accidentally transition States
	p.RxMachineFsm.CurrentWhileTimerStop()
	/*
		LacpRxmEventBegin = iota + 1
		LacpRxmEventUnconditionalFallthrough
		LacpRxmEventNotPortEnabledAndNotPortMoved
		LacpRxmEventPortMoved
		LacpRxmEventPortEnabledAndLacpEnabled
		LacpRxmEventPortEnabledAndLacpDisabled
		LacpRxmEventCurrentWhileTimerExpired
		LacpRxmEventLacpEnabled
		LacpRxmEventLacpPktRx
		LacpRxmEventKillSignal
	*/

	// BEGIN -> INITIALIZE automatically falls through to PORT_DISABLED so no
	// need to tests

	// PORT_DISABLED
	portDisableInvalidStates := [4]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventCurrentWhileTimerExpired,
		LacpRxmEventLacpEnabled,
		LacpRxmEventLacpPktRx}

	str, ok := InvalidStateCheck(p, portDisableInvalidStates[:], LacpRxmStateInitialize, LacpRxmStatePortDisabled)
	if !ok {
		t.Error(str)
	}

	// EXPIRED - note disabling current while timer so State does not change
	expiredInvalidStates := [5]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventLacpEnabled}

	str, ok = InvalidStateCheck(p, expiredInvalidStates[:], LacpRxmStatePortDisabled, LacpRxmStateExpired)
	if !ok {
		t.Error(str)
	}

	// LACP_DISABLED
	lacpDisabledInvalidStates := [6]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventCurrentWhileTimerExpired,
		LacpRxmEventLacpPktRx}

	str, ok = InvalidStateCheck(p, lacpDisabledInvalidStates[:], LacpRxmStatePortDisabled, LacpRxmStateLacpDisabled)
	if !ok {
		t.Error(str)
	}

	// DEFAULTED
	defaultedInvalidStates := [6]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventCurrentWhileTimerExpired,
		LacpRxmEventLacpEnabled}

	str, ok = InvalidStateCheck(p, defaultedInvalidStates[:], LacpRxmStateExpired, LacpRxmStateDefaulted)
	if !ok {
		t.Error(str)
	}

	// DEFAULTED
	currentInvalidStates := [5]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventLacpEnabled}

	str, ok = InvalidStateCheck(p, currentInvalidStates[:], LacpRxmStateExpired, LacpRxmStateCurrent)
	if !ok {
		t.Error(str)
	}

	p.LaAggPortDelete()
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

func TestTwoAggsBackToBackSinglePort(t *testing.T) {

	const LaAggPortActor = 10
	const LaAggPortPeer = 20
	LaAggPortActorIf := "SIMeth0"
	LaAggPortPeerIf := "SIM2eth0"
	// must be called to initialize the global
	LaSystemActor := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x64}}
	LaSystemPeer := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0xC8}}

	bridge := SimulationBridge{
		port1:       LaAggPortActor,
		port2:       LaAggPortPeer,
		rxLacpPort1: make(chan gopacket.Packet, 10),
		rxLacpPort2: make(chan gopacket.Packet, 10),
	}

	ActorSystem := LacpSysGlobalInfoInit(LaSystemActor)
	PeerSystem := LacpSysGlobalInfoInit(LaSystemPeer)
	ActorSystem.LaSysGlobalRegisterTxCallback(LaAggPortActorIf, bridge.TxViaGoChannel)
	PeerSystem.LaSysGlobalRegisterTxCallback(LaAggPortPeerIf, bridge.TxViaGoChannel)

	p1conf := &LaAggPortConfig{
		Id:     LaAggPortActor,
		Prio:   0x80,
		Key:    100,
		AggId:  100,
		Enable: true,
		Mode:   LacpModeActive,
		//Timeout: LacpFastPeriodicTime,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPortActor, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortActorIf,
		TraceEna: false,
	}

	p2conf := &LaAggPortConfig{
		Id:     LaAggPortPeer,
		Prio:   0x80,
		Key:    200,
		AggId:  200,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPortPeer, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortPeerIf,
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(p1conf)
	CreateLaAggPort(p2conf)

	// port 1
	LaRxMain(bridge.port1, bridge.rxLacpPort1)
	// port 2
	LaRxMain(bridge.port2, bridge.rxLacpPort2)

	a1conf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x01, 0x01, 0x01},
		Id:  100,
		Key: 100,
		Lacp: LacpConfigInfo{Interval: LacpSlowPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:00:00:00:00:64",
			SystemPriority: 128},
	}

	a2conf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x02, 0x02, 0x02, 0x02},
		Id:  200,
		Key: 200,
		Lacp: LacpConfigInfo{Interval: LacpSlowPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:00:00:00:00:C8",
			SystemPriority: 128},
	}

	// Create Aggregation
	CreateLaAgg(a1conf)
	CreateLaAgg(a2conf)

	// Add port to agg
	//AddLaAggPortToAgg(a1conf.Id, p1conf.Id)
	//AddLaAggPortToAgg(a2conf.Id, p2conf.Id)

	//time.Sleep(time.Second * 30)
	testWait := make(chan bool)

	var p1 *LaAggPort
	var p2 *LaAggPort
	if LaFindPortById(p1conf.Id, &p1) &&
		LaFindPortById(p2conf.Id, &p2) {

		go func() {
			for i := 0; i < 10 &&
				(p1.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDistributing ||
					p2.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDistributing); i++ {
				time.Sleep(time.Second * 1)
			}
			testWait <- true
		}()

		<-testWait
		close(testWait)

		State1 := GetLaAggPortActorOperState(p1conf.Id)
		State2 := GetLaAggPortActorOperState(p2conf.Id)

		const portUpState = LacpStateActivityBit | LacpStateAggregationBit |
			LacpStateSyncBit | LacpStateCollectingBit | LacpStateDistributingBit

		if !LacpStateIsSet(State1, portUpState) {
			t.Error(fmt.Sprintf("Actor Port State 0x%x did not come up properly with peer expected 0x%x", State1, portUpState))
		}
		if !LacpStateIsSet(State2, portUpState) {
			t.Error(fmt.Sprintf("Peer Port State 0x%x did not come up properly with actor expected 0x%x", State2, portUpState))
		}

		// TODO check the States of the other State machines
	} else {
		t.Error("Unable to find port just created")
	}

	bridge.rxLacpPort1 = nil
	bridge.rxLacpPort2 = nil
	// cleanup the provisioning
	DeleteLaAgg(a1conf.Id)
	DeleteLaAgg(a2conf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

// TestTwoAggsBackToBackSinglePortTimeout will allow for
// two ports to sync up then force a timeout by disabling
// one end of the connection by setting the mode to "ON"
func xTestTwoAggsBackToBackSinglePortTimeout(t *testing.T) {

	const LaAggPortActor = 11
	const LaAggPortPeer = 21
	const LaAggPortActorIf = "SIMeth0"
	const LaAggPortPeerIf = "SIMeth1"
	// must be called to initialize the global
	LaSystemActor := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x64}}
	LaSystemPeer := LacpSystem{Actor_System_priority: 128,
		actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0xC8}}

	bridge := SimulationBridge{
		port1:       LaAggPortActor,
		port2:       LaAggPortPeer,
		rxLacpPort1: make(chan gopacket.Packet),
		rxLacpPort2: make(chan gopacket.Packet),
	}

	ActorSystem := LacpSysGlobalInfoInit(LaSystemActor)
	PeerSystem := LacpSysGlobalInfoInit(LaSystemPeer)
	ActorSystem.LaSysGlobalRegisterTxCallback(LaAggPortActorIf, bridge.TxViaGoChannel)
	PeerSystem.LaSysGlobalRegisterTxCallback(LaAggPortPeerIf, bridge.TxViaGoChannel)

	// port 1
	go LaRxMain(bridge.port1, bridge.rxLacpPort1)
	// port 2
	go LaRxMain(bridge.port2, bridge.rxLacpPort2)

	p1conf := &LaAggPortConfig{
		Id:      LaAggPortActor,
		Prio:    0x80,
		Key:     100,
		AggId:   100,
		Enable:  true,
		Mode:    LacpModeActive,
		Timeout: LacpShortTimeoutTime,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPortActor, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortActorIf,
		TraceEna: true,
	}

	p2conf := &LaAggPortConfig{
		Id:      LaAggPortPeer,
		Prio:    0x80,
		Key:     200,
		AggId:   200,
		Enable:  true,
		Mode:    LacpModeActive,
		Timeout: LacpShortTimeoutTime,
		Properties: PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPortPeer, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortPeerIf,
		TraceEna: false,
	}

	// lets create a port and start the machines
	CreateLaAggPort(p1conf)
	CreateLaAggPort(p2conf)

	a1conf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x01, 0x01, 0x01},
		Id:  100,
		Key: 100,
		Lacp: LacpConfigInfo{Interval: LacpFastPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:00:00:00:00:64",
			SystemPriority: 128},
	}

	a2conf := &LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x02, 0x02, 0x02, 0x02},
		Id:  200,
		Key: 200,
		Lacp: LacpConfigInfo{Interval: LacpFastPeriodicTime,
			Mode:           LacpModeActive,
			SystemIdMac:    "00:00:00:00:00:C8",
			SystemPriority: 128},
	}

	// Create Aggregation
	CreateLaAgg(a1conf)
	CreateLaAgg(a2conf)

	// Add port to agg
	AddLaAggPortToAgg(a1conf.Key, p1conf.Id)
	AddLaAggPortToAgg(a2conf.Key, p2conf.Id)

	//time.Sleep(time.Second * 30)
	testWait := make(chan bool)

	var p1 *LaAggPort
	var p2 *LaAggPort
	if LaFindPortById(p1conf.Id, &p1) &&
		LaFindPortById(p2conf.Id, &p2) {

		go func() {
			for i := 0; i < 10 &&
				(p1.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDistributing ||
					p2.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDistributing); i++ {
				time.Sleep(time.Second * 1)
			}
			testWait <- true
		}()

		<-testWait

		State1 := GetLaAggPortActorOperState(p1conf.Id)
		State2 := GetLaAggPortActorOperState(p2conf.Id)

		const portUpState = LacpStateActivityBit | LacpStateAggregationBit |
			LacpStateSyncBit | LacpStateCollectingBit | LacpStateDistributingBit

		if !LacpStateIsSet(State1, portUpState) {
			t.Error(fmt.Sprintf("Actor Port State 0x%x did not come up properly with peer expected 0x%x", State1, portUpState))
		}
		if !LacpStateIsSet(State2, portUpState) {
			t.Error(fmt.Sprintf("Peer Port State 0x%x did not come up properly with actor expected 0x%x", State2, portUpState))
		}

		// TODO check the States of the other State machines

		// Lets disable lacp for p1
		SetLaAggPortLacpMode(p1conf.Id, LacpModeOn)

		go func() {
			var i int
			for i = 0; i < 10 &&
				(p1.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateDefaulted ||
					p1.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDetached ||
					p2.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateDefaulted ||
					p2.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDetached); i++ {

				time.Sleep(time.Second * 1)
			}

			if i == 10 {
				testWait <- false
			} else {
				testWait <- true
			}

		}()

		testResult := <-testWait
		if !testResult {
			t.Error(fmt.Sprintf("Actor and Peer States are not correct Expected P1 RXM/MUX",
				LacpRxmStateLacpDisabled, LacpMuxmStateDetached, "Actual", p1.RxMachineFsm.Machine.Curr.CurrentState(),
				p1.MuxMachineFsm.Machine.Curr.CurrentState(), "Expected P2 RXM/MUX", LacpRxmStateDefaulted,
				LacpMuxmStateDetached, "Actual", p2.RxMachineFsm.Machine.Curr.CurrentState(),
				p2.MuxMachineFsm.Machine.Curr.CurrentState()))
		}

		// TODO check the States of the other State machines
	} else {
		t.Error("Unable to find port just created")
	}
	// cleanup the provisioning
	DeleteLaAgg(a1conf.Id)
	DeleteLaAgg(a2conf.Id)
	for _, sgi := range LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
}

//
func xTestLaAggPortPeriodicTxMachineStateTransitions(t *testing.T) {

}

func xTestLaAggPortPeriodicTxMachineInvalidStateTransitions(t *testing.T) {

}

func xTestLaAggPortMuxMachineStateTransitions(t *testing.T) {

}

func xTestLaAggPortMuxMachineInvalidStateTransitions(t *testing.T) {

}

func xTestLaAggPortChurnDetectionMachineStateTransitions(t *testing.T) {

}

func xTestLaAggPortChurnDetectionMachineInvalidStateTransitions(t *testing.T) {

}

func xTestLaAggPortTxMachineStateTransitions(t *testing.T) {

}

func xTestLaAggPortTxMachineInvalidStateTransitions(t *testing.T) {

}

// TODO add more tests
// 1) invalid events on stats
// 2) pkt events
