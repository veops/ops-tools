package internal

type NodeType int

const (
	NTServer NodeType = iota
	NTSwitch
	NTRouter
	NTInternet
	NTOffline
	NTOther
)

func (nt NodeType) String() string {
	switch nt {
	case NTServer:
		return "server"
	case NTSwitch:
		return "switch"
	case NTRouter:
		return "router"
	case NTInternet:
		return "internet"
	case NTOffline:
		return "offline"
	default:
		return "other"
	}
}
