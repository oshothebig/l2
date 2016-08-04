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
// config_test.go
package drcp

import (
	"l2/lacp/protocol/lacp"
	"l2/lacp/protocol/utils"
	"net"
	"testing"
	asicdmock "utils/asicdClient/mock"
	"utils/logging"
)

const ipplink1 int32 = 3
const aggport1 int32 = 1
const aggport2 int32 = 2

const ipplink2 int32 = 4
const aggport3 int32 = 5
const aggport4 int32 = 6

type MyTestMock struct {
	asicdmock.MockAsicdClientMgr
}

func OnlyForTestSetup() {
	logger, _ := logging.NewLogger("lacpd", "TEST", false)
	utils.SetLaLogger(logger)
	utils.SetAsicDPlugin(&MyTestMock{})
}

func OnlyForTestTeardown() {

	utils.SetLaLogger(nil)
	utils.DeleteAllAsicDPlugins()
}

func OnlyForTestSetupCreateAggGroup(aggId uint32) *lacp.LaAggregator {
	a1conf := &lacp.LaAggConfig{
		Mac: [6]uint8{0x00, 0x00, 0x01, 0x01, 0x01, 0x01},
		Id:  int(aggId),
		Key: uint16(aggId),
		Lacp: lacp.LacpConfigInfo{Interval: lacp.LacpFastPeriodicTime,
			Mode:           lacp.LacpModeActive,
			SystemIdMac:    "00:00:00:00:00:64",
			SystemPriority: 128},
	}
	lacp.CreateLaAgg(a1conf)

	p1conf := &lacp.LaAggPortConfig{
		Id:      uint16(aggport1),
		Prio:    0x80,
		Key:     uint16(aggId),
		AggId:   int(aggId),
		Enable:  true,
		Mode:    lacp.LacpModeActive,
		Timeout: lacp.LacpShortTimeoutTime,
		Properties: lacp.PortProperties{
			Mac:    net.HardwareAddr{0x00, byte(aggport1), 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: lacp.LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   utils.PortConfigMap[aggport1].Name,
		TraceEna: false,
	}

	lacp.CreateLaAggPort(p1conf)
	lacp.AddLaAggPortToAgg(a1conf.Key, p1conf.Id)

	var a *lacp.LaAggregator
	if lacp.LaFindAggById(a1conf.Id, &a) {
		return a
	}
	return nil
}

func ConfigTestSetup() {
	OnlyForTestSetup()
	utils.PortConfigMap[ipplink1] = utils.PortConfig{Name: "SIMeth1.3",
		HardwareAddr: net.HardwareAddr{0x00, 0x33, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[aggport1] = utils.PortConfig{Name: "SIMeth1.1",
		HardwareAddr: net.HardwareAddr{0x00, 0x11, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[aggport2] = utils.PortConfig{Name: "SIMeth1.2",
		HardwareAddr: net.HardwareAddr{0x00, 0x22, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[ipplink2] = utils.PortConfig{Name: "SIMeth0.3",
		HardwareAddr: net.HardwareAddr{0x00, 0x44, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[aggport3] = utils.PortConfig{Name: "SIMeth0.1",
		HardwareAddr: net.HardwareAddr{0x00, 0x55, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[aggport4] = utils.PortConfig{Name: "SIMeth0.2",
		HardwareAddr: net.HardwareAddr{0x00, 0x66, 0x11, 0x22, 0x22, 0x33},
	}
}
func ConfigTestTeardwon() {

	OnlyForTestTeardown()
	delete(utils.PortConfigMap, ipplink1)
	delete(utils.PortConfigMap, aggport1)
	delete(utils.PortConfigMap, aggport2)
	delete(utils.PortConfigMap, ipplink2)
	delete(utils.PortConfigMap, aggport3)
	delete(utils.PortConfigMap, aggport4)
}

func TestDistributedRelayValidCreateAggWithPortsThenCreateDR(t *testing.T) {

	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)},
		DrniAggregator:                    uint32(a.AggId),
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}
	// map vlan 100 to this system
	// in real system this should be filled in by vlan membership
	cfg.DrniConvAdminGateway[100][0] = cfg.DrniPortalSystemNumber

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err != nil {
		t.Error("Parameter check failed for what was expected to be a valid config", err)
	}

	CreateDistributedRelay(cfg)

	// configuration is incomplete because lag has not been created as part of this
	if len(DistributedRelayDB) == 0 ||
		len(DistributedRelayDBList) == 0 {
		t.Error("ERROR Distributed Relay Object was not added to global DB's")
	}
	dr, ok := DistributedRelayDB[cfg.DrniName]
	if !ok {
		t.Error("ERROR Distributed Relay Object was not found in global DB's")
	}

	// check the inital state of each of the state machines
	if dr.a == nil {
		t.Error("ERROR BEGIN was called before an Agg has been attached")
	}
	if dr.PsMachineFsm == nil ||
		dr.PsMachineFsm.Machine.Curr.CurrentState() != PsmStatePortalSystemInitialize {
		t.Error("ERROR BEGIN Initial Portal System Machine state is not correct", dr.PsMachineFsm.Machine.Curr.CurrentState())
	}
	if dr.GMachineFsm == nil ||
		dr.GMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayInitialize {
		t.Error("ERROR BEGIN Initial Gateway Machine state is not correct", dr.GMachineFsm.Machine.Curr.CurrentState())
	}
	if dr.AMachineFsm == nil ||
		dr.AMachineFsm.Machine.Curr.CurrentState() != AmStateDRNIPortInitialize {
		t.Error("ERROR BEGIN Initial Aggregator System Machine state is not correct", dr.AMachineFsm.Machine.Curr.CurrentState())
	}

	if len(dr.Ipplinks) == 0 {
		t.Error("ERROR Why did the IPL IPP link not get created")
	}

	for _, ipp := range dr.Ipplinks {
		// IPL should be disabled thus state should be in initialized state
		if ipp.RxMachineFsm == nil ||
			ipp.RxMachineFsm.Machine.Curr.CurrentState() != RxmStateExpired {
			t.Error("ERROR BEGIN Initial Receive Machine state is not correct", ipp.RxMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.PtxMachineFsm == nil ||
			ipp.PtxMachineFsm.Machine.Curr.CurrentState() != PtxmStateSlowPeriodic {
			t.Error("ERROR BEGIN Initial Periodic Tx Machine state is not correct", ipp.PtxMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.TxMachineFsm == nil ||
			ipp.TxMachineFsm.Machine.Curr.CurrentState() != TxmStateOff {
			t.Error("ERROR BEGIN Initial Tx Machine state is not correct", ipp.TxMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.NetIplShareMachineFsm == nil ||
			ipp.NetIplShareMachineFsm.Machine.Curr.CurrentState() != NetIplSharemStateNoManipulatedFramesSent {
			t.Error("ERROR BEGIN Initial Net/IPL Sharing Machine state is not correct", ipp.NetIplShareMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.IAMachineFsm == nil ||
			ipp.IAMachineFsm.Machine.Curr.CurrentState() != IAmStateIPPPortInitialize {
			t.Error("ERROR BEGIN Initial IPP Aggregator state is not correct", ipp.IAMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.IGMachineFsm == nil ||
			ipp.IGMachineFsm.Machine.Curr.CurrentState() != IGmStateIPPGatewayInitialize {
			t.Error("ERROR BEGIN Initial IPP Gateway Machine state is not correct", ipp.IGMachineFsm.Machine.Curr.CurrentState())
		}
	}
	DeleteDistributedRelay(cfg.DrniName)
	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}
