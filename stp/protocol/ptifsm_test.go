// timer_test.go
package stp

import (
	"net"
	"testing"
	"time"
)

func UsedForTestOnlyPtmInitPortConfigTest() {

	if PortConfigMap == nil {
		PortConfigMap = make(map[int32]portConfig)
	}
	// In order to test a packet we must listen on loopback interface
	// and send on interface we expect to receive on.  In order
	// to do this a couple of things must occur the PortConfig
	// must be updated with "dummy" ifindex pointing to 'lo'
	TEST_RX_PORT_CONFIG_IFINDEX = 0x0ADDBEEF
	PortConfigMap[TEST_RX_PORT_CONFIG_IFINDEX] = portConfig{Name: "lo",
		HardwareAddr: net.HardwareAddr{0x00, 0x11, 0x11, 0x22, 0x22, 0x33},
	}
	PortConfigMap[TEST_TX_PORT_CONFIG_IFINDEX] = portConfig{Name: "lo",
		HardwareAddr: net.HardwareAddr{0x00, 0x33, 0x22, 0x22, 0x11, 0x11},
	}
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
	UsedForTestOnlySetupAsicDPlugin()
}

func UsedForTestOnlyPtmTestSetup(t *testing.T) (p *StpPort) {
	UsedForTestOnlyPtmInitPortConfigTest()

	// configure a port
	stpconfig := &StpPortConfig{
		IfIndex:           TEST_RX_PORT_CONFIG_IFINDEX,
		Priority:          0x80,
		Enable:            true,
		PathCost:          1,
		ProtocolMigration: 0,
		AdminPointToPoint: StpPointToPointForceFalse,
		AdminEdgePort:     false,
		AdminPathCost:     0,
		BrgIfIndex:        100,
	}

	bridgeconfig := &StpBridgeConfig{
		Address:      "00:55:55:55:55:55",
		Priority:     0x20,
		MaxAge:       BridgeMaxAgeDefault,
		HelloTime:    BridgeHelloTimeDefault,
		ForwardDelay: BridgeForwardDelayDefault,
		ForceVersion: 2,
		TxHoldCount:  TransmitHoldCountDefault,
		Vlan:         100,
	}

	//StpBridgeCreate
	b := NewStpBridge(bridgeconfig)
	PrsMachineFSMBuild(b)
	b.PrsMachineFsm.Machine.ProcessEvent("TEST", PrsEventBegin, nil)
	b.PrsMachineFsm.Machine.ProcessEvent("TEST", PrsEventUnconditionallFallThrough, nil)

	// create a port
	p = NewStpPort(stpconfig)

	// start timer tick machine
	p.PtmMachineMain()

	// lets not start the main routines for the other state machines
	p.BEGIN(true)

	p.PollingTimer.Stop()

	// NOTE: must be called after BEGIN
	// Lets Instatiate but not run the following Machines
	// 1) Port Information Machine
	// 2) Port Protocol Migration Machine
	PrxmMachineFSMBuild(p)
	PrtMachineFSMBuild(p)
	PimMachineFSMBuild(p)
	PtxmMachineFSMBuild(p)
	BdmMachineFSMBuild(p)
	TcMachineFSMBuild(p)
	PstMachineFSMBuild(p)
	PpmmMachineFSMBuild(p)

	return p

}

