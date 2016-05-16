package server

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"l2/lldp/config"
	"l2/lldp/packet"
	"l2/lldp/plugin"
	"time"
	"utils/dbutils"
)

type InPktChannel struct {
	pkt     gopacket.Packet
	ifIndex int32
}

type SendPktChannel struct {
	ifIndex int32
}

type LLDPGlobalInfo struct {
	// Port information
	Port config.PortInfo
	// Pcap Handler for Each Port
	PcapHandle *pcap.Handle
	// rx information
	RxInfo *packet.RX
	// tx information
	TxInfo *packet.TX
}

type LLDPServer struct {
	// Basic server start fields
	lldpDbHdl *dbutils.DBUtil
	paramsDir string

	asicPlugin plugin.AsicIntf
	CfgPlugin  plugin.ConfigIntf

	// lldp per port global info
	lldpGblInfo          map[int32]LLDPGlobalInfo
	lldpIntfStateSlice   []int32
	lldpUpIntfStateSlice []int32

	// lldp pcap handler default config values
	lldpSnapshotLen int32
	lldpPromiscuous bool
	lldpTimeout     time.Duration

	// lldp packet rx channel
	lldpRxPktCh chan InPktChannel
	// lldp send packet channel
	lldpTxPktCh chan SendPktChannel
	// lldp global config channel
	GblCfgCh chan *config.Global
	// lldp asic notification channel
	IfStateCh chan *config.PortState

	// lldp exit
	lldpExit chan bool
}

const (
	// LLDP profiling
	LLDP_CPU_PROFILE_FILE = "/var/log/lldp.prof"

	// Consts Init Size/Capacity
	LLDP_INITIAL_GLOBAL_INFO_CAPACITY = 100
	LLDP_RX_PKT_CHANNEL_SIZE          = 10
	LLDP_TX_PKT_CHANNEL_SIZE          = 10

	// Port Operation State
	LLDP_PORT_STATE_DOWN = "DOWN"
	LLDP_PORT_STATE_UP   = "UP"

	LLDP_BPF_FILTER                 = "ether proto 0x88cc"
	LLDP_DEFAULT_TX_INTERVAL        = 30
	LLDP_DEFAULT_TX_HOLD_MULTIPLIER = 4
	LLDP_MIN_FRAME_LENGTH           = 12 // this is 12 bytes
)
