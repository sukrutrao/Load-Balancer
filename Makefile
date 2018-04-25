all: build_dependancies build

build_dependancies: build_prometheus build_node_exporter

build: build_master build_slave build_monitoring

build_master:
	@echo "Building master"
	@go build ./cmd/master

build_slave:
	@echo "Building slave"
	@go build ./cmd/slave

build_monitoring:
	@echo "Building monitoring"
	@go build ./cmd/monitoring

build_prometheus:
	@echo "Building prometheus"
	@go get github.com/prometheus/prometheus/cmd/prometheus

build_node_exporter:
	@echo "Building node_exporter"
	@go get github.com/prometheus/node_exporter

run_master:
	prometheus --config.file="config/prometheus.yml" 2> prometheus.log & ./master

run_slave:
	prometheus --config.file="config/prometheus.yml" 2> prometheus.log & ./slave

run_monitoring:
	sudo systemctl start grafana-server & ./monitoring