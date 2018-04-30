package monitoring

import (
	"bytes"
	"text/template"
)

type addDatasourceTmplObj struct {
	Name string
	URL  string
}

var addDatasourceTmpl *template.Template = template.Must(template.New("addData").Parse(`
	{
		"name":"{{.Name}}",
		"type":"prometheus",
		"url":"{{.URL}}",
		"access":"direct",
		"basicAuth":false
	}`))

type Datasource struct {
	IP    string
	Name  string
	Label string
}

type addDashboardTmplObj struct {
	Datasources []Datasource
}

var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
	"y_axis": func(x int) int {
		return 9 * x
	},
	"cpu_id": func(x int) int {
		return x * 2
	},
	"custom_id": func(x int) int {
		return (x * 2) + 1
	},
}

var addDashboardTmpl *template.Template = template.Must(template.New("addDashboard").Funcs(fns).Parse(`
{
	{{$n := len .Datasources}}
	"dashboard": {
		"__inputs": [
			{{range $i, $e := .Datasources }}
			{
				"name": "{{$e.Name}}",
				"label": "{{$e.Label}}",
				"description": "",
				"type": "datasource",
				"pluginId": "prometheus",
				"pluginName": "Prometheus"
				} {{if ne (plus1 $i) $n}},{{end}}
			{{- end}}
		],
		"__requires": [
			{
				"type": "grafana",
				"id": "grafana",
				"name": "Grafana",
				"version": "5.0.4"
			},
			{
				"type": "panel",
				"id": "graph",
				"name": "Graph",
				"version": "5.0.0"
			},
			{
				"type": "datasource",
				"id": "prometheus",
				"name": "Prometheus",
				"version": "5.0.0"
			}
		],
		"annotations": {
			"list": [
				{
					"builtIn": 1,
					"datasource": "-- Grafana --",
					"enable": true,
					"hide": true,
					"iconColor": "rgba(0, 211, 255, 1)",
					"name": "Annotations & Alerts",
					"type": "dashboard"
				}
			]
		},
		"editable": true,
		"gnetId": null,
		"graphTooltip": 0,
		"id": null,
		"links": [],
		"panels": [
			{{range $i, $e := .Datasources }}
	
			{
				"aliasColors": {},
				"bars": false,
				"dashLength": 10,
				"dashes": false,
				"datasource": "{{$e.Label}}",
				"fill": 1,
				"gridPos": {
					"h": 9,
					"w": 12,
					"x": 0,
					"y": {{(y_axis $i)}}
				},
				"id": {{(cpu_id $i)}},
				"legend": {
					"avg": false,
					"current": false,
					"max": false,
					"min": false,
					"show": true,
					"total": false,
					"values": false
				},
				"lines": true,
				"linewidth": 1,
				"links": [],
				"nullPointMode": "null",
				"percentage": false,
				"pointradius": 5,
				"points": false,
				"renderer": "flot",
				"seriesOverrides": [],
				"spaceLength": 10,
				"stack": false,
				"steppedLine": false,
				"targets": [
					{
						"expr": "rate(node_cpu_seconds_total{instance=\"localhost:9100\",job=\"node_exporter\",mode=\"user\"}[10s])",
						"format": "time_series",
						"intervalFactor": 1,
						"refId": "A"
					}
				],
				"thresholds": [],
				"timeFrom": null,
				"timeShift": null,
				"title": "Slave CPU Usage {{$e.IP}}",
				"tooltip": {
					"shared": true,
					"sort": 0,
					"value_type": "individual"
				},
				"type": "graph",
				"xaxis": {
					"buckets": null,
					"mode": "time",
					"name": null,
					"show": true,
					"values": []
				},
				"yaxes": [
					{
						"format": "short",
						"label": null,
						"logBase": 1,
						"max": null,
						"min": null,
						"show": true
					},
					{
						"format": "short",
						"label": null,
						"logBase": 1,
						"max": null,
						"min": null,
						"show": true
					}
				]
				},
				{
					"aliasColors": {},
					"bars": false,
					"dashLength": 10,
					"dashes": false,
					"datasource": "{{$e.Label}}",
					"fill": 1,
					"gridPos": {
						"h": 9,
						"w": 12,
						"x": 12,
						"y": {{(y_axis $i)}}
					},
					"id": {{(custom_id $i)}},
					"legend": {
						"avg": false,
						"current": false,
						"max": false,
						"min": false,
						"show": true,
						"total": false,
						"values": false
					},
					"lines": true,
					"linewidth": 1,
					"links": [],
					"nullPointMode": "null",
					"percentage": false,
					"pointradius": 5,
					"points": false,
					"renderer": "flot",
					"seriesOverrides": [],
					"spaceLength": 10,
					"stack": false,
					"steppedLine": false,
					"targets": [
						{
							"expr": "sum_over_time(tasks_requested[10s])",
							"format": "time_series",
							"intervalFactor": 1
						},
						{
							"expr": "sum_over_time(tasks_completed[10s])",
							"format": "time_series",
							"intervalFactor": 1
						},
						{
							"expr": "current_load",
							"format": "time_series",
							"intervalFactor": 1
						}
					],
					"thresholds": [],
					"timeFrom": null,
					"timeShift": null,
					"title": "Slave Task Count {{$e.IP}}",
					"tooltip": {
						"shared": true,
						"sort": 0,
						"value_type": "individual"
					},
					"type": "graph",
					"xaxis": {
						"buckets": null,
						"mode": "time",
						"name": null,
						"show": true,
						"values": []
					},
					"yaxes": [
						{
							"format": "short",
							"label": null,
							"logBase": 1,
							"max": null,
							"min": null,
							"show": true
						},
						{
							"format": "short",
							"label": null,
							"logBase": 1,
							"max": null,
							"min": null,
							"show": true
						}
					]
					} {{if ne (plus1 $i) $n}},{{end}}
	
			{{- end}}
	
	
		],
		"refresh": false,
		"schemaVersion": 16,
		"style": "dark",
		"tags": [],
		"templating": {
			"list": []
		},
		"time": {
			"from": "now-5m",
			"to": "now"
		},
		"timepicker": {
			"refresh_intervals": [ "5s", "10s", "30s", "1m", "5m", "15m", "30m", "1h", "2h", "1d"],
			"time_options": ["5m", "15m", "1h", "6h", "12h", "24h", "2d", "7d", "30d" ]
		},
		"timezone": "",
		"title": "Load Balancer"
	},

	"folderId": 0,
	"overwrite": true
}


`))

func grafanaAddDatasourceBody(ip string) (string, error) {
	name := datasourceLabel(ip)
	url := "http://" + ip + ":9090"
	buf := new(bytes.Buffer)

	err := addDatasourceTmpl.Execute(buf, addDatasourceTmplObj{name, url})
	return buf.String(), err
}

func grafanaUpdateDashboardBody(tmplObj addDashboardTmplObj) (string, error) {
	buf := new(bytes.Buffer)
	err := addDashboardTmpl.Execute(buf, tmplObj)
	return buf.String(), err
}
