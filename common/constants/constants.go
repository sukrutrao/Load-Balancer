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
	MasterBroadcastPort int16 = 3000
	SlaveBroadcastPort  int16 = 3001
	InfoSenderPort      int16 = 3002
	InfoReceiverPort    int16 = 3003
)

// Timeouts
const (
	WaitForSlaveTimeout time.Duration = 5 * time.Second
)

// Others
const (
	NumBurstAcks    int = 10
	MaxConnectRetry     = 6

	ConnectRetryBackoffBaseTime time.Duration = 2 * time.Second
)
