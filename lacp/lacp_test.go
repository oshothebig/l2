// test_lacp
package lacp_test

import (
	//"fmt"
	"l2/lacp"
	"testing"
	//	"time"
)

func TestLaAggPortCreateAndBeginEvent(t *testing.T) {

	// must be called to initialize the global
	lacp.LacpSysGlobalInfoInit()

	p := lacp.NewLaAggPort(1, 0x80, "eth1.1")

	// lets start all the state machines
	p.BEGIN(false)

	/*
		fmt.Println("Rx:", p.RxMachineFsm.Machine.Curr.CurrentState(),
			"Ptx:", p.PtxMachineFsm.Machine.Curr.CurrentState(),
			"Cd:", p.CdMachineFsm.Machine.Curr.CurrentState(),
			"Mux:", p.MuxMachineFsm.Machine.Curr.CurrentState(),
			"Tx:", p.TxMachineFsm.Machine.Curr.CurrentState())
	*/
	// lets test the states, after initialization port moves to Disabled State
	// Rx Machine
	if p.RxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected",
			lacp.LacpRxmStatePortDisabled, "actual",
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}
	// Periodic Tx Machine
	if p.PtxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpPtxmStateNoPeriodic {
		t.Error("ERROR PTX Machine state incorrect expected",
			lacp.LacpPtxmStateNoPeriodic, "actual",
			p.PtxMachineFsm.Machine.Curr.CurrentState())
	}
	// Churn Detection Machine
	if p.CdMachineFsm.Machine.Curr.CurrentState() != lacp.LacpCdmStateActorChurnMonitor {
		t.Error("ERROR CD Machine state incorrect expected",
			lacp.LacpCdmStateActorChurnMonitor, "actual",
			p.CdMachineFsm.Machine.Curr.CurrentState())
	}
	// Mux Machine
	if p.MuxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpMuxmStateDetached {
		t.Error("ERROR MUX Machine state incorrect expected",
			lacp.LacpMuxmStateDetached, "actual",
			p.MuxMachineFsm.Machine.Curr.CurrentState())
	}
	// Tx Machine
	if p.TxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpTxmStateOff {
		t.Error("ERROR TX Machine state incorrect expected",
			lacp.LacpTxmStateOff, "actual",
			p.TxMachineFsm.Machine.Curr.CurrentState())
	}
}
