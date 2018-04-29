package master

import (
	// "fmt"
	"net"
	// "os"
	// "os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

var GlobalTasks map[int]MasterTask = make(map[int]MasterTask)

// Master is used to store info of master node which is currently running
type Master struct {
	myIP        net.IP
	broadcastIP net.IP
	slavePool   *SlavePool
	Logger      *logging.Logger
	lastTaskId  int

	serverHandler *Handler

	unackedSlaves   map[string]struct{}
	unackedSlaveMtx sync.RWMutex
	loadBalancer    LoadBalancerInterface

	monitor *Monitor

	close     chan struct{}
	closeWait sync.WaitGroup
}

// task as seen by master
type MasterTask struct {
	TaskId     int
	Task       *packets.TaskPacket
	Load       int
	AssignedTo *Slave
	IsAssigned bool
	TaskStatus packets.Status
}

// master constructor
func (m *Master) initDS() {
	m.close = make(chan struct{})
	m.unackedSlaves = make(map[string]struct{})
	m.slavePool = &SlavePool{
		Logger: m.Logger,
	}
	m.monitor = &Monitor{
		id:          0,
		ip:          []byte{},
		reqSendPort: 0,
		acked:       false,
		logger:      m.Logger,
	}
	m.lastTaskId = 0
	m.loadBalancer = &RoundRobin{&LoadBalancerBase{slavePool: m.slavePool}, -1}
}

// run master
func (m *Master) Run() {

	// { // Handling ctrl+C for graceful shutdown.
	// 	c := make(chan os.Signal, 1)
	// 	signal.Notify(c, os.Interrupt)
	// 	go func() {
	// 		<-c
	// 		m.Logger.Info(logger.FormatLogMessage("msg", "Closing Master gracefully..."))
	// 		close(m.close)
	// 	}()
	// }

	m.initDS()
	m.updateAddress()
	m.StartServer(&HTTPOptions{
		Logger: m.Logger,
	})
	m.closeWait.Add(2)
	go m.connect()
	go m.gc_routine()
	m.Logger.Info(logger.FormatLogMessage("msg", "Master running"))
	// time.Sleep(5 * time.Second)
	// m.Logger.Info(logger.FormatLogMessage("msg", "Starting Tasks"))
	// for i := 0; i < 10; i++ {
	// 	t := packets.TaskPacket{TaskTypeID: packets.FibonacciTaskType, N: i + 1, Close: make(chan struct{})}
	// 	m.assignNewTask(&t, i+1)
	// 	<-t.Close
	// 	fmt.Println(t.Result)
	// 	// time.Sleep(2 * time.Second)
	// }
	m.Logger.Info(logger.FormatLogMessage("msg", "Tasks complete"))
	<-m.close
	m.Close()
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

func (m *Master) SlaveExists(ip net.IP, id uint16) bool {
	return m.slavePool.SlaveExists(ip, id)
}

type monitorTcpData struct {
	n   int
	buf [1024]byte
}

func (m *Master) StartMonitor() error {

	packetChan := make(chan monitorTcpData)

	if err := m.monitor.StartAcceptingRequests(packetChan); err != nil {
		return err
	}

	m.closeWait.Add(1)
	go func() {

		end := false
		for !end {
			select {
			case <-m.close:
				m.Logger.Info(logger.FormatLogMessage("msg", "Stopping Monitor Request Listener"))
				end = true
				break
			default:
				if m.monitor.acked {
					m.handleMonitorRequests(packetChan)
				} else {
					end = true
				}
			}
		}

		m.closeWait.Done()
	}()

	return nil
}

func (m *Master) handleMonitorRequests(packetChan <-chan monitorTcpData) {

	select {
	case packet, ok := <-packetChan:
		if !ok {
			break
		}

		packetType, err := packets.GetPacketType(packet.buf[:packet.n])
		if err != nil {
			m.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			return
		}
		switch packetType {
		case packets.MonitorRequest:
			m.monitor.SendSlaveIPs(m.slavePool.GetAllSlaveIPs())
		default:
			m.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
		}

	// Timeout
	case <-time.After(constants.WaitForReqTimeout):

	}

}

func (m *Master) gc_routine() {
	m.Logger.Info(logger.FormatLogMessage("msg", "Garbage collection routine started"))
	end := false
	for !end {
		select {
		case <-m.close:
			end = true
		default:
			m.slavePool.gc(m.Logger)
		}
		<-time.After(constants.GarbageCollectionInterval)
	}
	m.closeWait.Done()
}

func (m *Master) Close() {
	m.Logger.Info(logger.FormatLogMessage("msg", "Closing Master gracefully..."))

	// First stopping to accept any more tasks.
	if err := m.serverHandler.Shutdown(); err != nil {
		m.Logger.Error(logger.FormatLogMessage("msg", "Failed to ShutDown the server", "err", err.Error()))
	}

	// Closing all work of master.
	select {
	case <-m.close:
	default:
		close(m.close)
	}
	m.monitor.Close()

	// Closing all slaves.
	m.slavePool.Close(m.Logger)

	m.closeWait.Wait()
}

// create task, find whom to assign, and send to that slave's channel
func (m *Master) assignNewTask(task *packets.TaskPacket, load int) error {
	t := m.createTask(task, load)
	s := m.assignTask(t)
	m.Logger.Info(logger.FormatLogMessage("msg", "Assigned Task", "Task", task.Description(), "Slave", strconv.Itoa(int(s.id)))) // TODO - cast may not be correct
	p := m.assignTaskPacket(t)
	pt := packets.CreatePacketTransmit(p, packets.TaskRequest) // TODO - fix this
	s.tasksUndertaken = append(s.tasksUndertaken, t.TaskId)
	s.sendChan <- pt
	//	var packetType packets.TaskRequestPacket
	//	s.sendChan <- packetType // TODO - this could cause issues, packaging with pt (above) would be better
	return nil
}

// TODO: regularly send info request to all slaves.
