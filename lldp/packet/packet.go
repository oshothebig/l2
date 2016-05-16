package packet

import (
	"github.com/google/gopacket/layers"
	"net"
	"time"
)

const (
	LLDP_PROTO_DST_MAC = "01:80:c2:00:00:0e"
	LLDP_MAX_TTL       = 65535
)

type RX struct {
	// ethernet frame Info (used for rx/tx)
	SrcMAC net.HardwareAddr // NOTE: Please be informed this is Peer Mac Addr
	DstMAC net.HardwareAddr

	// lldp rx information
	RxFrame         *layers.LinkLayerDiscovery
	RxLinkInfo      *layers.LinkLayerDiscoveryInfo
	ClearCacheTimer *time.Timer
}

type TX struct {
	// tx information
	ttl                     int
	DstMAC                  net.HardwareAddr
	MessageTxInterval       int
	MessageTxHoldMultiplier int
	useCacheFrame           bool
	cacheFrame              []byte
	TxTimer                 *time.Timer
}
