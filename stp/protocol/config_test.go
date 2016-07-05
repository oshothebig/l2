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

package stp

import (
	"net"
	"testing"
	"time"
)

func StpBridgeConfigSetup() *StpBridgeConfig {
	brg := &StpBridgeConfig{
		Address:      "00:11:22:33:44:55",
		Priority:     32768,
		MaxAge:       10,
		HelloTime:    1,
		ForwardDelay: 6,
		ForceVersion: 2, // RSTP
		TxHoldCount:  2,
		Vlan:         1,
	}
	return brg
}

func StpPortConfigSetup(createbridge, update bool) (*StpPortConfig, *StpBridgeConfig) {
	var brg *StpBridgeConfig
	if createbridge {
		brg = StpBridgeConfigSetup()
		// bridge must exist
		StpBridgeCreate(brg)
	}

	p := &StpPortConfig{
		IfIndex:           1,
		Priority:          128,
		Enable:            true,
		PathCost:          200000,
		ProtocolMigration: 1,
		AdminPointToPoint: 0,
		AdminEdgePort:     false,
		AdminPathCost:     200000,
		BrgIfIndex:        1,
		BridgeAssurance:   false,
		BpduGuard:         false,
		BpduGuardInterval: 0,
	}

	PortConfigMap[p.IfIndex] = portConfig{Name: "lo",
		HardwareAddr: net.HardwareAddr{0x00, 0x11, 0x11, 0x22, 0x22, 0x33},
	}

	return p, brg
}

func TestStpBridgeCreationDeletion(t *testing.T) {
	// when creating a bridge it is recommended that the following calls are made

	// this is specific to test but config object should be filled in
	brgcfg := StpBridgeConfigSetup()
	// verify the paramaters are correct
	err := StpBrgConfigParamCheck(brgcfg)
	if err != nil {
		t.Error("ERROR valid config failed", err)
	}
	// create the bridge
	err = StpBridgeCreate(brgcfg)
	if err != nil {
		t.Error("ERROR valid creation failed", err)
	}
	// delete the bridge
	// config cleanup is done as part of bridge delete
	err = StpBridgeDelete(brgcfg)
	if err != nil {
		t.Error("ERROR valid deletion failed", err)
	}
}

func TestStpPortCreationDeletion(t *testing.T) {
	// when creating a bridge it is recommended that the following calls are made

	// this is specific to test but config object should be filled in
	brgcfg := StpBridgeConfigSetup()
	// verify the paramaters are correct
	err := StpBrgConfigParamCheck(brgcfg)
	if err != nil {
		t.Error("ERROR valid brg config failed", err)
	}
	// create the bridge
	err = StpBridgeCreate(brgcfg)
	if err != nil {
		t.Error("ERROR valid brg creation failed", err)
	}

	// configure valid stp port
	pcfg, _ := StpPortConfigSetup(false, false)
	pcfg.BrgIfIndex = 1
	pcfg.BridgeAssurance = true

	// valid config
	err = StpPortConfigParamCheck(pcfg, false)
	if err != nil {
		t.Error("ERROR valid stp port config failed", err)
	}

	// create the stp port
	err = StpPortCreate(pcfg)
	if err != nil {
		t.Error("ERROR valid stp port creation failed", err)
	}

	var p *StpPort
	if !StpFindPortByIfIndex(pcfg.IfIndex, pcfg.BrgIfIndex, &p) {
		t.Errorf("ERROR unable to find bridge port that was just created", err)

	}
	waitChan := make(chan bool)
	go func() {
		for i := 0; i < 30; i++ {
			if p.Forwarding == false || p.Learning == false {
				time.Sleep(time.Second * 1)
			}
		}
		waitChan <- true
	}()
	<-waitChan

	if p.Forwarding == false || p.Learning == false {
		t.Error("ERROR Bridge Port did not come up in a defaulted learning/forwarding state")
	}

	// delete the stp port
	err = StpPortDelete(pcfg)
	if err != nil {
		t.Error("ERROR valid stp port deletion failed", err)
	}

	if StpFindPortByIfIndex(pcfg.IfIndex, pcfg.BrgIfIndex, &p) {
		t.Errorf("ERROR found bridge port that was just deleted", err)

	}

	// delete the bridge
	// config cleanup is done as part of bridge delete
	err = StpBridgeDelete(brgcfg)
	if err != nil {
		t.Error("ERROR valid deletion failed", err)
	}
}

