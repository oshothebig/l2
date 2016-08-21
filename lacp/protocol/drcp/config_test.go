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
	"fmt"
	"github.com/google/gopacket"
	"l2/lacp/protocol/lacp"
	"l2/lacp/protocol/utils"
	"net"
	"testing"
	"time"
	asicdmock "utils/asicdClient/mock"
	"utils/commonDefs"
	"utils/logging"
)

// first two bytes are priority but in case of ipp this is the neighbor system number
const ipplink1 int32 = 3
const aggport1 int32 = 1
const aggport2 int32 = 2

// first two bytes are priority but in case of ipp this is the neighbor system number
const ipplink2 int32 = 4
const aggport3 int32 = 5
const aggport4 int32 = 6

const LaAggPort1NeighborActor = 10
const LaAggPort2NeighborActor = 30
const LaAggPort1Peer = 20
const LaAggPort2Peer = 21
const DRNeighborIpp1 = 11
const DRNeighborIpp2 = 31
const LaAggPortNeighborActor1If = "SIMeth0"
const LaAggPortNeighborActor2If = "SIMeth1"
const LaAggPortPeerIf1 = "SIM2eth0"
const LaAggPortPeerIf2 = "SIM2eth1"
const DRNeighborIppIf1 = "SIMIPPeth2"
const DRNeighborIppIf2 = "SIMIPPeth3"

type MyTestMock struct {
	asicdmock.MockAsicdClientMgr
}

func (m *MyTestMock) GetBulkVlan(curMark, count int) (*commonDefs.VlanGetInfo, error) {

	getinfo := &commonDefs.VlanGetInfo{
		StartIdx: 1,
		EndIdx:   1,
		Count:    1,
		More:     false,
		VlanList: make([]commonDefs.Vlan, 1),
	}

	getinfo.VlanList[0] = commonDefs.Vlan{
		VlanId:      100,
		IfIndexList: []int32{aggport1, aggport2, LaAggPort1NeighborActor, LaAggPort2NeighborActor, LaAggPort1Peer, LaAggPort2Peer},
	}
	return getinfo, nil
}

func OnlyForTestSetup() {
	logger, _ := logging.NewLogger("lacpd", "TEST", false)
	utils.SetLaLogger(logger)
	utils.DeleteAllAsicDPlugins()
	utils.SetAsicDPlugin(&MyTestMock{})
	// fill in conversations
	GetAllCVIDConversations()
}

