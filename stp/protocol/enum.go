// enum
package stp

type PortInfoState int

const (
	PortInfoStateMine PortInfoState = iota
	PortInfoStateAged
	PortInfoStateReceived
	PortInfoStateDisabled
)

type PortDesignatedRcvInfo int

const (
	SuperiorDesignatedInfo PortDesignatedRcvInfo = iota
	RepeatedDesignatedInfo
	InferiorDesignatedInfo
	InferiorRootAlternateInfo
	OtherInfo
)

type PortRole int

const (
	PortRoleInvalid PortRole = iota
	PortRoleBridgePort
	PortRoleRootPort
	PortRoleDesignatedPort
	PortRoleAlternatePort
	PortRoleBackupPort
	PortRoleDisabledPort
)

type PointToPointMac int

const (
	StpPointToPointForceTrue  PointToPointMac = 0
	StpPointToPointForceFalse                 = 1
	StpPointToPointAuto                       = 2
)
