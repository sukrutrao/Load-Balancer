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
	"github.com/op/go-logging"
)

// Slave is used to store info of slave node connected to it
type Slave struct {
	ip          string
	id          uint16
	loadReqPort uint16
	reqSendPort uint16
	reqRecvPort uint16
	Logger      *logging.Logger
	maxLoad     uint64
	currentLoad uint64

	sendChan        chan packets.PacketTransmit
	tasksUndertaken []int

	lastLoadTimestamp time.Time
	mtx               sync.RWMutex

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (s *Slave) UpdateLoad(l uint64, ml uint64, ts time.Time) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if ts.After(s.lastLoadTimestamp) {
		s.currentLoad = l
		s.maxLoad = ml
		s.lastLoadTimestamp = ts
	}
}

func (s *Slave) InitDS() {
	s.close = make(chan struct{})
	s.sendChan = make(chan packets.PacketTransmit)
	//	s.recvChan = make(chan struct{})
	//	go s.sendChannelHandler()
	//	go s.recvChannelHandler()
}

func (s *Slave) InitConnections() error {
	s.closeWait.Add(1)
	go s.loadRequestHandler()
	s.closeWait.Add(1)
	go s.taskRequestHandler()
	return nil
}

func (s *Slave) loadRequestHandler() {

	address := s.ip + ":" + strconv.Itoa(int(s.loadReqPort))
	// s.Logger.Info(logger.FormatLogMessage("loadReqPort", strconv.Itoa(int(s.loadReqPort)), "reqSendPort", strconv.Itoa(int(s.reqSendPort))))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		close(s.close)
	}

	s.closeWait.Add(1)
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
	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			var buf [2048]byte
			conn.SetReadDeadline(time.Now().Add(constants.ReceiveTimeout))
			n, err := conn.Read(buf[0:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else if err != nil {
				if err == io.EOF {
					s.Logger.Warning(logger.FormatLogMessage("msg", "Closing a slave (load handler)", "slave_ip", s.ip,
						"slave_id", strconv.Itoa(int(s.id))))
					select {
					case <-s.close:
					default:
						close(s.close)
					}
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

				s.UpdateLoad(p.Load, p.MaxLoad, p.Timestamp)
				// s.Logger.Info(logger.FormatLogMessage("msg", "Load updated",
				// 	"slave_ip", s.ip, "slave_id", strconv.Itoa(int(s.id)), "load", strconv.FormatFloat(p.Load, 'E', -1, 64)))

			default:
				s.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
			}
		}
	}
	s.closeWait.Done()
}

func (s *Slave) taskRequestHandler() {

	{
		address := s.ip + ":" + strconv.Itoa(int(s.reqSendPort))
		// s.Logger.Info(logger.FormatLogMessage("loadReqPort", strconv.Itoa(int(s.loadReqPort)), "reqSendPort", strconv.Itoa(int(s.reqSendPort))))

		connSend, err := net.Dial("tcp", address)
		if err != nil {
			s.Logger.Fatal(logger.FormatLogMessage("Error!", "Error!"))
			close(s.close)
		}

		s.closeWait.Add(1)
		go s.sendChannelHandler(connSend)
	}
	{
		address := s.ip + ":" + strconv.Itoa(int(s.reqRecvPort))
		// s.Logger.Info(logger.FormatLogMessage("loadReqPort", strconv.Itoa(int(s.loadReqPort)), "reqSendPort", strconv.Itoa(int(s.reqSendPort))))

		connRecv, err := net.Dial("tcp", address)
		if err != nil {
			s.Logger.Fatal(logger.FormatLogMessage("Error!", "Error!"))
			close(s.close)
		}

		s.closeWait.Add(1)
		go s.taskRecvAndUpdater(connRecv)
	}

	s.closeWait.Done()
}

type tcpData struct {
	n   int
	buf [2048]byte
}

func (s *Slave) collectIncomingRequests(conn net.Conn, packetChan chan<- tcpData) {
	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			var buf [2048]byte
			conn.SetReadDeadline(time.Now().Add(constants.SlaveReceiveTimeout))
			n, err := conn.Read(buf[0:])
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				continue
			} else if err != nil {
				// s.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP", "err", err.Error()))
				if err == io.EOF {
					s.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP", "err", err.Error()))
					select {
					case <-s.close:
					default:
						close(s.close)
					}
					end = true
				}
				continue
			}

			var bufCopy [2048]byte
			copy(bufCopy[:], buf[:])
			packetChan <- tcpData{
				n:   n,
				buf: bufCopy,
			}
		}
	}

}

func (s *Slave) taskRecvAndUpdater(conn net.Conn) {

	// collect packets
	packetChan := make(chan tcpData)
	go s.collectIncomingRequests(conn, packetChan)

	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		case packet, ok := <-packetChan:
			if !ok {
				break
			}

			packetType, err := packets.GetPacketType(packet.buf[:packet.n])
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
				return
			}

			switch packetType {
			case packets.TaskRequestResponse:
				var p packets.TaskRequestResponsePacket
				err := packets.DecodePacket(packet.buf[:packet.n], &p)
				if err != nil {
					s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
						"packet", packetType.String(), "err", err.Error()))
					return
				}

				go s.handleTaskRequestResponse(p)

			case packets.TaskResultResponse:
				var p packets.TaskResultResponsePacket
				err := packets.DecodePacket(packet.buf[:packet.n], &p)
				if err != nil {
					s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
						"packet", packetType.String(), "err", err.Error()))
					return
				}
				go s.handleTaskResult(p)

			case packets.TaskStatusResponse:
				var p packets.TaskStatusResponsePacket
				err := packets.DecodePacket(packet.buf[:packet.n], &p)
				if err != nil {
					s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
						"packet", packetType.String(), "err", err.Error()))
					return
				}

				go s.handleTaskStatusResponse(p)

			default:
				s.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
			}

		case <-time.After(constants.WaitForReqTimeout):

		}
	}
	s.closeWait.Done()
}

func (s *Slave) Close() {
	select {
	case <-s.sendChan:
	default:
		close(s.sendChan)
	}
	select {
	case <-s.close:
	default:
		close(s.close)
	}
	s.closeWait.Wait()
}

func (s *Slave) sendChannelHandler(conn net.Conn) {
	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			pt, ok := <-s.sendChan
			if ok {
				bytes, err := packets.EncodePacket(pt.Packet, pt.PacketType)
				if err != nil {
					s.Logger.Error(logger.FormatLogMessage("msg", "Error in reading packet to send", "err", err.Error()))
					continue
				}
				_, err = conn.Write(bytes)
				if err != nil {
					s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send packet",
						"slave_ip", s.ip, "err", err.Error()))
				}
			} else {
				select {
				case <-s.sendChan:
				default:
					select {
					case <-s.close:
					default:
						close(s.close)
						end = true
					}
				}
			}

		}
	}
	s.closeWait.Done()
}
