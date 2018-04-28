package master

import (
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node which is currently running
type Master struct {
	myIP        net.IP
	broadcastIP net.IP
	slavePool   *SlavePool
	Logger      *logging.Logger

	unackedSlaves   map[string]struct{}
	unackedSlaveMtx sync.RWMutex

	monitor *Monitor

	close     chan struct{}
	closeWait sync.WaitGroup
}

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
}

func (m *Master) Run() {
	m.initDS()
	m.updateAddress()
	go m.connect()
	go m.gc_routine()
	m.Logger.Info(logger.FormatLogMessage("msg", "Master running"))

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
}

func (m *Master) Close() {
	m.Logger.Info(logger.FormatLogMessage("msg", "Closing Master gracefully..."))
	select {
	case <-m.close:
	default:
		close(m.close)
	}
	m.monitor.Close()
	m.slavePool.Close(m.Logger)
	m.closeWait.Wait()
}

//////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////
////////////////////////// MONITOR ///////////////////////////////
//////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////

type Monitor struct {
	id          uint16
	ip          net.IP
	reqSendPort uint16
	acked       bool
	conn        net.Conn
	logger      *logging.Logger

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (mo *Monitor) StartAcceptingRequests(packetChan chan<- monitorTcpData) error {
	if !mo.acked {
		return errors.New("Unacked monitor")
	}

	address := mo.ip.String() + ":" + strconv.Itoa(int(mo.reqSendPort))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	mo.conn = conn

	mo.closeWait.Add(1)
	go func() {
		end := false
		for !end {
			select {
			case <-mo.close:
				end = true
			default:
				var buf [2048]byte
				// TODO: add timeout
				n, err := mo.conn.Read(buf[0:])
				mo.logger.Info(logger.FormatLogMessage("msg", "Monitor request"))
				if err != nil {
					mo.logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP", "err", err.Error()))
					if err == io.EOF {
						select {
						case <-mo.close:
						default:
							close(mo.close)
						}
						mo.acked = false
						end = true
					}
					continue
				}

				var bufCopy [1024]byte
				copy(bufCopy[:], buf[:])
				packetChan <- monitorTcpData{
					n:   n,
					buf: bufCopy,
				}
			}
		}

		mo.closeWait.Done()
	}()

	return nil
}

func (mo *Monitor) SendSlaveIPs(slaveIPs []string) {

	res := packets.MonitorResponsePacket{
		SlaveIPs: slaveIPs,
	}

	bytes, err := packets.EncodePacket(res, packets.MonitorResponse)
	if err != nil {
		mo.logger.Error(logger.FormatLogMessage("msg", "Failed to encode packet",
			"packet", packets.MonitorResponse.String(), "err", err.Error()))
		return
	}

	_, err = mo.conn.Write(bytes)
	mo.logger.Info(logger.FormatLogMessage("msg", "Sending slave ip"))
	if err != nil {
		mo.logger.Error(logger.FormatLogMessage("msg", "Failed to send packet",
			"packet", packets.MonitorResponse.String(), "err", err.Error()))
		return
	}

}

func (mo *Monitor) Close() {
	close(mo.close)
	mo.closeWait.Wait()
}

//////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////
////////////////////////// SLAVE /////////////////////////////////
//////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////

// Slave is used to store info of slave node connected to it
type Slave struct {
	ip          string
	id          uint16
	loadReqPort uint16
	reqSendPort uint16
	Logger      *logging.Logger

	load              float64
	lastLoadTimestamp time.Time
	mtx               sync.RWMutex

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (s *Slave) UpdateLoad(l float64, ts time.Time) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if ts.After(s.lastLoadTimestamp) {
		s.load = l
		s.lastLoadTimestamp = ts
	}
}

func (s *Slave) InitDS() {
	s.close = make(chan struct{})
}

func (s *Slave) InitConnections() error {
	go s.loadRequestHandler()
	// go s.requestHandler()
	return nil
}

func (s *Slave) loadRequestHandler() {
	s.closeWait.Add(1)

	address := s.ip + ":" + strconv.Itoa(int(s.loadReqPort))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		close(s.close)
	}

	go s.loadRecvAndUpdater(conn)

	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			packet := packets.LoadRequestPacket{}
			bytes, err := packets.EncodePacket(packet, packets.LoadRequest)
			if err != nil {
				s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to encode packet",
					"slave_ip", s.ip, "err", err.Error()))
			}

			_, err = conn.Write(bytes)
			if err != nil {
				s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send LoadReq packet",
					"slave_ip", s.ip, "err", err.Error()))
			}

			<-time.After(constants.LoadRequestInterval)
		}
	}

	s.closeWait.Done()
}

