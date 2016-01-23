// bridge.go
package stp

type Bridge struct {

	// 17.18.1
	BEGIN bool
	// 17.18.2
	BridgeIdentifier [8]uint8
	// 17.18.3
	// Root/Designated equal to Bridge Identifier
	BridgePriority PriorityVector
	// 17.18.4
	BridgeTimes Times
	// 17.18.6
	RootPortId int32
	// 17.18.7
	RootTimes Times
}

type PriorityVector struct {
	RootBridgeId       [8]uint8
	RootPathCost       int32
	DesignatedBridgeId [8]uint8
	DesignatedPortId   int32
	BridgePortId       int32
}

type Times struct {
	ForwardingDelay int32
	HelloTime       int32
	MaxAge          int32
	MessageAge      int32
}
