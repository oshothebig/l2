// aggregator
package lacp

// Indicates on a port what state
// the aggSelected is in
const (
	LacpAggSelected = iota
	LacpAggStandby
	LacpAggUnSelected
	LacpAggOutOfSync
)

// 6.4.6 Variables associated with each Aggregator
type LacpAggregator struct {
	mac_address [6]uint8
	// unique id of the aggregator within the system
	identifier int
	// aggregation capability
	// TRUE - port attached to this aggregetor is not capable
	//        of aggregation to any other aggregator
	// FALSE - port attached to this aggregator is able of
	//         aggregation to any other aggregator
	individual      bool
	actor_admin_key int
	actor_oper_key  int
	// MAC address component of the system identifier of the
	// remote system
	partner_system          [6]uint8
	partner_system_priority int
	partner_oper_key        int
	receive_state           bool
	transmit_state          bool
	// Port number from LaAggPort
	portNumList []int
}

func LacpAttachMuxToAggregator() {
	// TODO
}

func LacpDetachMuxFromAggregator() {
	// TODO
}