func TestStpPortAdminEdgeCreationDeletion(t *testing.T) {
	// when creating a bridge it is recommended that the following calls are made

	// this is specific to test but config object should be filled in
	brgcfg := StpBridgeConfigSetup()
	// verify the paramaters are correct
	err := StpBrgConfigParamCheck(brgcfg)
	if err != nil {
		t.Error("ERROR valid brg config failed", err)
	}
	// create the bridge
	err = StpBridgeCreate(brgcfg)
	if err != nil {
		t.Error("ERROR valid brg creation failed", err)
	}

	// configure valid stp port
	pcfg, _ := StpPortConfigSetup(false, false)
	pcfg.BrgIfIndex = 1
	pcfg.AdminEdgePort = true
	pcfg.BpduGuard = true

	// valid config
	err = StpPortConfigParamCheck(pcfg, false)
	if err != nil {
		t.Error("ERROR valid stp port config failed", err)
	}

	// create the stp port
	err = StpPortCreate(pcfg)
	if err != nil {
		t.Error("ERROR valid stp port creation failed", err)
	}

	var p *StpPort
	if !StpFindPortByIfIndex(pcfg.IfIndex, pcfg.BrgIfIndex, &p) {
		t.Errorf("ERROR unable to find bridge port that was just created", err)

	}
	waitChan := make(chan bool)
	go func() {
		for i := 0; i < 30; i++ {
			if p.Forwarding == false || p.Learning == false {
				time.Sleep(time.Second * 1)
			}
		}
		waitChan <- true
	}()
	<-waitChan

	if p.Forwarding == false || p.Learning == false {
		t.Error("ERROR Bridge Port did not come up in a defaulted learning/forwarding state")
	}

	// delete the stp port
	err = StpPortDelete(pcfg)
	if err != nil {
		t.Error("ERROR valid stp port deletion failed", err)
	}

	if StpFindPortByIfIndex(pcfg.IfIndex, pcfg.BrgIfIndex, &p) {
		t.Errorf("ERROR found bridge port that was just deleted", err)

	}

	// delete the bridge
	// config cleanup is done as part of bridge delete
	err = StpBridgeDelete(brgcfg)
	if err != nil {
		t.Error("ERROR valid deletion failed", err)
	}
}

func TestStpBridgeParamCheckPriority(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.Priority = 11111
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid priority was set should have errored", brgcfg.Priority)
	}

	// now lets send a good value according to table 802.1D 17-2
	// 0 - 61440 in increments of 4096
	for i := uint16(0); i <= 61440/4096; i++ {
		brgcfg.Priority = 4096 * i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid priority was set should not have errored", brgcfg.Priority)
		}
	}
}

func TestStpBridgeParamCheckMaxAge(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.MaxAge = 200
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid max age was set should have errored", brgcfg.MaxAge)
	}
	// set bad value
	brgcfg.MaxAge = 5
	err = StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid max age was set should have errored", brgcfg.MaxAge)
	}

	// now lets send all possible values according to table 802.1D 17-1
	for i := uint16(6); i <= 40; i++ {
		brgcfg.MaxAge = i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid max age was set should not have errored", brgcfg.MaxAge)
		}
	}
}

func TestStpBridgeParamCheckHelloTime(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.HelloTime = 0
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid hello time was set should have errored", brgcfg.HelloTime)
	}
	// set bad value
	brgcfg.HelloTime = 5
	err = StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid hello time was set should have errored", brgcfg.HelloTime)
	}

	// now lets send all possible values according to table 802.1D 17-1
	for i := uint16(1); i <= 2; i++ {
		brgcfg.HelloTime = i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid hello time was set should not have errored", brgcfg.HelloTime)
		}
	}
}

