# Oxidized Prometheus Exporter

## Description

Oxidized Prometheus Exporter to expose metrics from oxidized.

### Metrics
- oxidized_status: Status of oxidized connection, 1 = success, 0 = error
- oxidized_exporter_collect_duration: Time taken to collect metrics in ms  
- oxidized_device_status: Status of oxidized device, 2 = success, 1 = never, 0 = no connection
- oxidized_device_last_backup_time: Time of last backup in seconds
- oxidized_device_last_backup_start: Start time of last backup as unix timestamp
- oxidized_device_last_backup_end: End time of last backup as unix timestamp
- oxidized_device_last_backup_status: Status of last backup, 2 = success, 1 = error
- oxidized_device_config_size: Size of the device config in bytes
- oxidized_device_config_lines: Number of lines in the device config

## Installation
Under [releases](https://github.com/akquinet/oxidized-exporter/releases), you can find the latest binary, rpm & deb for Linux.

Alternatively, you can build the exporter yourself:

```bash
git clone htttps://github.com/akquinet/oxidized-exporter.git /tmp/oxidized-exporter
cd /tmp/oxidized-exporter
GOOS=linux GOARCH=amd64 CGO_ENALD=0 go build -o oxidized-exporter
```

### Container

A container image is available under [packages](https://github.com/akquinet/oxidized-exporter/pkgs/container/oxidized-exporter).

Example docker-compose configuration where the exporter is in the same network as oxidized:

```yaml
---
version: "3.5"

services:
  oxidized:
    restart: unless-stopped
    image: oxidized/oxidized:0.29.1
    ports:
      - 127.0.0.1:8088:8888
    environment:
      - "CONFIG_RELOAD_INTERVAL=600"
    volumes:
      - "./data:/home/oxidized/.config/oxidized"
    env_file:
      - ".env"
    networks:
      - default

  oxidized-exporter:
    restart: unless-stopped
    image: ghcr.io/akquinet/oxidized-exporter:latest
    command:
      - "--verbose"
    ports:
      - 127.0.0.1:8089:8080
    environment:
      - "OXIDIZED_EXPORTER_URL=http://oxidized:8888/oxidized"
    env_file:
      - ".env"
    networks:
      - default
```

## Usage

All options can be set via command line arguments or environment variables.
`OXIDIZED_EXPORTER` is the prefix for all environment variables.

```bash
# show all available options
$ oxidized-exporter --help
Oxidized exporter for Prometheus

Usage:
  oxidized-exporter [flags]

Flags:
  -d, --debug         Enable debug logging
  -h, --help          help for oxidized-exporter
  -p, --pass string   Password for oxidized API
      --path string   Path to expose metrics on (default "/metrics")
      --port int      Port to listen on (default 8080)
  -U, --url string    URL of oxidized API (default "http://localhost:8888")
  -u, --user string   Username for oxidized API
  -v, --verbose       Enable verbose logging

# run against an oxidized instance with basic authentication
$ oxidized-exporter --url "https://oxidized.mydomain.com" -u myuser -p mypass --verbose
```

## Prometheus Configuration

```yaml
- job_name: "oxidized_exporter"
  scrape_interval: 5m
  scrape_timeout: 300s
  scheme: "https"
  static_configs:
    - targets:
        - "oxidized.mydomain.com"
  basic_auth:
    username: "myuser"
    password: "mypass"
```

## Grafana Dashboard
A example Grafana dashboard is available under [docs/grafana-dashboard.json](docs/grafana-dashboard.json). It can be imported into Grafana.

![Grafana Dashboard](docs/grafana-dashboard.png)

### Features

- Filter for model, group and device
- Show successful, failed and never backed up devices
- Show backup duration
- Show group and model statistics
- Show number of lines and size of the config for groups, models and devices
