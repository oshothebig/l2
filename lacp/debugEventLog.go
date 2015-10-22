// debugEventLog this code is meant to serialize the logging states
package lacp

import (
	"fmt"
	"strings"
	"time"
)

type LacpDebug struct {
	LacpLogChan chan string
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpDebug() *LacpDebug {
	lacpdebug := &LacpDebug{
		LacpLogChan: make(chan string)}

	return lacpdebug
}

func (l *LacpDebug) Stop() {
	close(l.LacpLogChan)
}

func (p *LaAggPort) LacpDebugEventLogMain() {

	p.LacpDebug = NewLacpDebug()

	go func(port *LaAggPort) {
		select {

		case msg, logEvent := <-port.LacpDebug.LacpLogChan:
			if logEvent {
				fmt.Println(strings.Join([]string{time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006"), string(p.portNum), p.intfNum, msg}, "-"))
			} else {
				return
			}
		}
	}(p)

}