func TestStpBridgeParamCheckFowardingDelay(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.ForwardDelay = 0
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid forwarding delay was set should have errored", brgcfg.ForwardDelay)
	}
	// set bad value
	brgcfg.ForwardDelay = 50
	err = StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid forwardng delay was set should have errored", brgcfg.ForwardDelay)
	}

	// now lets send all possible values according to table 802.1D 17-1
	for i := uint16(4); i <= 30; i++ {
		brgcfg.ForwardDelay = i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid forwarding delay was set should not have errored", brgcfg.ForwardDelay)
		}
	}
}

func TestStpBridgeParamCheckTxHoldCount(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.TxHoldCount = 0
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid tx hold count was set should have errored", brgcfg.TxHoldCount)
	}
	// set bad value
	brgcfg.TxHoldCount = 50
	err = StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid tx hold count was set should have errored", brgcfg.TxHoldCount)
	}

	// now lets send all possible values according to table 802.1D 17-1
	for i := int32(1); i <= 10; i++ {
		brgcfg.TxHoldCount = i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid tx hold count was set should not have errored", brgcfg.TxHoldCount)
		}
	}
}

func TestStpBridgeParamCheckVlan(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.Vlan = 4095
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid vlan was set should have errored", brgcfg.Vlan)
	}

	// now lets send all possible values according to table 802.1Q
	for i := uint16(1); i <= 4094; i++ {
		brgcfg.Vlan = i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid vlan was set should not have errored", brgcfg.Vlan)
		}
	}
}

func TestStpBridgeParamCheckForceVersion(t *testing.T) {

	// setup
	brgcfg := StpBridgeConfigSetup()

	// set bad value
	brgcfg.ForceVersion = 0
	err := StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid force version was set should have errored", brgcfg.ForceVersion)
	}

	brgcfg.ForceVersion = 3
	err = StpBrgConfigParamCheck(brgcfg)
	if err == nil {
		t.Error("ERROR an invalid force version was set should have errored", brgcfg.ForceVersion)
	}

	// now lets send all possible values according to 802.1D
	for i := int32(1); i <= 2; i++ {
		brgcfg.ForceVersion = i
		err = StpBrgConfigParamCheck(brgcfg)
		if err != nil {
			t.Error("ERROR valid force version was set should not have errored", brgcfg.ForceVersion)
		}
	}
}

/*
type StpPortConfig struct {
	IfIndex           int32
	Priority          uint16
	Enable            bool
	PathCost          int32
	ProtocolMigration int32
	AdminPointToPoint int32
	AdminEdgePort     bool
	AdminPathCost     int32
	BrgIfIndex        int32
	BridgeAssurance   bool
	BpduGuard         bool
	BpduGuardInterval int32
}
*/

func TestStpPortParamIfIndex(t *testing.T) {

	p, b := StpPortConfigSetup(true, false)
	defer StpPortConfigDelete(p.IfIndex)
	if b != nil {
		defer StpBridgeDelete(b)
	}
	// invalid value
	p.IfIndex = 0
	err := StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid ifindex was set should have errored", p.IfIndex, err)
	}
}

func TestStpPortParamBrgIfIndex(t *testing.T) {
	p, b := StpPortConfigSetup(false, false)
	defer StpPortConfigDelete(p.IfIndex)

	// port created without bridge being created
	err := StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid brgIfndex was set should have errored", p.BrgIfIndex, err)
	}

	// port created with invalid bridge
	p, b = StpPortConfigSetup(true, false)
	if b != nil {
		defer StpBridgeDelete(b)
	}
	p.BrgIfIndex = 1000
	err = StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid brgIfndex was set should have errored", p.BrgIfIndex, err)
	}

	p, b = StpPortConfigSetup(true, false)
	err = StpPortConfigParamCheck(p, true)
	if err != nil {
		t.Error("ERROR: an invalid brgIfndex was not set should have errored", p.BrgIfIndex, err)
	}
}

