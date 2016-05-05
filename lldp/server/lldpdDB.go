package lldpServer

import (
	"fmt"
	_ "lldpd"
	"utils/dbutils"
)

func (svr *LLDPServer) InitDB() error {
	var err error
	svr.logger.Info("Initializing DB")
	svr.lldpDbHdl = dbutils.NewDBUtil(svr.logger)
	err = svr.lldpDbHdl.Connect()
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
		return err
	}
	svr.logger.Info("DB connection is established")
	return err
}

func (svr *LLDPServer) CloseDB() {
	svr.logger.Info("Closed lldp db")
	svr.lldpDbHdl.Disconnect()
}

func (svr *LLDPServer) ReadDB() error {
	svr.logger.Info("Reading from Database")
	svr.logger.Info("Done reading from DB")
	return nil
}
