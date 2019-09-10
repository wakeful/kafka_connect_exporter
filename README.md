# Kafka connect exporter

A [Prometheus](https://prometheus.io/) exporter that collects [Kafka connect](https://docs.confluent.io/current/connect/index.html) metrics.

### Usage

```sh
$ ./kafka_connect_exporter -h
Usage of ./kafka_connect_exporter:
  -listen-address string
        Address on which to expose metrics. (default ":8080")
  -scrape-uri string
        URI on which to scrape kafka connect. (default "http://127.0.0.1:8080")
  -telemetry-path string
        Path under which to expose metrics. (default "/metrics")
  -version
        show version and exit
```

## Metrics

```
# HELP kafka_connect_connector_state_running is the connector running?
# TYPE kafka_connect_connector_state_running gauge
kafka_connect_connector_state_running{connector="test-changesets",state="running",worker="kafka-connect:8083"} 1
# HELP kafka_connect_connector_tasks_state the state of tasks. 0-failed, 1-running, 2-unassigned, 3-paused
# TYPE kafka_connect_connector_tasks_state gauge
kafka_connect_connector_tasks_state{connector="test-changesets",state="running",worker_id="kafka-connect:8083"} 1
# HELP kafka_connect_connectors_count number of deployed connectors
# TYPE kafka_connect_connectors_count gauge
kafka_connect_connectors_count 1
# HELP kafka_connect_up was the last scrape of kafka connect successful?
# TYPE kafka_connect_up gauge
kafka_connect_up 1
```
