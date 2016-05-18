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
	"errors"
	"fmt"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"l2/lldp/config"
	"l2/lldp/packet"
	"l2/lldp/utils"
	"models"
	"net"
	"time"
)

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

/*  Init l2 port information for global runtime information
 */
func (gblInfo *LLDPGlobalInfo) InitRuntimeInfo(portConf *config.PortInfo) {
	gblInfo.Port = *portConf
	gblInfo.RxInfo = packet.RxInit()
	gblInfo.TxInfo = packet.TxInit(LLDP_DEFAULT_TX_INTERVAL,
		LLDP_DEFAULT_TX_HOLD_MULTIPLIER)
}

/*  De-Init l2 port information
 */
func (gblInfo *LLDPGlobalInfo) DeInitRuntimeInfo() {
	gblInfo.StopCacheTimer()
	gblInfo.DeletePcapHandler()
	gblInfo.FreeDynamicMemory()
}

/*  Delete l2 port pcap handler
 */
func (gblInfo *LLDPGlobalInfo) DeletePcapHandler() {
	if gblInfo.PcapHandle != nil {
		// @FIXME: some bug in close handling that causes 5 mins delay
		//gblInfo.PcapHandle.Close()
		gblInfo.PcapHandle = nil
	}
}

/*  Stop RX cache timer
 */
func (gblInfo *LLDPGlobalInfo) StopCacheTimer() {
	if gblInfo.RxInfo.ClearCacheTimer == nil {
		return
	}
	gblInfo.RxInfo.ClearCacheTimer.Stop()
}

/*  Return back all the memory which was allocated using new
 */
func (gblInfo *LLDPGlobalInfo) FreeDynamicMemory() {
	gblInfo.RxInfo.RxFrame = nil
	gblInfo.RxInfo.RxLinkInfo = nil
}

/*  Create Pcap Handler
 */
func (gblInfo *LLDPGlobalInfo) CreatePcapHandler(lldpSnapshotLen int32,
	lldpPromiscuous bool, lldpTimeout time.Duration) error {
	pcapHdl, err := pcap.OpenLive(gblInfo.Port.Name, lldpSnapshotLen,
		lldpPromiscuous, lldpTimeout)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Creating Pcap Handler failed for",
			gblInfo.Port.Name, "Error:", err))
		return errors.New("Creating Pcap Failed")
	}
	err = pcapHdl.SetBPFFilter(LLDP_BPF_FILTER)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("setting filter", LLDP_BPF_FILTER,
			"for", gblInfo.Port.Name, "failed with error:", err))
		return errors.New("Setting BPF Filter Failed")
	}
	gblInfo.PcapHandle = pcapHdl
	return nil
}

/*  Get Chassis Id info
 *	 Based on SubType Return the string, mac address then form string using
 *	 net package
 */
func (gblInfo *LLDPGlobalInfo) GetChassisIdInfo() string {

	retVal := ""
	switch gblInfo.RxInfo.RxFrame.ChassisID.Subtype {
	case layers.LLDPChassisIDSubTypeReserved:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPChassisIDSubTypeChassisComp:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPChassisIDSubtypeIfaceAlias:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPChassisIDSubTypePortComp:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPChassisIDSubTypeMACAddr:
		var mac net.HardwareAddr
		mac = gblInfo.RxInfo.RxFrame.ChassisID.ID
		return mac.String()
	case layers.LLDPChassisIDSubTypeNetworkAddr:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPChassisIDSubtypeIfaceName:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPChassisIDSubTypeLocal:
		debug.Logger.Debug("Need to handle this case")
	default:
		return retVal

	}
	return retVal
}

/*  Get Port Id info
 *	 Based on SubType Return the string, mac address then form string using
 *	 net package
 */
func (gblInfo *LLDPGlobalInfo) GetPortIdInfo() string {

	retVal := ""
	switch gblInfo.RxInfo.RxFrame.PortID.Subtype {
	case layers.LLDPPortIDSubtypeReserved:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeIfaceAlias:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypePortComp:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeMACAddr:
		var mac net.HardwareAddr
		mac = gblInfo.RxInfo.RxFrame.ChassisID.ID
		return mac.String()
	case layers.LLDPPortIDSubtypeNetworkAddr:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeIfaceName:
		return string(gblInfo.RxInfo.RxFrame.PortID.ID)
	case layers.LLDPPortIDSubtypeAgentCircuitID:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeLocal:
		debug.Logger.Debug("Need to handle this case")
	default:
		return retVal

	}
	return retVal
}

/*  dump received lldp frame and other TX information
 */
func (gblInfo LLDPGlobalInfo) DumpFrame() {
	debug.Logger.Info(fmt.Sprintln("L2 Port:", gblInfo.Port.Name, "Port Num:",
		gblInfo.Port.PortNum))
	debug.Logger.Info(fmt.Sprintln("SrcMAC:", gblInfo.RxInfo.SrcMAC.String(),
		"DstMAC:", gblInfo.RxInfo.DstMAC.String()))
	debug.Logger.Info(fmt.Sprintln("ChassisID info is",
		gblInfo.RxInfo.RxFrame.ChassisID))
	debug.Logger.Info(fmt.Sprintln("PortID info is",
		gblInfo.RxInfo.RxFrame.PortID))
	debug.Logger.Info(fmt.Sprintln("TTL info is", gblInfo.RxInfo.RxFrame.TTL))
	debug.Logger.Info(fmt.Sprintln("Optional Values is",
		gblInfo.RxInfo.RxLinkInfo))
}

/*  Api used to get entry.. This is mainly used by LLDP Server API Layer when it get config from
 *  North Bound Plugin...
 */
func (svr *LLDPServer) EntryExist(ifIndex int32) bool {
	_, exists := svr.lldpGblInfo[ifIndex]
	return exists
}

/*  Api to get System information used for TX Frame
 */
func (svr *LLDPServer) GetSystemInfo() {
	if svr.SysInfo != nil {
		return
	}
	svr.SysInfo = &models.SystemParams{}
	debug.Logger.Info("Reading System Information From Db")
	dbHdl := svr.lldpDbHdl
	if dbHdl != nil {
		var dbObj models.SystemParams
		objList, err := dbHdl.GetAllObjFromDb(dbObj)
		if err != nil {
			debug.Logger.Err("DB query failed for System Info")
			return
		}
		for idx := 0; idx < len(objList); idx++ {
			dbObject := objList[idx].(models.SystemParams)
			svr.SysInfo.SwitchMac = dbObject.SwitchMac
			svr.SysInfo.RouterId = dbObject.RouterId
			svr.SysInfo.MgmtIp = dbObject.MgmtIp
			svr.SysInfo.Version = dbObject.Version
			svr.SysInfo.Description = dbObject.Description
			svr.SysInfo.Hostname = dbObject.Hostname
			svr.SysInfo.Vrf = dbObject.Vrf
			break
		}
	}
	debug.Logger.Info(fmt.Sprintln("reading system info from db done", svr.SysInfo))
	return
}

/*  Api to update system cache on next send frame
 */
func (svr *LLDPServer) UpdateSystemCache() {
	svr.SysInfo = nil
	for _, ifIndex := range svr.lldpUpIntfStateSlice {
		gblInfo, exists := svr.lldpGblInfo[ifIndex]
		if !exists {
			continue
		}
		gblInfo.TxInfo.SetCache(false)
	}
}
