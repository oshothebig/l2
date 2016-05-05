package lldpServer

import (
	"asicdServices"
	"fmt"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"l2/lldp/utils"
	"net"
	"sync"
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
func (gblInfo *LLDPGlobalInfo) InitRuntimeInfo(portConf *asicdServices.PortState) {
	gblInfo.IfIndex = portConf.IfIndex
	gblInfo.Name = portConf.Name
	gblInfo.OperState = portConf.OperState
	gblInfo.PortNum = portConf.PortNum
	gblInfo.OperStateLock = &sync.RWMutex{}
	gblInfo.PcapHdlLock = &sync.RWMutex{}
	gblInfo.useCacheFrame = false
	gblInfo.SetTxInterval(LLDP_DEFAULT_TX_INTERVAL)
	gblInfo.SetTxHoldMultiplier(LLDP_DEFAULT_TX_HOLD_MULTIPLIER)
	gblInfo.SetTTL()
	gblInfo.SetDstMac()
}

/*  updating l2 port information with mac address. If needed update other
 *  information also in future
 */
func (gblInfo *LLDPGlobalInfo) UpdatePortInfo(portConf *asicdServices.Port) {
	gblInfo.MacAddr = portConf.MacAddr
}

/*  De-Init l2 port information
 */
func (gblInfo *LLDPGlobalInfo) DeInitRuntimeInfo() {
	gblInfo.StopCacheTimer()
	gblInfo.DeletePcapHandler()
	gblInfo.FreeDynamicMemory()
}

/*  Set TTL Value at the time of init or update of lldp config
 *  default value comes out to be 120
 */
func (gblInfo *LLDPGlobalInfo) SetTTL() {
	gblInfo.ttl = Min(LLDP_MAX_TTL, (gblInfo.lldpMessageTxInterval *
		gblInfo.lldpMessageTxHoldMultiplier))
}

/*  Set tx interval during init or update
 *  default value is 30
 */
func (gblInfo *LLDPGlobalInfo) SetTxInterval(interval int) {
	gblInfo.lldpMessageTxInterval = interval
}

/*  Set tx hold multiplier during init or update
 *  default value is 4
 */
func (gblInfo *LLDPGlobalInfo) SetTxHoldMultiplier(hold int) {
	gblInfo.lldpMessageTxHoldMultiplier = hold
}

/*  Set DstMac as lldp protocol mac address
 */
func (gblInfo *LLDPGlobalInfo) SetDstMac() {
	var err error
	gblInfo.DstMAC, err = net.ParseMAC(LLDP_PROTO_DST_MAC)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("parsing lldp protocol mac failed",
			err))
	}
}

/*  Delete l2 port pcap handler
 */
func (gblInfo *LLDPGlobalInfo) DeletePcapHandler() {
	gblInfo.PcapHdlLock.Lock()
	if gblInfo.PcapHandle != nil {
		// @FIXME: some bug in close handling that causes 5 mins delay
		//gblInfo.PcapHandle.Close()
		gblInfo.PcapHandle = nil
	}
	gblInfo.PcapHdlLock.Unlock()
}

/*  Stop Tx timer... as we have already delete the pcap handle
 */
func (gblInfo *LLDPGlobalInfo) StopTxTimer() {
	if gblInfo.txTimer != nil {
		gblInfo.txTimer.Stop()
	}
}

/*  Stop RX cache timer
 */
func (gblInfo *LLDPGlobalInfo) StopCacheTimer() {
	if gblInfo.clearCacheTimer == nil {
		return
	}
	gblInfo.clearCacheTimer.Stop()
}

/*  We have deleted the pcap handler and hence we will invalid the cache buffer
 */
func (gblInfo *LLDPGlobalInfo) DeleteCacheFrame() {
	gblInfo.useCacheFrame = false
	gblInfo.cacheFrame = nil
}

/*  Return back all the memory which was allocated using new
 */
func (gblInfo *LLDPGlobalInfo) FreeDynamicMemory() {
	gblInfo.rxFrame = nil
	gblInfo.rxLinkInfo = nil
	gblInfo.OperStateLock = nil
	gblInfo.PcapHdlLock = nil
}

/*  Create Pcap Handler
 */
func (gblInfo *LLDPGlobalInfo) CreatePcapHandler(lldpSnapshotLen int32,
	lldpPromiscuous bool, lldpTimeout time.Duration) {
	gblInfo.PcapHdlLock.RLock()
	if gblInfo.PcapHandle != nil {
		gblInfo.PcapHdlLock.RUnlock()
		debug.Logger.Alert("Pcap already exists and create pcap called for " +
			gblInfo.Name)
		return
	}
	gblInfo.PcapHdlLock.RUnlock()
	pcapHdl, err := pcap.OpenLive(gblInfo.Name, lldpSnapshotLen,
		lldpPromiscuous, lldpTimeout)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Creating Pcap Handler failed for",
			gblInfo.Name, "Error:", err))
	}
	err = pcapHdl.SetBPFFilter(LLDP_BPF_FILTER)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("setting filter", LLDP_BPF_FILTER,
			"for", gblInfo.Name, "failed with error:", err))
	}
	gblInfo.PcapHdlLock.Lock()
	gblInfo.PcapHandle = pcapHdl
	gblInfo.PcapHdlLock.Unlock()
}

/*  Get Chassis Id info
 *	 Based on SubType Return the string, mac address then form string using
 *	 net package
 */
func (gblInfo *LLDPGlobalInfo) GetChassisIdInfo() string {

	retVal := ""

	switch gblInfo.rxFrame.ChassisID.Subtype {
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
		mac = gblInfo.rxFrame.ChassisID.ID
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

	switch gblInfo.rxFrame.PortID.Subtype {
	case layers.LLDPPortIDSubtypeReserved:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeIfaceAlias:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypePortComp:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeMACAddr:
		var mac net.HardwareAddr
		mac = gblInfo.rxFrame.ChassisID.ID
		return mac.String()
	case layers.LLDPPortIDSubtypeNetworkAddr:
		debug.Logger.Debug("Need to handle this case")
	case layers.LLDPPortIDSubtypeIfaceName:
		return string(gblInfo.rxFrame.PortID.ID)
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
	debug.Logger.Info(fmt.Sprintln("L2 Port:", gblInfo.Name, "Port Num:",
		gblInfo.PortNum))
	debug.Logger.Info(fmt.Sprintln("SrcMAC:", gblInfo.SrcMAC.String(),
		"DstMAC:", gblInfo.DstMAC.String()))
	debug.Logger.Info(fmt.Sprintln("ChassisID info is",
		gblInfo.rxFrame.ChassisID))
	debug.Logger.Info(fmt.Sprintln("PortID info is",
		gblInfo.rxFrame.PortID))
	debug.Logger.Info(fmt.Sprintln("TTL info is", gblInfo.rxFrame.TTL))
	debug.Logger.Info(fmt.Sprintln("Optional Values is",
		gblInfo.rxLinkInfo))
}
