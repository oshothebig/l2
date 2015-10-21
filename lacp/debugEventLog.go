// debugEventLog this code is meant to serialize the logging states
package lacp

import (
	"fmt"
	"strings"
)

const (
	LacpLogEventKillSignal = iota + 1
)

type LacpDebug struct {
	LacpLogStateTransitionChan chan string
	LacpLogKillEventSignal     chan bool
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpDebug() *LacpDebug {
	lacpdebug := &LacpDebug{
		LacpLogStateTransitionChan: make(chan string),
		LacpLogKillEventSignal:     make(chan bool)}

	return lacpdebug
}

func (p *LaAggPort) LacpDebugEventLogMain() {

	p.LacpDebug = NewLacpDebug()

	go func(port *LaAggPort) {
		select {

		case <-port.LacpDebug.LacpLogKillEventSignal:
			return

		case msg := <-port.LacpDebug.LacpLogStateTransitionChan:
			fmt.Println(strings.Join([]string{string(p.portNum), p.intfNum, msg}, "-"))
		}
	}(p)

}
