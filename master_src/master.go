package master

import (
	"fmt"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"net"
	"os"
	"sync"
)

type Master struct {
	myIP        net.IP
	broadcastIP net.IP

	// Should acquire this lock before accessing slave.
	// Acquire Write Lock only when removing or adding a slave.
	// Read Lock in other cases (modifying or accessing individual slaves)
	slavesMtx sync.RWMutex

	slaves []Slave

	close chan struct{}
}

type Slave struct {
	ip string

	// Need to acquire Write Lock which modifying any value.
	mtx sync.RWMutex
}

func (m *Master) Run() {
	fmt.Println("Master running")
	m.updateAddress()
	m.connect()
}

func (m *Master) updateAddress() {

	ipnet, err := utility.GetMyIP()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}

	m.myIP = ipnet.IP

	for i, b := range ipnet.Mask {
		m.broadcastIP = append(m.broadcastIP, (m.myIP[i] | (^b)))
	}

}
