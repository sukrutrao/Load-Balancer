package master

import (
	"net"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node which is currently running
type Master struct {
	myIP        net.IP
	broadcastIP net.IP
	slavePool   SlavePool
	close       chan struct{}
	Logger      *logging.Logger
	tasks       map[int]Task
}

type Task struct {
	TaskId     int
	Task       string
	Load       int
	AssignedTo *Slave
	IsAssigned bool
	TaskStatus constants.Status
}

func (m *Master) Run() {
	m.Logger.Info(logger.FormatLogMessage("msg", "Master running"))
	m.updateAddress()
	m.connect()
}

func (m *Master) updateAddress() {
	ipnet, err := utility.GetMyIP()
	if err != nil {
		m.Logger.Fatal(logger.FormatLogMessage("msg", "Failed to get IP", "err", err.Error()))
	}

	m.myIP = ipnet.IP
	for i, b := range ipnet.Mask {
		m.broadcastIP = append(m.broadcastIP, (m.myIP[i] | (^b)))
	}
}

func (m *Master) Close() {
	close(m.close)
}

// Slave is used to store info of slave node connected to it
type Slave struct {
	ip string

	// Need to acquire Write Lock which modifying any value.
	mtx sync.RWMutex
}

type SlavePool struct {
	mtx    sync.RWMutex
	slaves []*Slave
}
