package lldpRpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"l2/lldp/server"
	"l2/lldp/utils"
	"lldpd"
	"strconv"
)

const (
	LLDP_RPC_NO_PORT = "could not find port and hence not starting rpc"
)

type LLDPHandler struct {
	server *server.LLDPServer
}
type LLDPClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

func LLDPNewHandler(lldpSvr *server.LLDPServer) *LLDPHandler {
	hdl := new(LLDPHandler)
	hdl.server = lldpSvr
	return hdl
}

func LLDPRPCStartServer(handler *LLDPHandler, paramsDir string) error {
	fileName := paramsDir

	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	clientJson, err := LLDPRPCGetClient(fileName, "lldpd")
	if err != nil || clientJson == nil {
		return err
	}
	debug.Logger.Info(fmt.Sprintln("Got Client Info for", clientJson.Name, " port",
		clientJson.Port))
	// create processor, transport and protocol for server
	processor := lldpd.NewLLDPDServicesProcessor(handler)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket("localhost:" + strconv.Itoa(clientJson.Port))
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

func LLDPRPCGetClient(fileName string, process string) (*LLDPClientJson, error) {
	var allClients []LLDPClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		debug.Logger.Err(fmt.Sprintf("Failed to open LLDPd config file:%s, err:%s",
			fileName, err))
		return nil, err
	}

	json.Unmarshal(data, &allClients)
	for _, client := range allClients {
		if client.Name == process {
			return &client, nil
		}
	}

	debug.Logger.Err(fmt.Sprintf("Did not find port for %s in config file:%s",
		process, fileName))
	return nil, errors.New(LLDP_RPC_NO_PORT)

}

func (h *LLDPHandler) CreateLLDPIntf(config *lldpd.LLDPIntf) (r bool, err error) {
	return true, nil
}

func (h *LLDPHandler) DeleteLLDPIntf(config *lldpd.LLDPIntf) (r bool, err error) {
	return true, nil
}

func (h *LLDPHandler) UpdateLLDPIntf(origconfig *lldpd.LLDPIntf,
	newconfig *lldpd.LLDPIntf, attrset []bool) (r bool, err error) {
	return true, nil
}

func (h *LLDPHandler) convertLLDPIntfStateEntryToThriftEntry(
	state lldpd.LLDPIntfState) *lldpd.LLDPIntfState {
	entry := lldpd.NewLLDPIntfState()
	entry.LocalPort = state.LocalPort
	entry.PeerMac = state.PeerMac
	entry.Port = state.Port
	entry.HoldTime = state.HoldTime
	entry.Enable = state.Enable
	entry.IfIndex = state.IfIndex
	return entry
}

func (h *LLDPHandler) GetBulkLLDPIntfState(fromIndex lldpd.Int,
	count lldpd.Int) (*lldpd.LLDPIntfStateGetInfo, error) {
	nextIdx, currCount, lldpIntfStateEntries := h.server.GetBulkLLDPIntfState(
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

func (h *LLDPHandler) GetLLDPIntfState(ifIndex int32) (*lldpd.LLDPIntfState, error) {
	return nil, nil
}
