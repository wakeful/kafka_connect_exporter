package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
)

const nameSpace = "kafka_connect"

var (
	version    = "dev"
	versionUrl = "https://github.com/wakeful/kafka_connect_exporter"

	showVersion   = flag.Bool("version", false, "show version and exit")
	listenAddress = flag.String("listen-address", ":8080", "Address on which to expose metrics.")
	metricsPath   = flag.String("telemetry-path", "/metrics", "Path under which to expose metrics.")
	scrapeURI     = flag.String("scrape-uri", "http://127.0.0.1:8080", "URI on which to scrape kafka connect.")

	isConnectorRunning = prometheus.NewDesc(
		prometheus.BuildFQName(nameSpace, "connector", "state_running"),
		"is the connector running?",
		[]string{"connector", "state", "worker"}, nil)
	areConnectorTasksRunning = prometheus.NewDesc(
		prometheus.BuildFQName(nameSpace, "connector", "tasks_state"),
		"the state of tasks. 0-failed, 1-running, 2-unassigned, 3-paused",
		[]string{"connector", "state", "worker_id", "id"}, nil)
)

type connectors []string

type status struct {
	Name      string    `json:"name"`
	Connector connector `json:"connector"`
	Tasks     []task    `json:"tasks"`
}

type connector struct {
	State    string `json:"state"`
	WorkerId string `json:"worker_id"`
}

type task struct {
	State    string  `json:"state"`
	Id       float64 `json:"id"`
	WorkerId string  `json:"worker_id"`
}

type Exporter struct {
	URI             string
	up              prometheus.Gauge
	connectorsCount prometheus.Gauge
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {

	client := http.Client{
		Timeout: 3 * time.Second,
	}
	e.up.Set(0)

	response, err := client.Get(e.URI + "/connectors")
	if err != nil {
		log.Errorf("Can't scrape kafka connect: %v", err)
		ch <- e.up
		return
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			log.Errorf("Can't close connection to kafka connect: %v", err)
			ch <- e.up
			return
		}
	}()

	output, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Can't scrape kafka connect: %v", err)
		ch <- e.up
		return
	}

	var connectorsList connectors
	if err := json.Unmarshal(output, &connectorsList); err != nil {
		log.Errorf("Can't scrape kafka connect: %v", err)
		ch <- e.up
		return
	}

	e.up.Set(1)
	e.connectorsCount.Set(float64(len(connectorsList)))

	ch <- e.up
	ch <- e.connectorsCount

	for _, connector := range connectorsList {

		connectorStatusResponse, err := client.Get(e.URI + "/connectors/" + connector + "/status")
		if err != nil {
			log.Errorf("Can't get /status for: %v", err)
			continue
		}

		connectorStatusOutput, err := ioutil.ReadAll(connectorStatusResponse.Body)
		if err != nil {
			log.Errorf("Can't read Body for: %v", err)
			continue
		}

		var connectorStatus status
		if err := json.Unmarshal(connectorStatusOutput, &connectorStatus); err != nil {
			log.Errorf("Can't decode response for: %v", err)
			continue
		}

		var isRunning float64 = 0
		if strings.ToLower(connectorStatus.Connector.State) == "running" {
			isRunning = 1
		}

		ch <- prometheus.MustNewConstMetric(
			isConnectorRunning, prometheus.GaugeValue, isRunning,
			connectorStatus.Name, strings.ToLower(connectorStatus.Connector.State), connectorStatus.Connector.WorkerId,
		)

		for _, connectorTask := range connectorStatus.Tasks {

			var state float64
			switch taskState := strings.ToLower(connectorTask.State)
			taskState {
			case "running":
			    state = 1
			case "unassigned":
			    state = 2
			case "paused":
			    state = 3
			default:
			    state = 0
			}

			ch <- prometheus.MustNewConstMetric(
				areConnectorTasksRunning, prometheus.GaugeValue, state,
				connectorStatus.Name, strings.ToLower(connectorTask.State), connectorTask.WorkerId, fmt.Sprintf("%d", int(connectorTask.Id)),
			)
		}

		err = connectorStatusResponse.Body.Close()
		if err != nil {
			log.Errorf("Can't close connection to connector: %v", err)
		}
	}

	return
}

func NewExporter(uri string) *Exporter {
	log.Infoln("Collecting data from:", uri)

	return &Exporter{
		URI: uri,
		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Name:      "up",
			Help:      "was the last scrape of kafka connect successful?",
		}),
		connectorsCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: nameSpace,
			Subsystem: "connectors",
			Name:      "count",
			Help:      "number of deployed connectors",
		}),
	}

}

var supportedSchema = map[string]bool{
	"http":  true,
	"https": true,
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("kafka_connect_exporter\n url: %s\n version: %s\n", versionUrl, version)
		os.Exit(2)
	}

	parseURI, err := url.Parse(*scrapeURI)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	if !supportedSchema[parseURI.Scheme] {
		log.Error("schema not supported")
		os.Exit(1)
	}

	log.Infoln("Starting kafka_connect_exporter")

	prometheus.Unregister(prometheus.NewGoCollector())
	prometheus.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	prometheus.MustRegister(NewExporter(*scrapeURI))

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}
