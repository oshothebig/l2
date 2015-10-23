// test_lacp
package lacp_test

import (
	"l2/lacp"
	"testing"
	"time"
)

func TestLaAggPortCreate(t *testing.T) {

	p := lacp.NewLaAggPort(1, "eth1.1")

	p.Start(false)

	time.Sleep(time.Second * 30)

}
