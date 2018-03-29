all: build

build: build_master build_slave build_monitoring

build_master:
	go build ./cmd/master

build_slave:
	go build ./cmd/slave

build_monitoring:
	go build ./cmd/monitoring