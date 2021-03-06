package master

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/op/go-logging"
)

type HTTPOptions struct {
	Logger *logging.Logger
}

type Handler struct {
	m      *Master
	server *http.Server
	opts   *HTTPOptions
}

func (m *Master) StartServer(opts *HTTPOptions) {

	m.serverHandler = &Handler{
		m:    m,
		opts: opts,
	}

	listenPortStr := ":" + strconv.Itoa(int(constants.HTTPServerPort))
	m.serverHandler.server = &http.Server{Addr: listenPortStr}

	http.HandleFunc("/ok", m.serverHandler.serverOk)
	http.HandleFunc("/fibonacii", m.serverHandler.fibonaciiHandler(m))
	http.HandleFunc("/cprimt", m.serverHandler.cprimeHandler(m))

	m.Logger.Info(logger.FormatLogMessage("msg", "Starting the server"))

	m.closeWait.Add(1)
	go func() {
		if err := m.serverHandler.server.ListenAndServe(); err != nil {
			m.Logger.Error(logger.FormatLogMessage("msg", "ListenAndServe()", "err", err.Error()))
			select {
			case <-m.close:
			default:
				close(m.close)
			}
		}
		m.Logger.Info(logger.FormatLogMessage("msg", "Shutting down server"))
		m.closeWait.Done()
	}()

}

func (h *Handler) serverOk(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Server is running")
}

func (h *Handler) fibonaciiHandler(m *Master) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		n, ok := r.URL.Query()["n"]
		if !ok || len(n) == 0 {
			w.WriteHeader(400)
			fmt.Fprint(w, "Needs parameter n")
			return
		}
		nInt, err := strconv.Atoi(n[0])
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprint(w, "Parameters are improper")
			return
		}

		t := packets.TaskPacket{
			TaskTypeID: packets.FibonacciTaskType,
			N:          nInt,
			Close:      make(chan struct{}),
		}
		err = m.assignNewTask(&t, uint64(nInt))
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprint(w, "Task lost")
		} else {
			select {
			case <-t.Close:
				fmt.Fprint(w, t.Result)
			case <-time.After(2 * time.Second):
				select {
				case <-t.Close:
				default:
					close(t.Close)
				}
				w.WriteHeader(500)
				fmt.Fprint(w, "Task lost")
			}
		}
	}
}

func (h *Handler) cprimeHandler(m *Master) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		n, ok := r.URL.Query()["n"]
		if !ok || len(n) == 0 {
			w.WriteHeader(400)
			fmt.Fprint(w, "Needs parameter n")
			return
		}
		nInt, err := strconv.Atoi(n[0])
		if err != nil {
			w.WriteHeader(400)
			fmt.Fprint(w, "Parameters are improper")
			return
		}

		t := packets.TaskPacket{
			TaskTypeID: packets.CountPrimesTaskType,
			N:          nInt,
			Close:      make(chan struct{}),
		}
		err = m.assignNewTask(&t, uint64(nInt))
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprint(w, "Task lost")
		} else {
			select {
			case <-t.Close:
				fmt.Fprint(w, t.Result)
			case <-time.After(2 * time.Second):
				select {
				case <-t.Close:
				default:
					close(t.Close)
				}
				w.WriteHeader(500)
				fmt.Fprint(w, "Task lost")
			}
		}
	}
}

func (h *Handler) Shutdown() error {
	return h.server.Shutdown(nil)
}