func UsedForTestOnlyPtmTestTeardown(p *StpPort, t *testing.T) {

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
	if len(p.PrxmMachineFsm.PrxmEvents) > 0 {
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
	p.PrtMachineFsm = nil
	p.PimMachineFsm = nil
	p.PtxmMachineFsm = nil
	p.BdmMachineFsm = nil
	p.PrxmMachineFsm = nil
	p.TcMachineFsm = nil
	p.PstMachineFsm = nil
	p.PpmmMachineFsm = nil

	b := p.b
	p.b.PrsMachineFsm = nil
	DelStpPort(p)
	DelStpBridge(b, true)
}

func TestPtmEdgeDelayTimerExpire(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.BdmMachineFsm.Machine.Curr.SetState(BdmStateNotEdge)

	// bdm event param requirements
	p.AutoEdgePort = true
	p.EdgeDelayWhileTimer.count = 1
	p.SendRSTP = true
	p.Proposing = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {

		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.BdmMachineFsm != nil {
			p.BdmMachineFsm.BdmEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.BdmMachineFsm.BdmEvents
	if event.e != BdmEventEdgeDelayWhileEqualZeroAndAutoEdgeAndSendRSTPAndProposing {
		t.Error("ERROR: Error did not receive BDM Machine event after operedge expired")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmEdgeDelayTimerNotEqualMigrateTimeAndPortDisabled(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PrxmMachineFsm.Machine.Curr.SetState(PrxmStateDiscard)

	// bdm event param requirements
	p.AutoEdgePort = true
	p.EdgeDelayWhileTimer.count = 10
	p.SendRSTP = false
	p.Proposing = false
	p.AdminPortEnabled = false
	p.PortEnabled = false

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 3)
		if p.PrxmMachineFsm != nil {
			p.PrxmMachineFsm.PrxmEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrxmMachineFsm.PrxmEvents
	if event.e != PrxmEventEdgeDelayWhileNotEqualMigrateTimeAndNotPortEnabled {
		t.Error("ERROR: Error did not receive RXM Machine event as expected", event.e)
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmFdWhileNotEqualMaxAgeAndPrtStateDisabledPort(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDisabledPort)

	// bdm event param requirements
	p.FdWhileTimer.count = 6
	// lets force the RX event to not happen
	p.EdgeDelayWhileTimer.count = MigrateTimeDefault + 1
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = false
	p.PortEnabled = false

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrtMachineFsm.PrtEvents
	if event.e != PrtEventFdWhileNotEqualMaxAgeAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PRT Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmFdWhileNotEqualMaxAgeAndPrtStateAlternatePort(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateAlternatePort)

	// bdm event param requirements
	p.FdWhileTimer.count = 6
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrtMachineFsm.PrtEvents
	if event.e != PrtEventFdWhileNotEqualForwardDelayAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PRT Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmFdWhileTimerExpiredRootPortRstpVersionNotLearn(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite PRT Machine
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateRootPort)

	// bdm event param requirements
	p.RstpVersion = true
	p.Learn = false
	p.FdWhileTimer.count = 1
	// lets ensure that this event does not get generated as well
	p.RrWhileTimer.count = int32(p.b.RootTimes.ForwardingDelay + 1)
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 4)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrtMachineFsm.PrtEvents
	if event.e != PrtEventFdWhileEqualZeroAndRstpVersionAndNotLearnAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PRT Machine event as expected", event.e)
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmFdWhileTimerExpiredRootPortLearnAndNotForward(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateRootPort)

	// bdm event param requirements
	p.RstpVersion = true
	p.Learn = true
	p.Forward = false
	p.FdWhileTimer.count = 1
	// lets ensure that this event does not get generated as well
	p.RrWhileTimer.count = int32(p.b.RootTimes.ForwardingDelay + 1)
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrtMachineFsm.PrtEvents
	if event.e != PrtEventFdWhileEqualZeroAndRstpVersionAndLearnAndNotForwardAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PRT Machine event as expected", event.e)
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmFdWhileTimerExpiredDesignatedPortRrwhileEqualZeroAndNotSyncAndNotLearn(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDesignatedPort)

	// bdm event param requirements
	p.RstpVersion = true
	p.FdWhileTimer.count = 1
	p.RrWhileTimer.count = 0
	p.Sync = false
	p.Learn = false
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrtMachineFsm.PrtEvents
	if event.e != PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndNotLearnAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PRT Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmFdWhileTimerExpiredDesignatedPortRrwhileEqualZeroAndNotSyncAndLearnAndNotForward(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDesignatedPort)

	// bdm event param requirements
	p.RstpVersion = true
	p.FdWhileTimer.count = 1
	p.RrWhileTimer.count = 0
	p.Sync = false
	p.Learn = true
	p.Forward = false
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PrtMachineFsm.PrtEvents
	if event.e != PrtEventFdWhileEqualZeroAndRrWhileEqualZeroAndNotSyncAndLearnAndNotForwardSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PRT Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmHelloWhenTimerExpired(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PtxmMachineFsm.Machine.Curr.SetState(PtxmStateIdle)
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDesignatedPort)

	// bdm event param requirements
	p.Role = PortRoleDesignatedPort
	p.SelectedRole = PortRoleDesignatedPort
	p.HelloWhenTimer.count = 1
	p.Sync = true
	p.Synced = true
	p.Learn = true
	p.Forward = true
	p.Learning = true
	p.Forwarding = true
	p.RstpVersion = true
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PtxmMachineFsm.PtxmEvents
	if event.e != PtxmEventHelloWhenEqualsZeroAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PTX Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmHelloWhenTimerNotEqualZeroSendRSTPNewInfoTxCountLessThanTxHoldCount(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PtxmMachineFsm.Machine.Curr.SetState(PtxmStateIdle)
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateDesignatedPort)

	// bdm event param requirements
	p.Role = PortRoleDesignatedPort
	p.SelectedRole = PortRoleDesignatedPort
	p.SendRSTP = true
	p.NewInfo = true
	p.b.TxHoldCount = TransmitHoldCountDefault - 1
	p.Sync = true
	p.Synced = true
	p.Learn = true
	p.Forward = true
	p.Learning = true
	p.Forwarding = true
	p.RstpVersion = true
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PtxmMachineFsm.PtxmEvents
	if event.e != PtxmEventSendRSTPAndNewInfoAndTxCountLessThanTxHoldCoundAndHelloWhenNotEqualZeroAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PTX Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmHelloWhenTimerNotEqualZeroNotSendRSTPNewInfoRootPortTxCountLessThanTxHoldCount(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PtxmMachineFsm.Machine.Curr.SetState(PtxmStateIdle)
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateRootPort)

	// bdm event param requirements
	p.Role = PortRoleRootPort
	p.SelectedRole = PortRoleRootPort
	p.SendRSTP = false
	p.NewInfo = true
	p.b.TxHoldCount = TransmitHoldCountDefault - 1
	p.Sync = true
	p.Synced = true
	p.Learn = true
	p.Forward = true
	p.Learning = true
	p.Forwarding = true
	p.RstpVersion = true
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PtxmMachineFsm.PtxmEvents
	if event.e != PtxmEventNotSendRSTPAndNewInfoAndRootPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PTX Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}

func TestPtmHelloWhenTimerNotEqualZeroNotSendRSTPNewInfoDesignatedPortTxCountLessThanTxHoldCount(t *testing.T) {

	p := UsedForTestOnlyPtmTestSetup(t)

	// prequisite BDM Machine
	p.PtxmMachineFsm.Machine.Curr.SetState(PtxmStateIdle)
	p.PrtMachineFsm.Machine.Curr.SetState(PrtStateRootPort)

	// bdm event param requirements
	p.Role = PortRoleDesignatedPort
	p.SelectedRole = PortRoleDesignatedPort
	p.SendRSTP = false
	p.NewInfo = true
	p.b.TxHoldCount = TransmitHoldCountDefault - 1
	p.Sync = true
	p.Synced = true
	p.Learn = true
	p.Forward = true
	p.Learning = true
	p.Forwarding = true
	p.RstpVersion = true
	p.Selected = true
	p.UpdtInfo = false
	p.AdminPortEnabled = true
	p.PortEnabled = true

	// may need to delay a bit in order to allow for packet to be receive
	// by pcap
	go func(p *StpPort) {
		// lets sleep later than the tick timer 2 seconds should be enough
		time.Sleep(time.Second * 2)
		if p.PrtMachineFsm != nil {
			p.PrtMachineFsm.PrtEvents <- MachineEvent{
				e:   0, // invalid event
				src: "TEST",
			}
		}
	}(p)

	event := <-p.PtxmMachineFsm.PtxmEvents
	if event.e != PtxmEventNotSendRSTPAndNewInfoAndDesignatedPortAndTxCountLessThanTxHoldCountAndHellWhenNotEqualZeroAndSelectedAndNotUpdtInfo {
		t.Error("ERROR: Error did not receive PTX Machine event as expected")
	}

	UsedForTestOnlyPtmTestTeardown(p, t)
}
