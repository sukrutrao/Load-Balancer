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
	HTTPServerPort      uint16 = 4242
)

// Timeouts
const (
	WaitForSlaveTimeout       time.Duration = 5 * time.Second
	WaitForReqTimeout                       = 5 * time.Second
	LoadRequestInterval                     = 5 * time.Second
	GarbageCollectionInterval               = 5 * time.Second
	ReceiveTimeout                          = 5 * time.Second

	// SlaveReceiveTimeout should be bigger than LoadRequestInterval.
	SlaveReceiveTimeout          = 10 * time.Second
	SlaveConnectionAcceptTimeout = 15 * time.Second
)

// Others
const (
	NumBurstAcks    int = 10
	MaxConnectRetry     = 6

	ConnectRetryBackoffBaseTime time.Duration = 2 * time.Second
)
