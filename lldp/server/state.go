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

package server

import (
	"fmt"
	"l2/lldp/config"
	"l2/lldp/utils"
	"strconv"
)

/*  helper function to convert TLV's (chassisID, portID, TTL) from byte
 *  format to string
 */
func (svr *LLDPServer) PopulateTLV(ifIndex int32, entry *config.IntfState) bool {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Err(fmt.Sprintln("Entry not found for", ifIndex))
		return exists
	}
	gblInfo.RxLock.RLock()
	defer gblInfo.RxLock.RUnlock()
	entry.LocalPort = gblInfo.Port.Name
	if gblInfo.RxInfo.RxFrame != nil {
		entry.PeerMac = gblInfo.GetChassisIdInfo()
		entry.PeerPort = gblInfo.GetPortIdInfo()
		entry.HoldTime = strconv.Itoa(int(gblInfo.RxInfo.RxFrame.TTL))
	}

	if gblInfo.RxInfo.RxLinkInfo != nil {
		entry.SystemCapabilities = gblInfo.GetSystemCap()
		entry.EnabledCapabilities = gblInfo.GetEnabledCap()
		entry.PeerHostName = gblInfo.GetPeerHostName()
		entry.SystemDescription = gblInfo.GetSystemDescription()
	}

	entry.IfIndex = gblInfo.Port.IfIndex
	entry.Enable = gblInfo.enable
	entry.IntfRef = gblInfo.Port.Name
	return exists
}

/*  Server get bulk for lldp up intfs. This is used for Auto-Discovery
 */
func (svr *LLDPServer) GetIntfs(idx, cnt int) (int, int, []config.Intf) {
	var nextIdx int
	var count int

	if svr.lldpIntfStateSlice == nil {
		debug.Logger.Info("No neighbor learned")
		return 0, 0, nil
	}
	length := len(svr.lldpIntfStateSlice)
	result := make([]config.Intf, cnt)
	var i, j int
	for i, j = 0, idx; i < cnt && j < length; {
		key := svr.lldpIntfStateSlice[j]
		gblInfo, exists := svr.lldpGblInfo[key]
		if exists {
			//result[i].IfIndex = gblInfo.Port.IfIndex
			result[i].IntfRef = gblInfo.Port.Name
			result[i].Enable = gblInfo.enable
			i++
			j++
		}
	}
	if j == length {
		nextIdx = 0
	}
	count = i

	return nextIdx, count, result
}

/*  Server get bulk for lldp up intf state's
 */
func (svr *LLDPServer) GetIntfStates(idx, cnt int) (int, int, []config.IntfState) {
	var nextIdx int
	var count int

	if svr.lldpIntfStateSlice == nil {
		debug.Logger.Info("No neighbor learned")
		return 0, 0, nil
	}

	length := len(svr.lldpUpIntfStateSlice)
	result := make([]config.IntfState, cnt)

	var i, j int

	for i, j = 0, idx; i < cnt && j < length; j++ {
		key := svr.lldpUpIntfStateSlice[j]
		succes := svr.PopulateTLV(key, &result[i])
		if !succes {
			result = nil
			return 0, 0, nil
		}
		i++
	}

	if j == length {
		nextIdx = 0
	}
	count = i
	return nextIdx, count, result
}

/*  Server get lldp interface state per interface
 */
func (svr *LLDPServer) GetIntfState(intfRef string) *config.IntfState {
	entry := config.IntfState{}
	ifIndex, exists := svr.lldpIntfRef2IfIndexMap[intfRef]
	if !exists {
		return &entry
	}

	success := svr.PopulateTLV(ifIndex, &entry)
	if success {
		return &entry
	}
	return nil
}