func OnlyForTestTeardown() {

	utils.SetLaLogger(nil)
	utils.DeleteAllAsicDPlugins()
	ConversationIdMap[100].Valid = false
	ConversationIdMap[100].PortList = nil
	ConversationIdMap[100].Cvlan = 0
	ConversationIdMap[100].Refcnt = 0
	ConversationIdMap[100].Idtype = [4]uint8{}
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
		a.DistributedPortNumList = append(a.DistributedPortNumList, utils.PortConfigMap[aggport1].Name)

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

func TestConfigDistributedRelayValidCreateAggWithPortsThenCreateDR(t *testing.T) {

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

	waitChan := make(chan bool)
	go func(wc *chan bool) {
		for i := 0; i < 10 && (dr.PsMachineFsm.Machine.Curr.CurrentState() != PsmStatePortalSystemUpdate); i++ {
			time.Sleep(time.Millisecond * 10)
		}
		*wc <- true
	}(&waitChan)

	<-waitChan

	// Rx machine sets the change portal which should inform the psm
	if dr.PsMachineFsm == nil ||
		dr.PsMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayUpdate {
		t.Error("ERROR BEGIN Initial Portal System Machine state is not correct", PsmStateStrMap[dr.PsMachineFsm.Machine.Curr.CurrentState()])
	}

	go func(wc *chan bool) {
		for i := 0; i < 10 && dr.GMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayUpdate; i++ {
			time.Sleep(time.Millisecond * 10)
		}
		*wc <- true
	}(&waitChan)

	<-waitChan

	// Ps Machine updates the Gm based on the rx gateway update
	if dr.GMachineFsm == nil ||
		dr.GMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayUpdate {
		t.Error("ERROR BEGIN Initial Gateway Machine state is not correct", GmStateStrMap[dr.GMachineFsm.Machine.Curr.CurrentState()])
	}
	if dr.AMachineFsm == nil ||
		dr.AMachineFsm.Machine.Curr.CurrentState() != AmStateDRNIPortUpdate {
		t.Error("ERROR BEGIN Initial Aggregator System Machine state is not correct", AmStateStrMap[dr.AMachineFsm.Machine.Curr.CurrentState()])
	}

	if len(dr.Ipplinks) == 0 {
		t.Error("ERROR Why did the IPL IPP link not get created")
	}

	for _, ipp := range dr.Ipplinks {
		// IPL should be disabled thus state should be in initialized state
		if ipp.RxMachineFsm == nil ||
			ipp.RxMachineFsm.Machine.Curr.CurrentState() != RxmStateExpired {
			t.Error("ERROR BEGIN Initial Receive Machine state is not correct", RxmStateStrMap[ipp.RxMachineFsm.Machine.Curr.CurrentState()])
		}
		// port is enabled and drcp is enabled thus we should be in fast periodic state
		if ipp.PtxMachineFsm == nil {
			t.Error("ERROR BEGIN Initial Periodic Tx Machine state is not correct")
		}
		if ipp.TxMachineFsm == nil ||
			ipp.TxMachineFsm.Machine.Curr.CurrentState() != TxmStateOff {
			t.Error("ERROR BEGIN Initial Tx Machine state is not correct", TxmStateStrMap[ipp.TxMachineFsm.Machine.Curr.CurrentState()])
		}
		if ipp.NetIplShareMachineFsm == nil ||
			ipp.NetIplShareMachineFsm.Machine.Curr.CurrentState() != NetIplSharemStateNoManipulatedFramesSent {
			t.Error("ERROR BEGIN Initial Net/IPL Sharing Machine state is not correct", ipp.NetIplShareMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.IAMachineFsm == nil ||
			ipp.IAMachineFsm.Machine.Curr.CurrentState() != IAmStateIPPPortInitialize {
			t.Error("ERROR BEGIN Initial IPP Aggregator state is not correct", IAmStateStrMap[ipp.IAMachineFsm.Machine.Curr.CurrentState()])
		}

		go func(wc *chan bool) {
			for i := 0; i < 10 && ipp.IGMachineFsm.Machine.Curr.CurrentState() != IGmStateIPPGatewayUpdate; i++ {
				time.Sleep(time.Millisecond * 10)
			}
			*wc <- true
		}(&waitChan)

		<-waitChan

		if ipp.IGMachineFsm == nil ||
			ipp.IGMachineFsm.Machine.Curr.CurrentState() != IGmStateIPPGatewayUpdate {
			t.Error("ERROR BEGIN Initial IPP Gateway Machine state is not correct", GmStateStrMap[ipp.IGMachineFsm.Machine.Curr.CurrentState()])
		}
	}
	DeleteDistributedRelay(cfg.DrniName)

	if len(DistributedRelayDB) != 0 ||
		len(DistributedRelayDBList) != 0 {
		t.Error("ERROR Distributed Relay DB was not cleaned up")
	}
	if len(DRCPIppDB) != 0 ||
		len(DRCPIppDBList) != 0 {
		t.Error("ERROR IPP DB was not cleaned up")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigDistributedRelayCreateDRThenCreateAgg(t *testing.T) {

	ConfigTestSetup()

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)},
		DrniAggregator:                    200,
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
	// This should fail as agg has not been created yet
	AttachAggregatorToDistributedRelay(200)

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
	if dr.a != nil {
		t.Error("ERROR Agg does not exist no associated should exist")
	}
	if dr.PsMachineFsm != nil {
		t.Error("ERROR BEGIN Initial Portal System Machine state was created, provisioning incomplete")
	}
	if dr.GMachineFsm != nil {
		t.Error("ERROR BEGIN Initial Gateway Machine state was created, provisioning incomplete")
	}
	if dr.AMachineFsm != nil {
		t.Error("ERROR BEGIN Initial Aggregator System Machine state was created, provisioning incomplete")
	}

	for _, ipp := range dr.Ipplinks {
		if ipp.DRCPEnabled {
			t.Error("ERROR Why is the IPL IPP link DRCP enabled")
		}
	}

	// TEST aggregator created after DR
	a := OnlyForTestSetupCreateAggGroup(200)
	AttachAggregatorToDistributedRelay(200)

	// check the inital state of each of the state machines
	if dr.a == nil {
		t.Error("ERROR BEGIN was called before an Agg has been attached")
	}
	waitChan := make(chan bool)
	go func(wc *chan bool) {
		for i := 0; i < 10 && (dr.PsMachineFsm.Machine.Curr.CurrentState() != PsmStatePortalSystemUpdate); i++ {
			time.Sleep(time.Millisecond * 10)
		}
		*wc <- true
	}(&waitChan)

	<-waitChan

	// Rx machine sets the change portal which should inform the psm
	if dr.PsMachineFsm == nil ||
		dr.PsMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayUpdate {
		t.Error("ERROR BEGIN Initial Portal System Machine state is not correct", PsmStateStrMap[dr.PsMachineFsm.Machine.Curr.CurrentState()])
	}

	go func(wc *chan bool) {
		for i := 0; i < 10 && dr.GMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayUpdate; i++ {
			time.Sleep(time.Millisecond * 10)
		}
		*wc <- true
	}(&waitChan)

	<-waitChan

	// Ps Machine updates the Gm based on the rx gateway update
	if dr.GMachineFsm == nil ||
		dr.GMachineFsm.Machine.Curr.CurrentState() != GmStateDRNIGatewayUpdate {
		t.Error("ERROR BEGIN Initial Gateway Machine state is not correct", GmStateStrMap[dr.GMachineFsm.Machine.Curr.CurrentState()])
	}
	if dr.AMachineFsm == nil ||
		dr.AMachineFsm.Machine.Curr.CurrentState() != AmStateDRNIPortUpdate {
		t.Error("ERROR BEGIN Initial Aggregator System Machine state is not correct", AmStateStrMap[dr.AMachineFsm.Machine.Curr.CurrentState()])
	}

	if len(dr.Ipplinks) == 0 {
		t.Error("ERROR Why did the IPL IPP link not get created")
	}

	for _, ipp := range dr.Ipplinks {
		// IPL should be disabled thus state should be in initialized state
		if ipp.RxMachineFsm == nil ||
			ipp.RxMachineFsm.Machine.Curr.CurrentState() != RxmStateExpired {
			t.Error("ERROR BEGIN Initial Receive Machine state is not correct", RxmStateStrMap[ipp.RxMachineFsm.Machine.Curr.CurrentState()])
		}
		// port is enabled and drcp is enabled thus we should be in fast periodic state
		if ipp.PtxMachineFsm == nil {
			t.Error("ERROR BEGIN Initial Periodic Tx Machine state is not correct")
		}
		if ipp.TxMachineFsm == nil ||
			ipp.TxMachineFsm.Machine.Curr.CurrentState() != TxmStateOff {
			t.Error("ERROR BEGIN Initial Tx Machine state is not correct", TxmStateStrMap[ipp.TxMachineFsm.Machine.Curr.CurrentState()])
		}
		if ipp.NetIplShareMachineFsm == nil ||
			ipp.NetIplShareMachineFsm.Machine.Curr.CurrentState() != NetIplSharemStateNoManipulatedFramesSent {
			t.Error("ERROR BEGIN Initial Net/IPL Sharing Machine state is not correct", ipp.NetIplShareMachineFsm.Machine.Curr.CurrentState())
		}
		if ipp.IAMachineFsm == nil ||
			ipp.IAMachineFsm.Machine.Curr.CurrentState() != IAmStateIPPPortInitialize {
			t.Error("ERROR BEGIN Initial IPP Aggregator state is not correct", IAmStateStrMap[ipp.IAMachineFsm.Machine.Curr.CurrentState()])
		}

		go func(wc *chan bool) {
			for i := 0; i < 10 && ipp.IGMachineFsm.Machine.Curr.CurrentState() != IGmStateIPPGatewayUpdate; i++ {
				time.Sleep(time.Millisecond * 10)
			}
			*wc <- true
		}(&waitChan)

		<-waitChan

		if ipp.IGMachineFsm == nil ||
			ipp.IGMachineFsm.Machine.Curr.CurrentState() != IGmStateIPPGatewayUpdate {
			t.Error("ERROR BEGIN Initial IPP Gateway Machine state is not correct", GmStateStrMap[ipp.IGMachineFsm.Machine.Curr.CurrentState()])
		}
	}

	DeleteDistributedRelay(cfg.DrniName)

	if len(DistributedRelayDB) != 0 ||
		len(DistributedRelayDBList) != 0 {
		t.Error("ERROR Distributed Relay DB was not cleaned up")
	}
	if len(DRCPIppDB) != 0 ||
		len(DRCPIppDBList) != 0 {
		t.Error("ERROR IPP DB was not cleaned up")
	}
	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigDistributedRelayInValidCreateDRNoAgg(t *testing.T) {

	ConfigTestSetup()
	//a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)},
		DrniAggregator:                    200,
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
	if dr.a != nil {
		t.Error("ERROR Agg exists no associated should exist")
	}
	if dr.PsMachineFsm != nil {
		t.Error("ERROR BEGIN Initial Portal System Machine state was created, provisioning incomplete")
	}
	if dr.GMachineFsm != nil {
		t.Error("ERROR BEGIN Initial Gateway Machine state was created, provisioning incomplete")
	}
	if dr.AMachineFsm != nil {
		t.Error("ERROR BEGIN Initial Aggregator System Machine state was created, provisioning incomplete")
	}

	for _, ipp := range dr.Ipplinks {
		if ipp.DRCPEnabled {
			t.Error("ERROR Why is the IPL IPP link DRCP enabled")
		}
	}

	DeleteDistributedRelay(cfg.DrniName)

	if len(DistributedRelayDB) != 0 ||
		len(DistributedRelayDBList) != 0 {
		t.Error("ERROR Distributed Relay DB was not cleaned up")
	}
	if len(DRCPIppDB) != 0 ||
		len(DRCPIppDBList) != 0 {
		t.Error("ERROR IPP DB was not cleaned up")
	}

	ConfigTestTeardwon()
}

func TestConfigInvalidPortalAddressString(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)
	/*




	 */
	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE", // invalid!!!
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)},
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail for bad Portal Address")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidThreePortalSystemSet(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             true, // invalid not supported
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)},
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail setting 3P system")
	}
	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidPortalSytemNumber(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            0, // invalid in 2P system
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)},
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail portal system number 0")
	}
	// invalid in 2P system
	cfg.DrniPortalSystemNumber = 3
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail portal system number 3")
	}
	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidIntraPortalLink(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail IPP link not supplied")
	}
	// invalid ipp link
	cfg.DrniIntraPortalLinkList[0] = 300
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail invalid port")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidGatewayAlgorithm(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:88:C2:01", // invalid string
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Gateway Algorithm")
	}

	cfg.DrniGatewayAlgorithm = ""
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Gateway Algorithm empty string")
	}

	cfg.DrniGatewayAlgorithm = "00:80:C2"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Gateway Algorithm wrong format to short missing actual type byte")
	}

	cfg.DrniGatewayAlgorithm = "00-80-C2-02"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Gateway Algorithm separator")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidNeighborGatewayAlgorithm(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "01:80:C2:01", // invalid string
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Gateway Algorithm")
	}

	cfg.DrniNeighborAdminGatewayAlgorithm = ""
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Gateway Algorithm empty string")
	}

	cfg.DrniNeighborAdminGatewayAlgorithm = "00:80:C2"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Gateway Algorithm wrong format to short missing actual type byte")
	}

	cfg.DrniNeighborAdminGatewayAlgorithm = "00-80-C2-02"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Gateway Algorithm separator")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidNeighborPortAlgorithm(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "10:80:C2:01", // invalid string
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Port Algorithm")
	}

	cfg.DrniNeighborAdminPortAlgorithm = ""
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Port Algorithm empty string")
	}

	cfg.DrniNeighborAdminPortAlgorithm = "00:80:C2"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Port Algorithm wrong format to short missing actual type byte")
	}

	cfg.DrniNeighborAdminPortAlgorithm = "00-80-C2-02"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Neighbor Port Algorithm separator")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidEncapMethod(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "22:80:C2:01", // invalid string
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Encap Method")
	}

	cfg.DrniEncapMethod = ""
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Encap Method empty string")
	}

	cfg.DrniEncapMethod = "00:80:C2"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Encap Method wrong format to short missing actual type byte")
	}

	cfg.DrniEncapMethod = "00-80-C2-02"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Encap Method separator")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidPortConversationControl(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       true,                // invalid assuming protocol will take care of conversation always
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Port Conversaton Control")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func TestConfigInvalidPortalPortProtocolDA(t *testing.T) {
	ConfigTestSetup()
	a := OnlyForTestSetupCreateAggGroup(100)

	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(ipplink1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "00:80:C2:01",
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01", // invalid string
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C0:00:00:03", // invalid
	}

	err := DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail invalid Portal Port Potocol DA")
	}

	cfg.DrniIntraPortalPortProtocolDA = "01-80-C2-00-00-11"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Portal Port Potocol DA different format")
	}

	cfg.DrniIntraPortalPortProtocolDA = "80-C2-00-00-11"
	err = DistrubtedRelayConfigParamCheck(cfg)
	if err == nil {
		t.Error("Parameter check did not fail Invalid Portal Port Potocol DA not enough bytes")
	}

	lacp.DeleteLaAgg(a.AggId)
	ConfigTestTeardwon()
}

