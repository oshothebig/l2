package config

type Global struct {
	IfIndex int32
	Enable  bool
}

type PortInfo struct {
	IfIndex   int32
	PortNum   int32
	Name      string
	OperState string
	MacAddr   string
}

type PortState struct {
	IfIndex int32
	IfState string
}
