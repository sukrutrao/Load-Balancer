package slave

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/op/go-logging"
)

type HTTPOptions struct {
	Logger *logging.Logger
}

type Handler struct {
	s      *Slave
	server *http.Server
	opts   *HTTPOptions
	Port   int
}

func (s *Slave) StartServer(opts *HTTPOptions) {

	s.serverHandler = &Handler{
		s:    s,
		opts: opts,
	}

	listenPortStr := ":" + strconv.Itoa(int(constants.HTTPServerPort))
	s.serverHandler.server = &http.Server{Addr: listenPortStr}

	http.HandleFunc("/ok", s.serverHandler.serverOk)
	http.HandleFunc("/metrics", s.serverHandler.metricHandler(s))

	s.Logger.Info(logger.FormatLogMessage("msg", "Starting the server"))

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		s.Logger.Error(logger.FormatLogMessage("msg", "Listen()", "err", err.Error()))
		select {
		case <-s.close:
		default:
			close(s.close)
		}
		return
	}

	s.closeWait.Add(1)
	go func() {
		if err := s.serverHandler.server.Serve(listener); err != nil {
			s.Logger.Error(logger.FormatLogMessage("msg", "Serve()", "err", err.Error()))
			select {
			case <-s.close:
			default:
				close(s.close)
			}
		}
		s.Logger.Info(logger.FormatLogMessage("msg", "Shutting down server"))
		s.closeWait.Done()
	}()

	f, err := os.OpenFile("/tmp/prometheus.yml", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		f.Close()
		panic(err)
	}
	f.Sync()

	// TODO: handle overwrite
	s.serverHandler.Port = listener.Addr().(*net.TCPAddr).Port
	text := "\n"
	text += "  - job_name: 'slave_" + strconv.Itoa(s.serverHandler.Port) + "'\n"
	text += "    static_configs:\n"
	text += "      - targets: ['localhost:" + strconv.Itoa(s.serverHandler.Port) + "']"
	if _, err = f.WriteString(text); err != nil {
		f.Close()
		panic(err)
	}

	f.Close()

	time.Sleep(5 * time.Second)
	cmd := exec.Command("curl", "-s", "-XPOST", "localhost:9090/-/reload")
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	s.Logger.Info(logger.FormatLogMessage("msg", "Server and metrics started"))
}

func (h *Handler) serverOk(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Server is running")
}

func (h *Handler) metricHandler(s *Slave) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "tasks_requested{instance=\"localhost:%d\"} %d\n", h.Port, s.metric.TasksRequested)
		fmt.Fprintf(w, "tasks_completed{instance=\"localhost:%d\"} %d\n", h.Port, s.metric.TasksCompleted)
	}
}

func (h *Handler) Shutdown() error {
	return h.server.Shutdown(nil)
}
