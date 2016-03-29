package lldpServer

import (
	_ "asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/google/gopacket"
	"io/ioutil"
	_ "net"
	"os"
	"os/signal"
	"strconv"
	_ "strings"
	_ "sync"
	"syscall"
	"time"
	"utils/ipcutils"
	"utils/logging"
)

func LLDPNewServer(log *logging.Writer) *LLDPServer {
	lldpServerInfo := &LLDPServer{}
	lldpServerInfo.logger = log
	// Allocate memory to all the Data Structures
	//lldpServerInfo.LLDPInitGlobalDS()
	return lldpServerInfo
}

func (svr *LLDPServer) LLDPStartServer(paramsDir string) {
	svr.paramsDir = paramsDir
	// First connect to client to avoid any issues with start/re-start
	svr.LLDPConnectAndInitPortVlan()

	// Initialize DB
	err := svr.LLDPInitDB()
	if err != nil {
		svr.logger.Err("DB init failed")
	} else {
		// Populate Gbl Configs
		svr.LLDPReadDB()
		svr.LLDPCloseDB()
	}
	go svr.LLDPChannelHanlder()
}

func (svr *LLDPServer) LLDPConnectAndInitPortVlan() error {
	configFile := svr.paramsDir + "/clients.json"
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Error while reading configuration file",
			configFile))
		return err
	}
	var unConnectedClients []LLDPClientJson
	err = json.Unmarshal(bytes, &unConnectedClients)
	if err != nil {
		svr.logger.Err("Error in Unmarshalling Json")
		return err
	}

	// connect to client
	for {
		time.Sleep(time.Millisecond * 500)
		for i := 0; i < len(unConnectedClients); i++ {
			err := svr.LLDPConnectToUnConnectedClient(unConnectedClients[i])
			if err == nil {
				svr.logger.Info("Connected to " +
					unConnectedClients[i].Name)
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)

			} else if err.Error() == LLDP_CLIENT_CONNECTION_NOT_REQUIRED {
				svr.logger.Info("connection to " +
					unConnectedClients[i].Name +
					" not required")
				unConnectedClients = append(unConnectedClients[:i],
					unConnectedClients[i+1:]...)
			}
		}
		if len(unConnectedClients) == 0 {
			svr.logger.Info("all clients connected successfully")
			break
		}
	}

	svr.LLDPGetInfoFromAsicd()

	// OS Signal channel listener thread
	svr.LLDPOSSignalHandle()
	return err
}

func (svr *LLDPServer) LLDPConnectToUnConnectedClient(client LLDPClientJson) error {
	switch client.Name {
	case "asicd":
		return svr.LLDPConnectToAsicd(client)
	default:
		return errors.New(LLDP_CLIENT_CONNECTION_NOT_REQUIRED)
	}
}

func (svr *LLDPServer) LLDPConnectToAsicd(client LLDPClientJson) error {
	svr.logger.Info(fmt.Sprintln("Connecting to asicd at port",
		client.Port))
	var err error
	svr.asicdClient.Address = "localhost:" + strconv.Itoa(client.Port)
	svr.asicdClient.Transport, svr.asicdClient.PtrProtocolFactory, err =
		ipcutils.CreateIPCHandles(svr.asicdClient.Address)
	if svr.asicdClient.Transport == nil ||
		svr.asicdClient.PtrProtocolFactory == nil ||
		err != nil {
		svr.logger.Err(fmt.Sprintln("Connecting to",
			client.Name, "failed ", err))
		return err
	}
	svr.asicdClient.ClientHdl =
		asicdServices.NewASICDServicesClientFactory(
			svr.asicdClient.Transport,
			svr.asicdClient.PtrProtocolFactory)
	svr.asicdClient.IsConnected = true
	return nil
}

func (svr *LLDPServer) LLDPOSSignalHandle() {
	sigChannel := make(chan os.Signal, 1)
	signalList := []os.Signal{syscall.SIGHUP}
	signal.Notify(sigChannel, signalList...)
	go svr.LLDPSignalHandler(sigChannel)
}

func (svr *LLDPServer) LLDPSignalHandler(sigChannel <-chan os.Signal) {
	signal := <-sigChannel
	switch signal {
	case syscall.SIGHUP:
		svr.logger.Alert("Received SIGHUP Signal")
		//svr.LLDPCloseAllPcapHandlers()
		//svr.LLDPDeAllocateMemoryToGlobalDS()
		svr.logger.Alert("Closed all pcap's and freed memory")
		os.Exit(0)
	default:
		svr.logger.Info(fmt.Sprintln("Unhandled Signal:", signal))
	}
}

func (svr *LLDPServer) LLDPChannelHanlder() {
	// Start receviing in rpc values in the channell
}