func FullBackToBackConfigTestSetup() {
	OnlyForTestSetup()
	utils.PortConfigMap[LaAggPort1NeighborActor] = utils.PortConfig{Name: LaAggPortNeighborActor1If,
		HardwareAddr: net.HardwareAddr{0x00, LaAggPort1NeighborActor, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[LaAggPort2NeighborActor] = utils.PortConfig{Name: LaAggPortNeighborActor2If,
		HardwareAddr: net.HardwareAddr{0x00, LaAggPort2NeighborActor, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[LaAggPort1Peer] = utils.PortConfig{Name: LaAggPortPeerIf1,
		HardwareAddr: net.HardwareAddr{0x00, LaAggPort1Peer, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[LaAggPort2Peer] = utils.PortConfig{Name: LaAggPortPeerIf2,
		HardwareAddr: net.HardwareAddr{0x00, LaAggPort2Peer, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[DRNeighborIpp1] = utils.PortConfig{Name: DRNeighborIppIf1,
		HardwareAddr: net.HardwareAddr{0x00, DRNeighborIpp1, 0x11, 0x22, 0x22, 0x33},
	}
	utils.PortConfigMap[DRNeighborIpp2] = utils.PortConfig{Name: DRNeighborIppIf2,
		HardwareAddr: net.HardwareAddr{0x00, DRNeighborIpp2, 0x11, 0x22, 0x22, 0x33},
	}
}

func FullBackToBackConfigTestTeardown() {
	OnlyForTestTeardown()
	delete(utils.PortConfigMap, LaAggPort1NeighborActor)
	delete(utils.PortConfigMap, LaAggPort2NeighborActor)
	delete(utils.PortConfigMap, LaAggPort1Peer)
	delete(utils.PortConfigMap, LaAggPort2Peer)
	delete(utils.PortConfigMap, DRNeighborIpp1)
	delete(utils.PortConfigMap, DRNeighborIpp2)
}

// 3 node system where two neighbors are connected to 1 peer device
func TestCreateBackToBackMLagAndPeer(t *testing.T) {

	FullBackToBackConfigTestSetup()

	// actor1 to actor2
	neighborbridge := SimulationNeighborBridge{
		Port1:      DRNeighborIpp1,
		Port2:      DRNeighborIpp2,
		RxIppPort1: make(chan gopacket.Packet, 10),
		RxIppPort2: make(chan gopacket.Packet, 10),
	}
	ipp1Key := IppDbKey{
		Name:   DRNeighborIppIf1,
		DrName: "DR-1",
	}
	ipp2Key := IppDbKey{
		Name:   DRNeighborIppIf2,
		DrName: "DR-2",
	}

	DRGlobalSystem.DRSystemGlobalRegisterTxCallback(ipp1Key, neighborbridge.TxViaGoChannel)
	DRGlobalSystem.DRSystemGlobalRegisterTxCallback(ipp2Key, neighborbridge.TxViaGoChannel)

	DrRxMain(uint16(DRNeighborIpp1), "00:00:DE:AD:BE:EF", neighborbridge.RxIppPort1)
	DrRxMain(uint16(DRNeighborIpp2), "00:00:DE:AD:BE:EF", neighborbridge.RxIppPort2)

	// Lets create the Distributed Relay
	cfg := &DistrubtedRelayConfig{
		DrniName:                          "DR-1",
		DrniPortalAddress:                 "00:00:DE:AD:BE:EF",
		DrniPortalPriority:                128,
		DrniThreePortalSystem:             false,
		DrniPortalSystemNumber:            1,
		DrniIntraPortalLinkList:           [3]uint32{uint32(DRNeighborIpp1)}, // no link supplied is invalid
		DrniAggregator:                    100,
		DrniGatewayAlgorithm:              "00:80:C2:01",
		DrniNeighborAdminGatewayAlgorithm: "00:80:C2:01",
		DrniNeighborAdminPortAlgorithm:    "10:80:C2:01", // invalid string
		DrniNeighborAdminDRCPState:        "00000000",
		DrniEncapMethod:                   "00:80:C2:01",
		DrniPortConversationControl:       false,
		DrniIntraPortalPortProtocolDA:     "01:80:C2:00:00:03", // only supported value that we are going to support
	}

	// create first drni
	CreateDistributedRelay(cfg)

	// create second drni
	cfg.DrniName = "DR-2"
	cfg.DrniPortalSystemNumber = 2
	cfg.DrniAggregator = 101
	cfg.DrniIntraPortalLinkList = [3]uint32{uint32(DRNeighborIpp2)}
	CreateDistributedRelay(cfg)

	// must be called to initialize the global
	LaSystem1NeighborActor := lacp.LacpSystem{Actor_System_priority: 128,
		Actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x01, 0x64}}
	LaSystem2NeighborActor := lacp.LacpSystem{Actor_System_priority: 128,
		Actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x02, 0x64}}

	LaSystemPeer := lacp.LacpSystem{Actor_System_priority: 128,
		Actor_System: [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0xC8}}

	// actor1 to peer
	bridge1 := lacp.SimulationBridge{
		Port1:       LaAggPort1NeighborActor,
		Port2:       LaAggPort1Peer,
		RxLacpPort1: make(chan gopacket.Packet, 10),
		RxLacpPort2: make(chan gopacket.Packet, 10),
	}
	// actor2 to peer
	bridge2 := lacp.SimulationBridge{
		Port1:       LaAggPort2NeighborActor,
		Port2:       LaAggPort2Peer,
		RxLacpPort1: make(chan gopacket.Packet, 10),
		RxLacpPort2: make(chan gopacket.Packet, 10),
	}

	Actor1System := lacp.LacpSysGlobalInfoInit(LaSystem1NeighborActor)
	Actor2System := lacp.LacpSysGlobalInfoInit(LaSystem2NeighborActor)
	PeerSystem := lacp.LacpSysGlobalInfoInit(LaSystemPeer)
	// la and ipp ports
	Actor1System.LaSysGlobalRegisterTxCallback(LaAggPortNeighborActor1If, bridge1.TxViaGoChannel)
	// la and ipp ports
	Actor2System.LaSysGlobalRegisterTxCallback(LaAggPortNeighborActor2If, bridge2.TxViaGoChannel)
	// la ports
	PeerSystem.LaSysGlobalRegisterTxCallback(LaAggPortPeerIf1, bridge1.TxViaGoChannel)
	PeerSystem.LaSysGlobalRegisterTxCallback(LaAggPortPeerIf2, bridge2.TxViaGoChannel)

	// port 1
	lacp.LaRxMain(bridge1.Port1, bridge1.RxLacpPort1)
	// port 2
	lacp.LaRxMain(bridge1.Port2, bridge1.RxLacpPort2)

	// port 1
	lacp.LaRxMain(bridge2.Port1, bridge2.RxLacpPort1)
	// port 2
	lacp.LaRxMain(bridge2.Port2, bridge2.RxLacpPort2)

	// lag system 1 actor && lag system 2 actor
	a1conf := &lacp.LaAggConfig{
		Mac:  [6]uint8{0x00, 0x00, 0x01, 0x01, 0x01, 0x01},
		Id:   100,
		Name: "agg1",
		Key:  100,
		Lacp: lacp.LacpConfigInfo{Interval: lacp.LacpSlowPeriodicTime,
			Mode:           lacp.LacpModeActive,
			SystemIdMac:    "00:00:00:00:01:64",
			SystemPriority: 128},
	}

	a2conf := &lacp.LaAggConfig{
		Mac:  [6]uint8{0x00, 0x00, 0x02, 0x01, 0x01, 0x01},
		Id:   101,
		Name: "agg2",
		Key:  101,
		Lacp: lacp.LacpConfigInfo{Interval: lacp.LacpSlowPeriodicTime,
			Mode:           lacp.LacpModeActive,
			SystemIdMac:    "00:00:00:00:02:64",
			SystemPriority: 128},
	}

	// lag sytem 3 peer
	a3conf := &lacp.LaAggConfig{
		Mac:  [6]uint8{0x00, 0x00, 0x02, 0x02, 0x02, 0x02},
		Id:   200,
		Name: "agg3",
		Key:  200,
		Lacp: lacp.LacpConfigInfo{Interval: lacp.LacpSlowPeriodicTime,
			Mode:           lacp.LacpModeActive,
			SystemIdMac:    "00:00:00:00:00:C8",
			SystemPriority: 128},
	}

	// Create Aggregation
	lacp.CreateLaAgg(a1conf)
	AttachAggregatorToDistributedRelay(a1conf.Id)
	lacp.CreateLaAgg(a2conf)
	AttachAggregatorToDistributedRelay(a2conf.Id)
	lacp.CreateLaAgg(a3conf)

	// actor        peer
	// p1config <-> p2config
	// p3config <-> p4config
	p1conf := &lacp.LaAggPortConfig{
		Id:     LaAggPort1NeighborActor,
		Prio:   0x80,
		Key:    100,
		AggId:  100,
		Enable: true,
		Mode:   lacp.LacpModeActive,
		//Timeout: LacpFastPeriodicTime,
		Properties: lacp.PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPort1NeighborActor, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: lacp.LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortNeighborActor1If,
		TraceEna: true,
	}
	p2conf := &lacp.LaAggPortConfig{
		Id:     LaAggPort1Peer,
		Prio:   0x80,
		Key:    200,
		AggId:  200,
		Enable: true,
		Mode:   lacp.LacpModeActive,
		Properties: lacp.PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPort1Peer, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: lacp.LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortPeerIf1,
		TraceEna: true,
	}

	p3conf := &lacp.LaAggPortConfig{
		Id:     LaAggPort2NeighborActor,
		Prio:   0x80,
		Key:    101,
		AggId:  101,
		Enable: true,
		Mode:   lacp.LacpModeActive,
		//Timeout: LacpFastPeriodicTime,
		Properties: lacp.PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPort2NeighborActor, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: lacp.LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortNeighborActor2If,
		TraceEna: true,
	}

	p4conf := &lacp.LaAggPortConfig{
		Id:     LaAggPort2Peer,
		Prio:   0x80,
		Key:    200,
		AggId:  200,
		Enable: true,
		Mode:   lacp.LacpModeActive,
		Properties: lacp.PortProperties{
			Mac:    net.HardwareAddr{0x00, LaAggPort2Peer, 0xDE, 0xAD, 0xBE, 0xEF},
			Speed:  1000000000,
			Duplex: lacp.LacpPortDuplexFull,
			Mtu:    1500,
		},
		IntfId:   LaAggPortPeerIf2,
		TraceEna: true,
	}

	// actor / neighbor
	lacp.CreateLaAggPort(p1conf)
	UpdateAggregatorPortList(p1conf.AggId)
	// peer
	lacp.CreateLaAggPort(p2conf)

	// actor / neighbor
	lacp.CreateLaAggPort(p3conf)
	UpdateAggregatorPortList(p3conf.AggId)
	// peer
	lacp.CreateLaAggPort(p4conf)

	//time.Sleep(time.Second * 30)
	testWait := make(chan bool)

	var p1 *lacp.LaAggPort
	var p2 *lacp.LaAggPort
	// TODO this should fail as the ports should not sync up with the peer because the agg key does not agree between
	// the ports
	if lacp.LaFindPortById(p1conf.Id, &p1) &&
		lacp.LaFindPortById(p2conf.Id, &p2) {

		go func() {
			for i := 0; i < 10 &&
				(p1.MuxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpMuxmStateDistributing ||
					p2.MuxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpMuxmStateDistributing); i++ {
				time.Sleep(time.Second * 1)
			}
			testWait <- true
		}()

		<-testWait
		close(testWait)

		State1 := lacp.GetLaAggPortActorOperState(p1conf.Id)
		State2 := lacp.GetLaAggPortActorOperState(p2conf.Id)

		const portUpState = lacp.LacpStateActivityBit | lacp.LacpStateAggregationBit |
			lacp.LacpStateSyncBit | lacp.LacpStateCollectingBit | lacp.LacpStateDistributingBit

		if !lacp.LacpStateIsSet(State1, portUpState) {
			t.Error(fmt.Sprintf("Actor Port State 0x%x did not come up properly with peer expected 0x%x", State1, portUpState))
		}
		if !lacp.LacpStateIsSet(State2, portUpState) {
			t.Error(fmt.Sprintf("Peer Port State 0x%x did not come up properly with actor expected 0x%x", State2, portUpState))
		}

		// TODO check the States of the other State machines
	} else {
		t.Error("Unable to find port just created")
	}

	testWait = make(chan bool)
	// TODO this should fail as the ports should not sync up with the peer because the agg key does not agree between
	// the ports
	if lacp.LaFindPortById(p2conf.Id, &p1) &&
		lacp.LaFindPortById(p3conf.Id, &p2) {

		go func() {
			for i := 0; i < 10 &&
				(p1.MuxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpMuxmStateDistributing ||
					p2.MuxMachineFsm.Machine.Curr.CurrentState() != lacp.LacpMuxmStateDistributing); i++ {
				time.Sleep(time.Second * 1)
			}
			testWait <- true
		}()

		<-testWait
		close(testWait)

		State1 := lacp.GetLaAggPortActorOperState(p3conf.Id)
		State2 := lacp.GetLaAggPortActorOperState(p4conf.Id)

		const portUpState = lacp.LacpStateActivityBit | lacp.LacpStateAggregationBit |
			lacp.LacpStateSyncBit | lacp.LacpStateCollectingBit | lacp.LacpStateDistributingBit

		if !lacp.LacpStateIsSet(State1, portUpState) {
			t.Error(fmt.Sprintf("Actor Port State %s did not come up properly with peer expected %s", lacp.LacpStateToStr(State1), lacp.LacpStateToStr(portUpState)))
		}
		if !lacp.LacpStateIsSet(State2, portUpState) {
			t.Error(fmt.Sprintf("Peer Port State %s did not come up properly with actor expected %s", lacp.LacpStateToStr(State2), lacp.LacpStateToStr(portUpState)))
		}

		// TODO check the States of the other State machines
	} else {
		t.Error("Unable to find port just created")
	}

	// cleanup the provisioning
	close(bridge1.RxLacpPort1)
	close(bridge1.RxLacpPort2)
	close(bridge2.RxLacpPort1)
	close(bridge2.RxLacpPort2)
	bridge1.RxLacpPort1 = nil
	bridge1.RxLacpPort2 = nil
	bridge2.RxLacpPort1 = nil
	bridge2.RxLacpPort2 = nil
	lacp.DeleteLaAgg(a1conf.Id)
	lacp.DeleteLaAgg(a2conf.Id)
	lacp.DeleteLaAgg(a3conf.Id)
	for _, sgi := range lacp.LacpSysGlobalInfoGet() {
		if len(sgi.AggList) > 0 || len(sgi.AggMap) > 0 {
			t.Error("System Agg List or Map is not empty", sgi.AggList, sgi.AggMap)
		}
		if len(sgi.PortList) > 0 || len(sgi.PortMap) > 0 {
			t.Error("System Port List or Map is not empty", sgi.PortList, sgi.PortMap)
		}
	}
	lacp.LacpSysGlobalInfoDestroy(LaSystem1NeighborActor)
	lacp.LacpSysGlobalInfoDestroy(LaSystem2NeighborActor)
	lacp.LacpSysGlobalInfoDestroy(LaSystemPeer)
	FullBackToBackConfigTestTeardown()
}
