package lldpServer

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	_ "lldpd"
)

func (svr *LLDPServer) InitDB() error {
	var err error
	svr.logger.Info("Initializing DB")
	svr.lldpDbHdl, err = redis.Dial("tcp", ":6379")
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
		return err
	}
	svr.logger.Info("DB connection is established")
	return err
}

func (svr *LLDPServer) CloseDB() {
	svr.logger.Info("Closed lldp db")
	svr.lldpDbHdl.Close()
}

func (svr *LLDPServer) ReadDB() error {
	svr.logger.Info("Reading from Database")
	svr.logger.Info("Done reading from DB")
	return nil
}
