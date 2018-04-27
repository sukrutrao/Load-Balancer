package main

import (
	"fmt"
	"os"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/monitoring_src"
)

// Key eyJrIjoiWEZnaVhOS1hYcG9sMWtMd201NU5xbDNGU0tTNGd5aEUiLCJuIjoiQWRtaW4iLCJpZCI6MX0=

func main() {
	logger.SetLogLevel(logger.DEBUG)
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, "API key missing")
	}
	m := monitoring.Monitor{
		APIKey: os.Args[1],
		Logger: logger.NewLogger("monitoring"),
	}
	m.Run()
}
