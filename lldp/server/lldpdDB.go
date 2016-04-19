package lldpServer

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	_ "lldpd"
)

func (svr *LLDPServer) InitDB() error {
	svr.logger.Info("Initializing SQL DB")
	var err error
	dbName := svr.paramsDir + LLDP_USR_CONF_DB
	svr.logger.Info("location for DB is " + dbName)
	svr.lldpDbHdl, err = sql.Open("sqlite3", dbName)
	if err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
		return err
	}

	if err = svr.lldpDbHdl.Ping(); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to keep db connection alive", err))
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
