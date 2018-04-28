package slave

import (
	"fmt"
	"net/http"
	"strconv"

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
}

func (s *Slave) StartServer(opts *HTTPOptions) {

	s.serverHandler = &Handler{
		s:    s,
		opts: opts,
	}

	listenPortStr := ":" + strconv.Itoa(int(constants.MetricServerPort))
	s.serverHandler.server = &http.Server{Addr: listenPortStr}

	http.HandleFunc("/ok", s.serverHandler.serverOk)
	http.HandleFunc("/metric", s.serverHandler.metricHandler(s))

	s.Logger.Info(logger.FormatLogMessage("msg", "Starting the server"))

	s.closeWait.Add(1)
	go func() {
		if err := s.serverHandler.server.ListenAndServe(); err != nil {
			s.Logger.Error(logger.FormatLogMessage("msg", "ListenAndServe()", "err", err.Error()))
			select {
			case <-s.close:
			default:
				close(s.close)
			}
		}
		s.Logger.Info(logger.FormatLogMessage("msg", "Shutting down server"))
		s.closeWait.Done()
	}()

}

func (h *Handler) serverOk(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Server is running")
}

func (h *Handler) metricHandler(s *Slave) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "tasks_requested %d\n", s.metric.TasksRequested)
		fmt.Fprintf(w, "tasks_completed %d\n", s.metric.TasksCompleted)
	}
}

func (h *Handler) Shutdown() error {
	return h.server.Shutdown(nil)
}
