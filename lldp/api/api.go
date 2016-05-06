package api

import (
	"l2/lldp/config"
	"sync"
)

type ApiLayer struct {
	gblCfgCh  chan *config.Global
	ifstateCh chan *config.PortState
}

var lldpapi *ApiLayer = nil
var once sync.Once

/*  Singleton instance should be accesible only within api
 */
func getInstance() *ApiLayer {
	once.Do(func() {
		lldpapi = &ApiLayer{}
	})
	return lldpapi
}

func Init(gCh chan *config.Global, ifCh chan *config.PortState) {
	lldpapi = getInstance()
	lldpapi.gblCfgCh = gCh
	lldpapi.ifstateCh = ifCh
}

func SendGlobalConfig(ifIndex int32, enable bool) {
	lldpapi.gblCfgCh <- &config.Global{ifIndex, enable}
}

func SendPortStateChange(ifIndex int32, state string) {
	lldpapi.ifstateCh <- &config.PortState{ifIndex, state}
}
