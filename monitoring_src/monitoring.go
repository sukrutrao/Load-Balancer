package monitoring

import (
	"bytes"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node
type Master struct {
	ip net.IP
}

type Monitor struct {
	myIP        net.IP
	broadcastIP net.IP
	reqSendPort uint16
	master      Master
	APIKey      string

	Logger *logging.Logger

	slaveIPs       map[string]struct{}
	failedDeleteIP map[string]struct{}
	mtx            sync.RWMutex

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (mo *Monitor) initDS() {
	mo.slaveIPs = make(map[string]struct{})
	mo.failedDeleteIP = make(map[string]struct{})
	mo.close = make(chan struct{})
}

func (mo *Monitor) Run() {
	mo.initDS()
	mo.updateAddress()
	mo.Logger.Info(logger.FormatLogMessage("msg", "Monitor running"))
	mo.connect()
	mo.closeWait.Wait()
}

func (mo *Monitor) updateAddress() {
	ipnet, err := utility.GetMyIP()
	if err != nil {
		mo.Logger.Fatal(logger.FormatLogMessage("msg", "Failed to get IP", "err", err.Error()))
	}

	mo.myIP = ipnet.IP
	for i, b := range ipnet.Mask {
		mo.broadcastIP = append(mo.broadcastIP, (mo.myIP[i] | (^b)))
	}
}

func (mo *Monitor) UpdateSlaveIPs(slaveIPs []string) (bool, []string, []string) {
	mo.mtx.Lock()
	defer mo.mtx.Unlock()

	var deleted []string
	var added []string

	modified := false
	before := len(mo.slaveIPs)

	newMap := make(map[string]struct{})
	for _, sip := range slaveIPs {
		newMap[sip] = struct{}{}
		if _, ok := mo.slaveIPs[sip]; !ok {
			added = append(added, sip)
			modified = true
		}
	}

	for sip := range mo.slaveIPs {
		if _, ok := newMap[sip]; !ok {
			deleted = append(deleted, sip)
		}
	}
	mo.slaveIPs = newMap

	after := len(mo.slaveIPs)

	return (modified || before != after || len(mo.failedDeleteIP) > 0), added, deleted
}

func (mo *Monitor) UpdateGrafana(added, deleted []string) {
	mo.mtx.RLock()
	defer mo.mtx.RUnlock()
	mo.UpdateGrafanaDatasource(added, deleted)
	mo.UpdateGrafanaDashboard()
}

func (mo *Monitor) UpdateGrafanaDatasource(added, deleted []string) {
	mo.mtx.RLock()
	defer mo.mtx.RUnlock()

	client := &http.Client{}

	// Delete datasource
	for ip := range mo.failedDeleteIP {
		mo.Logger.Info(logger.FormatLogMessage("msg", "Deleting datasource", "ip", ip))
		statusCode, err := mo.deleteDatasource(client, ip)
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Delete datasource failed", "err", err.Error(), "ip", ip))
		} else if statusCode != 200 {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Delete datasource failed", "status", strconv.Itoa(statusCode), "ip", ip))
		} else {
			delete(mo.failedDeleteIP, ip)
		}
	}
	for _, ip := range deleted {
		mo.Logger.Info(logger.FormatLogMessage("msg", "Deleting datasource", "ip", ip))
		statusCode, err := mo.deleteDatasource(client, ip)
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Delete datasource failed", "err", err.Error(), "ip", ip))
			mo.failedDeleteIP[ip] = struct{}{}
		} else if statusCode != 200 {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Delete datasource failed", "status", strconv.Itoa(statusCode), "ip", ip))
			mo.failedDeleteIP[ip] = struct{}{}
		}
	}

	// Add datasource
	for _, ip := range added {
		mo.Logger.Info(logger.FormatLogMessage("msg", "Adding datasource", "ip", ip))
		body, err := grafanaAddDatasourceBody(ip)
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Add datasource failed", "err", err.Error(), "ip", ip))
			continue
		}
		req, err := http.NewRequest("POST", "http://localhost:3000/api/datasources", bytes.NewBuffer([]byte(body)))
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Add datasource failed", "err", err.Error(), "ip", ip))
			continue
		}
		mo.setJsonAndAuthHeaders(req)
		resp, err := client.Do(req)
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Add datasource failed", "err", err.Error(), "ip", ip))
			continue
		}
		if resp.StatusCode != 200 {
			mo.Logger.Error(logger.FormatLogMessage("msg", "Add datasource failed", "status", strconv.Itoa(resp.StatusCode), "ip", ip))
			continue
		}
	}

}

func (mo *Monitor) deleteDatasource(client *http.Client, ip string) (int, error) {
	req, err := http.NewRequest("DELETE", grafanaDeleteURL(ip), nil)
	if err != nil {
		return 0, err
	}
	mo.setJsonAndAuthHeaders(req)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

func (mo *Monitor) UpdateGrafanaDashboard() {
	mo.mtx.RLock()
	defer mo.mtx.RUnlock()

	mo.Logger.Info(logger.FormatLogMessage("msg", "Updating dashboard"))
	tmplObj := addDashboardTmplObj{}
	for sip := range mo.slaveIPs {
		tmplObj.Datasources = append(tmplObj.Datasources, Datasource{
			IP:    sip,
			Name:  datasourceName(sip),
			Label: datasourceLabel(sip),
		})
	}

	body, err := grafanaUpdateDashboardBody(tmplObj)
	if err != nil {
		mo.Logger.Error(logger.FormatLogMessage("msg", "Update dashboard failed", "err", err.Error()))
		return
	}

	req, err := http.NewRequest("POST", "http://localhost:3000/api/dashboards/db", bytes.NewBuffer([]byte(body)))
	if err != nil {
		mo.Logger.Error(logger.FormatLogMessage("msg", "Update dashboard failed", "err", err.Error()))
		return
	}
	mo.setJsonAndAuthHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		mo.Logger.Error(logger.FormatLogMessage("msg", "Update dashboard failed", "err", err.Error()))
		return
	}

	if resp.StatusCode != 200 {
		mo.Logger.Error(logger.FormatLogMessage("msg", "Update dashboard failed", "status", strconv.Itoa(resp.StatusCode)))
		return
	}

}

func (mo *Monitor) setJsonAndAuthHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+mo.APIKey)
}

func datasourceLabel(ip string) string {
	return "slave_" + strings.Replace(ip, ".", "", -1)
}

func datasourceName(ip string) string {
	return "DS_SLAVE_" + strings.Replace(ip, ".", "", -1)
}

func grafanaDeleteURL(datasourceIP string) string {
	return "http://localhost:3000/api/datasources/name/slave_" + strings.Replace(datasourceIP, ".", "", -1)
}
