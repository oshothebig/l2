package flexswitch

import (
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"l2/lldp/api"
	"l2/lldp/config"
	"l2/lldp/utils"
	"lldpd"
	"strconv"
)

type ConfigHandler struct {
}

type NBPlugin struct {
	handler  *ConfigHandler
	fileName string
}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

func NewNBPlugin(handler *ConfigHandler, fileName string) *NBPlugin {
	l := &NBPlugin{handler, fileName}
	return l
}

func (p *NBPlugin) Start() error {
	fileName := p.fileName + CLIENTS_FILE_NAME

	clientJson, err := getClient(fileName, "lldpd")
	if err != nil || clientJson == nil {
		return err
	}
	debug.Logger.Info(fmt.Sprintln("Got Client Info for", clientJson.Name, " port",
		clientJson.Port))
	// create processor, transport and protocol for server
	processor := lldpd.NewLLDPDServicesProcessor(p.handler)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket("localhost:" +
		strconv.Itoa(clientJson.Port))
	if err != nil {
		debug.Logger.Info(fmt.Sprintln("StartServer: NewTServerSocket "+
			"failed with error:", err))
		return err
	}
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to start the listener, err:", err))
		return err
	}
	return nil
}

func (h *ConfigHandler) CreateLLDPIntf(config *lldpd.LLDPIntf) (r bool, err error) {
	return true, nil
}

func (h *ConfigHandler) DeleteLLDPIntf(config *lldpd.LLDPIntf) (r bool, err error) {
	return true, nil
}

func (h *ConfigHandler) UpdateLLDPIntf(origconfig *lldpd.LLDPIntf,
	newconfig *lldpd.LLDPIntf, attrset []bool, op string) (r bool, err error) {
	return true, nil
}

func (h *ConfigHandler) convertLLDPIntfStateEntryToThriftEntry(
	state config.IntfState) *lldpd.LLDPIntfState {
	entry := lldpd.NewLLDPIntfState()
	entry.LocalPort = state.LocalPort
	entry.PeerMac = state.PeerMac
	entry.Port = state.Port
	entry.HoldTime = state.HoldTime
	entry.Enable = state.Enable
	entry.IfIndex = state.IfIndex
	return entry
}

func (h *ConfigHandler) GetBulkLLDPIntfState(fromIndex lldpd.Int,
	count lldpd.Int) (*lldpd.LLDPIntfStateGetInfo, error) {

	nextIdx, currCount, lldpIntfStateEntries := api.GetIntfStates(
		int(fromIndex), int(count))
	if lldpIntfStateEntries == nil {
		return nil, errors.New("No neighbor found")
	}

	lldpEntryResp := make([]*lldpd.LLDPIntfState, len(lldpIntfStateEntries))

	for idx, item := range lldpIntfStateEntries {
		lldpEntryResp[idx] = h.convertLLDPIntfStateEntryToThriftEntry(item)
	}

	lldpEntryBulk := lldpd.NewLLDPIntfStateGetInfo()
	lldpEntryBulk.StartIdx = fromIndex
	lldpEntryBulk.EndIdx = lldpd.Int(nextIdx)
	lldpEntryBulk.Count = lldpd.Int(currCount)
	lldpEntryBulk.More = (nextIdx != 0)
	lldpEntryBulk.LLDPIntfStateList = lldpEntryResp

	return lldpEntryBulk, nil
}

func (h *ConfigHandler) GetLLDPIntfState(ifIndex int32) (*lldpd.LLDPIntfState, error) {
	return nil, nil
}
