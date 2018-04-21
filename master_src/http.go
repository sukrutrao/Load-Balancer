package master

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/GoodDeeds/load-balancer/common/constants"
)

type HandlerFuncPair struct {
	Path        string
	HandlerFunc func(w http.ResponseWriter, r *http.Request)
}

type HTTPOptions struct {
	Handlers []*HandlerFuncPair
}

var defaultOptions *HTTPOptions = &HTTPOptions{
	Handlers: []*HandlerFuncPair{
		{"/helloworld", helloWorld},
	},
}

func DefaultOptions() *HTTPOptions {
	return defaultOptions
}

func StartServer(opts *HTTPOptions) {

	for _, h := range opts.Handlers {
		http.HandleFunc(h.Path, h.HandlerFunc)
	}

	listenPortStr := ":" + strconv.Itoa(int(constants.HTTPServerPort))
	http.ListenAndServe(listenPortStr, nil)

}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello world!")
}
