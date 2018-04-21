package master

import (
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

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (m *Master) initDS() {
	m.close = make(chan struct{})
	m.unackedSlaves = make(map[string]struct{})
	m.slavePool = &SlavePool{
		Logger: m.Logger,
	}
}

func (m *Master) Run() {
	m.initDS()
	m.updateAddress()
	go m.connect()
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

func (m *Master) SlaveIpExists(ip net.IP) bool {
	return m.slavePool.SlaveIpExists(ip)
}

func (m *Master) Close() {
	m.Logger.Info(logger.FormatLogMessage("msg", "Closing Master gracefully..."))
	select {
	case <-m.close:
	default:
		close(m.close)
	}
	m.slavePool.Close(m.Logger)
	m.closeWait.Wait()
}

// Slave is used to store info of slave node connected to it
type Slave struct {
	ip          string
	infoReqPort uint16
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
	go s.infoHandler()
	// go s.requestHandler()
	return nil
}

func (s *Slave) infoHandler() {
	s.closeWait.Add(1)

	address := s.ip + ":" + strconv.Itoa(int(s.infoReqPort))
	conn, err := net.Dial("tcp", address)
	if err != nil {
		close(s.close)
	}

	go s.infoUpdater(conn)

	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			packet := packets.InfoRequestPacket{}
			bytes, err := packets.EncodePacket(packet, packets.InfoRequest)
			if err != nil {
				s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to encode packet",
					"slave_ip", s.ip, "err", err.Error()))
			}

			_, err = conn.Write(bytes)
			if err != nil {
				s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send InfoReq packet",
					"slave_ip", s.ip, "err", err.Error()))
			}

			<-time.After(constants.InfoRequestInterval)
		}
	}

	s.closeWait.Done()
}

func (s *Slave) infoUpdater(conn net.Conn) {
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
					s.Logger.Warning(logger.FormatLogMessage("msg", "Closing a slave", "slave_ip", s.ip))
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
			case packets.InfoResponse:
				var p packets.InfoResponsePacket
				err := packets.DecodePacket(buf[:n], &p)
				if err != nil {
					s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
						"packet", packetType.String(), "err", err.Error()))
					return
				}

				s.UpdateLoad(p.Load, p.Timestamp)
				s.Logger.Info(logger.FormatLogMessage("msg", "Load updated",
					"slave_ip", s.ip, "load", strconv.FormatFloat(p.Load, 'E', -1, 64)))

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

type SlavePool struct {
	mtx    sync.RWMutex
	slaves []*Slave
	Logger *logging.Logger
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

func (sp *SlavePool) SlaveIpExists(ip net.IP) bool {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	ipStr := ip.String()
	for _, slave := range sp.slaves {
		if slave.ip == ipStr {
			return true
		}
	}
	return false
}

func (sp *SlavePool) Close(log *logging.Logger) {
	// close all go routines/listeners
	log.Info(logger.FormatLogMessage("msg", "Closing Slave Pool"))
	for _, s := range sp.slaves {
		s.Close()
	}
}
