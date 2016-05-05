package lldpServer

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"l2/lldp/utils"
	_ "lldpd"
)

func (svr *LLDPServer) InitDB() error {
	var err error
	debug.Logger.Info("Initializing DB")
	svr.lldpDbHdl, err = redis.Dial("tcp", ":6379")
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
		return err
	}
	debug.Logger.Info("DB connection is established")
	return err
}

func (svr *LLDPServer) CloseDB() {
	debug.Logger.Info("Closed lldp db")
	svr.lldpDbHdl.Close()
}

func (svr *LLDPServer) ReadDB() error {
	debug.Logger.Info("Reading from Database")
	debug.Logger.Info("Done reading from DB")
	return nil
}
