package main

import (
	"github.com/GoodDeeds/load-balancer/master_src"
)

func main() {
	m := master.Master{}
	m.Run()
}