func (s *Slave) loadRecvAndUpdater(conn net.Conn) {
	s.closeWait.Add(1)
	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			var buf [2048]byte
			// TODO: add timeout
			n, err := conn.Read(buf[0:])
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP"))
				if err == io.EOF {
					// TODO: remove myself from slavepool
					s.Logger.Warning(logger.FormatLogMessage("msg", "Closing a slave", "slave_ip", s.ip,
						"slave_id", strconv.Itoa(int(s.id))))
					close(s.close)
					end = true
				}
				continue
			}

			packetType, err := packets.GetPacketType(buf[:n])
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
				return
			}

			switch packetType {
			case packets.LoadResponse:
				var p packets.LoadResponsePacket
				err := packets.DecodePacket(buf[:n], &p)
				if err != nil {
					s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
						"packet", packetType.String(), "err", err.Error()))
					return
				}

				s.UpdateLoad(p.Load, p.Timestamp)
				s.Logger.Info(logger.FormatLogMessage("msg", "Load updated",
					"slave_ip", s.ip, "slave_id", strconv.Itoa(int(s.id)), "load", strconv.FormatFloat(p.Load, 'E', -1, 64)))

			default:
				s.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
			}
		}
	}
	s.closeWait.Done()
}

// func (s *Slave) requestHandler() {
// 	s.closeWait.Add(1)
// 	s.closeWait.Done()
// }

func (s *Slave) Close() {
	close(s.close)
	s.closeWait.Wait()
}

// TODO: regularly send info request to all slaves.

//////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////
////////////////////////// SLAVE POOL ////////////////////////////
//////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////

type SlavePool struct {
	mtx    sync.RWMutex
	slaves []*Slave
	Logger *logging.Logger
}

func (sp *SlavePool) NumSlaves() int {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	return len(sp.slaves)
}

func (sp *SlavePool) AddSlave(slave *Slave) {
	// TODO: make connection with slave over the listeners.
	slave.Logger = sp.Logger
	slave.InitConnections()
	slave.InitDS()
	sp.mtx.Lock()
	defer sp.mtx.Unlock()
	sp.slaves = append(sp.slaves, slave)
}

func (sp *SlavePool) RemoveSlave(ip string) bool {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()
	toRemove := -1
	for i, slave := range sp.slaves {
		if slave.ip == ip {
			toRemove = i
			break
		}
	}

	if toRemove < 0 {
		return false
	}

	sp.slaves = append(sp.slaves[:toRemove], sp.slaves[toRemove+1:]...)
	return true
}

func (sp *SlavePool) SlaveExists(ip net.IP, id uint16) bool {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	ipStr := ip.String()
	for _, slave := range sp.slaves {
		if slave.ip == ipStr && slave.id == id {
			return true
		}
	}
	return false
}

func (sp *SlavePool) GetAllSlaveIPs() []string {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	var ips []string

	for _, s := range sp.slaves {
		ips = append(ips, s.ip)
	}

	return ips
}
func (sp *SlavePool) Close(log *logging.Logger) {
	// close all go routines/listeners
	log.Info(logger.FormatLogMessage("msg", "Closing Slave Pool"))
	for _, s := range sp.slaves {
		s.Close()
	}
}

func (sp *SlavePool) gc(log *logging.Logger) {
	toRemove := []int{}
	for i, slave := range sp.slaves {
		select {
		case <-slave.close:
			slave.closeWait.Wait()
			toRemove = append(toRemove, i)
		default:
		}
	}

	if len(toRemove) > 0 {
		sp.mtx.Lock()
		defer sp.mtx.Unlock()

		for i, idx := range toRemove {
			log.Info(logger.FormatLogMessage("msg", "Slave removed in gc",
				"slave_ip", sp.slaves[idx-i].ip, "slave_id", strconv.Itoa(int(sp.slaves[idx-i].id))))
			sp.slaves = append(sp.slaves[:idx-i], sp.slaves[idx+1-i:]...)
		}

	}
}
