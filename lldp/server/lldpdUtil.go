package lldpServer

import (
	"fmt"
	"github.com/google/gopacket/pcap"
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

/*  Stop RX cache timer
 */
func (gblInfo *LLDPGlobalInfo) StopCacheTimer() {
	if gblInfo.clearCacheTimer == nil {
		return
	}
	gblInfo.clearCacheTimer.Stop()
}

/*  Return back all the memory which was allocated using new
 */
func (gblInfo *LLDPGlobalInfo) FreeDynamicMemory() {
	gblInfo.lldpFrame = nil
	gblInfo.lldpLinkInfo = nil
	gblInfo.OperStateLock = nil
	gblInfo.PcapHdlLock = nil
}

/*  Create Pcap Handler
 */
func (gblInfo *LLDPGlobalInfo) CreatePcapHandler(lldpSnapshotLen int32,
	lldpPromiscuous bool, lldpTimeout time.Duration) {
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
