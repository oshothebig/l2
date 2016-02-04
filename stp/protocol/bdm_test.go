// bdm_test.go
package stp

import ()

func UsedForTestOnlyBdmTestSetup(t *testing.T) (p *StpPort) {

	bridgeconfig := &StpBridgeConfig{
		Dot1dBridgeAddressKey:      "00:55:55:55:55:55",
		Dot1dStpPriorityKey:        0x20,
		Dot1dStpBridgeMaxAge:       BridgeMaxAgeDefault,
		Dot1dStpBridgeHelloTime:    BridgeHelloTimeDefault,
		Dot1dStpBridgeForwardDelay: BridgeForwardDelayDefault,
		Dot1dStpBridgeForceVersion: 2,
		Dot1dStpBridgeTxHoldCount:  TransmitHoldCountDefault,
	}

	//StpBridgeCreate
	b := NewStpBridge(bridgeconfig)
	BdmMachineFSMBuild(b)
	b.PrsMachineFsm.Machine.ProcessEvent("TEST", BdmEventBegin, nil)
	b.PrsMachineFsm.Machine.ProcessEvent("TEST", PrsEventUnconditionallFallThrough, nil)

	// configure a port
	stpconfig := &StpPortConfig{
		Dot1dStpPortKey:               TEST_RX_PORT_CONFIG_IFINDEX,
		Dot1dStpPortPriority:          0x80,
		Dot1dStpPortEnable:            true,
		Dot1dStpPortPathCost:          1,
		Dot1dStpPortPathCost32:        1,
		Dot1dStpPortProtocolMigration: 0,
		Dot1dStpPortAdminPointToPoint: StpPointToPointForceFalse,
		Dot1dStpPortAdminEdgePort:     0,
		Dot1dStpPortAdminPathCost:     0,
		Dot1dStpBridgeId:              b.BridgeIdentifier,
	}

	// create a port
	p = NewStpPort(stpconfig)

	// lets only start the Port Information State Machine
	p.PimMachineMain()

	// going just send event and not start main as we just did above
	p.BEGIN(true)

	// only instanciated object not starting go routine
	PrtMachineFSMBuild(p)
	PrxmMachineFSMBuild(p)
	PtxmMachineFSMBuild(p)
	BdmMachineFSMBuild(p)
	PtmMachineFSMBuild(p)
	PtmMachineFSMBuild(p)
	TcMachineFSMBuild(p)
	PstMachineFSMBuild(p)

	if p.PimMachineFsm.Machine.Curr.PreviousState() != PimStateNone {
		t.Error("Failed to Initial Port Information machine state not set correctly", p.PpmmMachineFsm.Machine.Curr.PreviousState())
		t.FailNow()
	}

	if p.PimMachineFsm.Machine.Curr.CurrentState() != PimStateDisabled {
		t.Error("Failed to Initial Port Information machine state not set correctly", p.PpmmMachineFsm.Machine.Curr.CurrentState())
		t.FailNow()
	}

	return p
}
