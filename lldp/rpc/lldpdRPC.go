package lldpRpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"l2/lldp/server"
	"lldpd"
	"strconv"
	"utils/logging"
)

const (
	LLDP_RPC_NO_PORT = "could not find port and hence not starting rpc"
)

type LLDPHandler struct {
	server *lldpServer.LLDPServer
	logger *logging.Writer
}
type LLDPClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

func LLDPNewHandler(lldpSvr *lldpServer.LLDPServer, logger *logging.Writer) *LLDPHandler {
	hdl := new(LLDPHandler)
	hdl.server = lldpSvr
	hdl.logger = logger
	return hdl
}

func LLDPRPCStartServer(log *logging.Writer, handler *LLDPHandler, paramsDir string) error {
	logger := log
	fileName := paramsDir

	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	clientJson, err := LLDPRPCGetClient(logger, fileName, "lldpd")
	if err != nil || clientJson == nil {
		return err
	}
	logger.Info(fmt.Sprintln("Got Client Info for", clientJson.Name, " port",
		clientJson.Port))
	// create processor, transport and protocol for server
	processor := lldpd.NewLLDPDServicesProcessor(handler)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTServerSocket("localhost:" + strconv.Itoa(clientJson.Port))
	if err != nil {
		logger.Info(fmt.Sprintln("StartServer: NewTServerSocket "+
			"failed with error:", err))
		return err
	}
	server := thrift.NewTSimpleServer4(processor, transport,
		transportFactory, protocolFactory)
	err = server.Serve()
	if err != nil {
		logger.Err(fmt.Sprintln("Failed to start the listener, err:", err))
		return err
	}
	return nil
}

func LLDPRPCGetClient(logger *logging.Writer, fileName string,
	process string) (*LLDPClientJson, error) {
	var allClients []LLDPClientJson

	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		logger.Err(fmt.Sprintf("Failed to open LLDPd config file:%s, err:%s",
			fileName, err))
		return nil, err
	}

	json.Unmarshal(data, &allClients)
	for _, client := range allClients {
		if client.Name == process {
			return &client, nil
		}
	}

	logger.Err(fmt.Sprintf("Did not find port for %s in config file:%s",
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
