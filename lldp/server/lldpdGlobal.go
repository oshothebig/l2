package lldpServer

import (
	"asicdServices"
	"database/sql"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	nanomsg "github.com/op/go-nanomsg"
	"sync"
	"time"
	"utils/logging"
)

type LLDPClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

type LLDPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type LLDPAsicdClient struct {
	LLDPClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type LLDPInPktChannel struct {
	pkt     gopacket.Packet
	ifIndex int32
}

type LLDPGlobalInfo struct {
	PortNum       int32
	IfIndex       int32
	Name          string
	OperState     string
	OperStateLock *sync.RWMutex
	// Pcap Handler for Each Port
	PcapHandle *pcap.Handle
	// Pcap Handler lock to write data one routine at a time
	PcapHdlLock *sync.RWMutex
}

type LLDPServer struct {
	// Basic server start fields
	logger         *logging.Writer
	lldpDbHdl      *sql.DB
	paramsDir      string
	asicdClient    LLDPAsicdClient
	asicdSubSocket *nanomsg.SubSocket

	// lldp per port global info
	lldpGblInfo        map[int32]LLDPGlobalInfo
	lldpIntfStateSlice []int32

	// lldp pcap handler default config values
	lldpSnapshotLen int32
	lldpPromiscuous bool
	lldpTimeout     time.Duration

	// lldp packet rx channel
	lldpRxPktCh chan LLDPInPktChannel
}

const (
	// Error Message
	LLDP_USR_CONF_DB                    = "/UsrConfDb.db"
	LLDP_CLIENT_CONNECTION_NOT_REQUIRED = "Connection to Client is not required"

	// Consts Init Size/Capacity
	LLDP_INITIAL_GLOBAL_INFO_CAPACITY = 100
	LLDP_RX_PKT_CHANNEL_SIZE          = 1

	// Port Operation State
	LLDP_PORT_STATE_DOWN = "DOWN"
	LLDP_PORT_STATE_UP   = "UP"

	LLDP_BPF_FILTER = "ether proto 0x88cc"
)
