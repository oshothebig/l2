package lldpServer

import (
	"asicdServices"
	"fmt"
	"github.com/google/gopacket/pcap"
	"net"
	"sync"
	"time"
	"utils/logging"
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
func (gblInfo *LLDPGlobalInfo) InitRuntimeInfo(logger *logging.Writer,
	portConf *asicdServices.PortState) {
	gblInfo.logger = logger
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
		gblInfo.logger.Err(fmt.Sprintln("parsing lldp protocol mac failed",
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
		gblInfo.logger.Alert("Pcap already exists and create pcap called for " +
			gblInfo.Name)
		return
	}
	gblInfo.PcapHdlLock.RUnlock()
	pcapHdl, err := pcap.OpenLive(gblInfo.Name, lldpSnapshotLen,
		lldpPromiscuous, lldpTimeout)
	if err != nil {
		gblInfo.logger.Err(fmt.Sprintln("Creating Pcap Handler failed for",
			gblInfo.Name, "Error:", err))
	}
	err = pcapHdl.SetBPFFilter(LLDP_BPF_FILTER)
	if err != nil {
		gblInfo.logger.Info(fmt.Sprintln("setting filter", LLDP_BPF_FILTER,
			"for", gblInfo.Name, "failed with error:", err))
	}
	gblInfo.PcapHdlLock.Lock()
	gblInfo.PcapHandle = pcapHdl
	gblInfo.PcapHdlLock.Unlock()
}
