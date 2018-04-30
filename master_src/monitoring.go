package master

import (
	"errors"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/op/go-logging"
)

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
	select {
	case <-mo.close:
	default:
		close(mo.close)
	}
	mo.closeWait.Wait()
}