func TestStpPortParamPriority(t *testing.T) {

	p, b := StpPortConfigSetup(true, false)
	defer StpPortConfigDelete(p.IfIndex)
	if b != nil {
		defer StpBridgeDelete(b)
	}

	// invalid value
	p.Priority = 241
	err := StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid priority was set should have errored", p.Priority)
	}
	p.Priority = 1
	err = StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid priority was set should have errored", p.Priority, err)
	}

	// valid values according to 802.1D table 17-2
	for i := uint16(0); i <= 240/16; i++ {
		p.Priority = 16 * i
		err := StpPortConfigParamCheck(p, true)
		if err != nil {
			t.Error("ERROR: valid priority was set should not have errored", p.Priority, err)
		}
	}

	// lets pretend another bridge port is being created and Priority is different
	p.BrgIfIndex = 100
	p.Priority = 16
	err = StpPortConfigParamCheck(p, false)
	if err == nil {
		t.Error("ERROR: an invalid port config change priority was set should have errored", p.Priority, err)
	}
}

func TestStpPortParamAdminPathCost(t *testing.T) {

	p, b := StpPortConfigSetup(true, false)
	defer StpPortConfigDelete(p.IfIndex)
	if b != nil {
		defer StpBridgeDelete(b)
	}

	// invalid value
	p.AdminPathCost = 200000001
	err := StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid admin path cost was set should have errored", p.AdminPathCost, err)
	}
	p.AdminPathCost = 0
	err = StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid admin path cost was set should have errored", p.AdminPathCost, err)
	}

	// valid recommended values according to 802.1D table 17-3
	// technically the range is 1-2000000000
	for i := int32(1); i <= 9; i++ {
		p.AdminPathCost = 2 * (i * 10)
		err := StpPortConfigParamCheck(p, true)
		if err != nil {
			t.Error("ERROR: valid admin path cost was set should not have errored", p.AdminPathCost, err)
		}
	}

	// lets pretend another bridge port is being created and admin path cost is different
	p.BrgIfIndex = 100
	p.AdminPathCost = 200
	err = StpPortConfigParamCheck(p, false)
	if err == nil {
		t.Error("ERROR: an invalid port config change admin path cost was set should have errored", p.AdminPathCost, err)
	}
}

func TestStpPortParamBridgeAssurance(t *testing.T) {
	p, b := StpPortConfigSetup(true, false)
	defer StpPortConfigDelete(p.IfIndex)
	if b != nil {
		defer StpBridgeDelete(b)
	}

	// bridge assurance is only valid on an non edge port
	p.AdminEdgePort = true
	p.BridgeAssurance = true
	err := StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: an invalid port config Admin Edge and Bridge Assurance set should have errored", p.AdminEdgePort, p.BridgeAssurance, err)
	}

	// bridge assurance is only valid on an non edge port
	p.AdminEdgePort = false
	p.BridgeAssurance = true
	err = StpPortConfigParamCheck(p, true)
	if err != nil {
		t.Error("ERROR: valid port config Bridge Assurance set should not have errored", p.AdminEdgePort, p.BridgeAssurance, err)
	}

	// lets save the config
	StpPortConfigSave(p, false)
	ifIndex := p.IfIndex
	brgIfIndex := p.BrgIfIndex
	// lets update Admin Edge when Bridge Assurance is already enabled
	p = StpPortConfigGet(ifIndex)
	if p == nil {
		t.Error("ERROR: could not find port config")
	}
	p.IfIndex = ifIndex
	p.BrgIfIndex = brgIfIndex
	p.AdminEdgePort = true
	err = StpPortConfigParamCheck(p, true)
	if err == nil {
		t.Error("ERROR: invalid port config Bridge Assurance set should have errored", p.AdminEdgePort, p.BridgeAssurance, err)
	}

}
