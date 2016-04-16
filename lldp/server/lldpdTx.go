package lldpServer

import (
	_ "errors"
	_ "fmt"
	_ "github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	_ "time"
)

func (gblInfo *LLDPGlobalInfo) SetTTL() {
	gblInfo.ttl = Min(LLDP_MAX_TTL, (gblInfo.lldpMessageTxInterval *
		gblInfo.lldpMessageTxHoldMultiplier))
}

func (gblInfo *LLDPGlobalInfo) SetTxInterval(interval int) {
	gblInfo.lldpMessageTxInterval = interval
}

func (gblInfo *LLDPGlobalInfo) SetTxHoldMultiplier(hold int) {
	gblInfo.lldpMessageTxHoldMultiplier = hold
}

func (svr *LLDPServer) TransmitFrames(pHandle *pcap.Handle, ifIndex int32) {
	for {
		_, exists := svr.lldpGblInfo[ifIndex]
		if !exists {
			return
		}
	}
}
