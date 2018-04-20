package constants

import (
	"net"
	"time"
)

// Addressess
var (
	BroadcastReceiveAddress net.IP = []byte{0, 0, 0, 0}
)

// Ports
const (
	MasterBroadcastPort uint16 = 3000
)

// Timeouts
const (
	WaitForSlaveTimeout   time.Duration = 5 * time.Second
	WaitForInfoReqTimeout time.Duration = 5 * time.Second
)

// Others
const (
	NumBurstAcks    int = 10
	MaxConnectRetry     = 6

	ConnectRetryBackoffBaseTime time.Duration = 2 * time.Second
)
