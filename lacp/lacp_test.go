// lacp tests
package lacp

import (
	//	"fmt"
	"testing"
	"time"
)

func TestLaAggPortCreateWithoutKeySetWithAgg(t *testing.T) {

}

func TestLaAggPortCreateWithoutKeySetNoAgg(t *testing.T) {

}

func TestLaAggPortCreateAndBeginEvent(t *testing.T) {

	// must be called to initialize the global
	LacpSysGlobalInfoInit()

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		IntfId: "eth1.1",
		Key:    100,
	}

	p := NewLaAggPort(pconf)

	// lets start all the state machines
	p.BEGIN(false)

	//	fmt.Println("Rx:", p.RxMachineFsm.Machine.Curr.CurrentState(),
	//		"Ptx:", p.PtxMachineFsm.Machine.Curr.CurrentState(),
	//		"Cd:", p.CdMachineFsm.Machine.Curr.CurrentState(),
	//		"Mux:", p.MuxMachineFsm.Machine.Curr.CurrentState(),
	//		"Tx:", p.TxMachineFsm.Machine.Curr.CurrentState())

	// lets test the states, after initialization port moves to Disabled State
	// Rx Machine
	if p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected",
			LacpRxmStatePortDisabled, "actual",
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}
	// Periodic Tx Machine
	if p.PtxMachineFsm.Machine.Curr.CurrentState() != LacpPtxmStateNoPeriodic {
		t.Error("ERROR PTX Machine state incorrect expected",
			LacpPtxmStateNoPeriodic, "actual",
			p.PtxMachineFsm.Machine.Curr.CurrentState())
	}
	// Churn Detection Machine
	if p.CdMachineFsm.Machine.Curr.CurrentState() != LacpCdmStateActorChurnMonitor {
		t.Error("ERROR CD Machine state incorrect expected",
			LacpCdmStateActorChurnMonitor, "actual",
			p.CdMachineFsm.Machine.Curr.CurrentState())
	}
	// Mux Machine
	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDetached {
		t.Error("ERROR MUX Machine state incorrect expected",
			LacpMuxmStateDetached, "actual",
			p.MuxMachineFsm.Machine.Curr.CurrentState())
	}
	// Tx Machine
	if p.TxMachineFsm.Machine.Curr.CurrentState() != LacpTxmStateOff {
		t.Error("ERROR TX Machine state incorrect expected",
			LacpTxmStateOff, "actual",
			p.TxMachineFsm.Machine.Curr.CurrentState())
	}
}

func TestLaAggPortRxMachineStateTransitions(t *testing.T) {

	var msg string
	var portchan chan string
	// must be called to initialize the global
	LacpSysGlobalInfoInit()

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		IntfId: "eth1.1",
		Key:    100,
	}

	p := NewLaAggPort(pconf)

	// lets start the Rx Machine only
	p.LacpRxMachineMain()

	// Rx Machine
	if p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateNone {
		t.Error("ERROR RX Machine state incorrect expected",
			LacpRxmStateNone, "actual",
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	portchan = p.PortChannelGet()
	// send event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventBegin,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port is initally disabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateInitialize &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateInitialize,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// check state info
	if p.aggSelected != LacpAggUnSelected {
		t.Error("expected UNSELECTED", LacpAggUnSelected, "actual", p.aggSelected)
	}
	if LacpStateIsSet(p.actorOper.state, LacpStateExpiredBit) {
		t.Error("expected state Expired to be cleared")
	}
	if p.portMoved != false {
		t.Error("expected port moved to be false")
	}
	// TODO check actor oper state

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

	p.portMoved = true
	// PORT MOVED
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateInitialize &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateInitialize,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.portMoved = false
	p.portEnabled = true
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStatePortDisabled,
			LacpRxmStateExpired,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	if LacpStateIsSet(p.partnerOper.state, LacpStateSyncBit) {
		t.Error("Expected partner Sync Bit to not be set")
	}
	if !LacpStateIsSet(p.partnerOper.state, LacpStateTimeoutBit) {
		t.Error("Expected partner Timeout bit to be set since we are in short timeout")
	}
	if p.RxMachineFsm.currentWhileTimerTimeout != LacpShortTimeoutTime {
		t.Error("Expected timer to be set to short timeout")
	}
	if !LacpStateIsSet(p.actorOper.state, LacpStateExpiredBit) {
		t.Error("Expected actor expired bit to be set")
	}

	p.portEnabled = false
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	if LacpStateIsSet(p.partnerOper.state, LacpStateSyncBit) {
		t.Error("Expected partner Sync Bit to not be set")
	}

	p.portEnabled = true
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

	p.portEnabled = false
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateLacpDisabled,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.portEnabled = true
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateDefaulted,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// TODO check default selected, record default, expired == false

	// LETS GET THE STATE BACK TO EXPIRED

	p.portEnabled = false
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

	p.portEnabled = true
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

	// lets adjust the actorOper timeout state
	// TODO Assume a method was called to adjust this
	LacpStateSet(&p.actorAdmin.state, LacpStateTimeoutBit)
	LacpStateSet(&p.actorOper.state, LacpStateTimeoutBit)

	// send valid pdu
	lacppdu := &LacpPdu{
		subType: LacpSubType,
		version: 1,
		actor: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpPortSystemInfo{systemId: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					systemPriority: 1},
				key:      100,
				port_pri: 0x80,
				port:     10,
				state:    LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		partner: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpPortSystemInfo{systemId: p.actorOper.system.systemId,
					systemPriority: p.actorOper.system.systemPriority},
				key:      p.key,
				port_pri: p.portPriority,
				port:     p.portNum,
				state:    p.actorOper.state},
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateDefaulted,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// send valid pdu
	lacppdu = &LacpPdu{
		subType: LacpSubType,
		version: 1,
		actor: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpPortSystemInfo{systemId: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					systemPriority: 1},
				key:      100,
				port_pri: 0x80,
				port:     10,
				state:    LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		partner: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpPortSystemInfo{systemId: p.actorOper.system.systemId,
					systemPriority: p.actorOper.system.systemPriority},
				key:      p.key,
				port_pri: p.portPriority,
				port:     p.portNum,
				state:    p.actorOper.state},
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// send valid pdu
	lacppdu = &LacpPdu{
		subType: LacpSubType,
		version: 1,
		actor: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpPortSystemInfo{systemId: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					systemPriority: 1},
				key:      100,
				port_pri: 0x80,
				port:     10,
				state:    LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		partner: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpPortSystemInfo{systemId: p.actorOper.system.systemId,
					systemPriority: p.actorOper.system.systemPriority},
				key:      p.key,
				port_pri: p.portPriority,
				port:     p.portNum,
				state:    p.actorOper.state},
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
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

}

// TODO add more tests
// 1) invalid events on stats
// 2) pkt events
