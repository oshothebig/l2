package main

import (
	"flag"
	"fmt"
	"l2/lldp/flexswitch"
	"l2/lldp/rpc"
	"l2/lldp/server"
	"l2/lldp/utils"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting lldp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("lldpd", "LLDP", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	debug.SetLogger(logger)
	debug.Logger.Info("Started the logger successfully.")

	debug.Logger.Info("Starting LLDP server....")
	//@FUTURE: read plugin name from json file and do init for that plugin
	//plugin.Init("flexswitch", fileName, *paramsDir)
	name := ""
	switch name {
	case "ovsdb":

	default:
		aPlugin, err := flexswitch.NewAsicPlugin(fileName)
		if err != nil {
			return
		}
		// Create lldp server handler
		lldpSvr := server.LLDPNewServer(aPlugin)
		// Until Server is connected to clients do not start with RPC
		lldpSvr.LLDPStartServer(*paramsDir)

		// Start keepalive routine
		go keepalive.InitKeepAlive("lldpd", fileName)

		// Create lldp rpc handler
		lldpHdl := lldpRpc.LLDPNewHandler(lldpSvr)
		debug.Logger.Info("Starting LLDP RPC listener....")
		err = lldpRpc.LLDPRPCStartServer(lldpHdl, *paramsDir)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln("Cannot start lldp server", err))
			return
		}
	}
}
