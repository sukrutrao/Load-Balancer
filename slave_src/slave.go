package slave

import (
	"net"
	// "os"
	// "os/signal"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node
type Master struct {
	ip net.IP
}

// Slave is used to store info of slave node which is currently running
type Slave struct {
	myIP        net.IP
	broadcastIP net.IP
	loadReqPort uint16

	reqRecvPort uint16
	reqSendPort uint16

	master      Master
	currentLoad uint64
	maxLoad     uint64
	sendChan    chan packets.PacketTransmit

	Logger *logging.Logger

	serverHandler *Handler
	metric        Metric

	close     chan struct{}
	closeWait sync.WaitGroup
	tasks     map[int]SlaveTask
}

type Metric struct {
	TasksCompleted uint32
	TasksAccepted  uint32
	TasksRequested uint32
}

type SlaveTask struct {
	TaskId     int
	Task       packets.TaskPacket
	Load       uint64
	TaskStatus packets.Status
	//	Result     packets.TaskPacket *packets.TaskResult
}

func (s *Slave) initDS() {
	s.close = make(chan struct{})
	s.maxLoad = 10000000
	s.tasks = make(map[int]SlaveTask)
	s.sendChan = make(chan packets.PacketTransmit)
}

type TaskResult struct {
	Result string
}

// Run starts the slave
func (s *Slave) Run() {

	s.initDS()
	s.updateAddress()
	s.StartServer(&HTTPOptions{
		Logger: s.Logger,
	})
	s.closeWait.Add(1)
	if err := s.connect(); err != nil {
		s.Logger.Error(logger.FormatLogMessage("msg", "Failed to connect to master", "err", err.Error()))
		s.Close()
	}
	s.Logger.Info(logger.FormatLogMessage("msg", "Slave running"))
	s.closeWait.Wait()
}

func (s *Slave) updateAddress() {
	ipnet, err := utility.GetMyIP()
	if err != nil {
		s.Logger.Fatal(logger.FormatLogMessage("msg", "Failed to get IP", "err", err.Error()))
	}

	s.myIP = ipnet.IP
	for i, b := range ipnet.Mask {
		s.broadcastIP = append(s.broadcastIP, (s.myIP[i] | (^b)))
	}
}

func (s *Slave) Close() {
	s.Logger.Info(logger.FormatLogMessage("msg", "Closing Slave gracefully..."))

	if err := s.serverHandler.Shutdown(); err != nil {
		s.Logger.Error(logger.FormatLogMessage("msg", "Failed to ShutDown the server", "err", err.Error()))
	}

	close(s.close)
	s.closeWait.Wait()
}
