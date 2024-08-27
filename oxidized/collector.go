package oxidized

import (
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// MetricSuccess represents a successful status.
	MetricSuccess = 2
	// MetricNever represents a "never" status.
	MetricNever = 1
	// MetricError represents an error status.
	MetricError = 0
)

// OxidizedCollector is a prometheus collector for oxidized
// It implements the prometheus.Collector interface
// See https://godoc.org/github.com/prometheus/client_golang/prometheus#Collector
type OxidizedCollector struct {
	oxidizedClient                        *OxidizedClient
	oxidizedStatusMetric                  *prometheus.Desc
	deviceStatusMetric                    *prometheus.Desc
	deviceLastBackupTimeMetric            *prometheus.Desc
	deviceLastBackupStartMetric           *prometheus.Desc
	deviceLastBackupEndMetric             *prometheus.Desc
	deviceLastBackupStatusMetric          *prometheus.Desc
	deviceConfigSizeMetric                *prometheus.Desc
	deviceConfigLinesMetric               *prometheus.Desc
	oxidizedExporterCollectDurationMetric *prometheus.Desc
}

// NewOxidizedCollector creates a new collector
// and creates the descriptors for the metrics
func NewOxidizedCollector(oxiClient *OxidizedClient) *OxidizedCollector {
	return &OxidizedCollector{
		oxidizedClient:                        oxiClient,
		oxidizedStatusMetric:                  prometheus.NewDesc("oxidized_status", "Status of oxidized connection, 1 = success, 0 = error", []string{}, nil),
		oxidizedExporterCollectDurationMetric: prometheus.NewDesc("oxidized_exporter_collect_duration", "Time taken to collect metrics in ms", []string{}, nil),
		deviceStatusMetric:                    prometheus.NewDesc("oxidized_device_status", "Status of oxidized device, 2 = success, 1 = never, 0 = no connection", []string{"full_name", "name", "group", "model"}, nil),
		deviceLastBackupTimeMetric:            prometheus.NewDesc("oxidized_device_last_backup_time", "Time of last backup in seconds", []string{"full_name", "name", "group", "model"}, nil),
		deviceLastBackupStartMetric:           prometheus.NewDesc("oxidized_device_last_backup_start", "Start time of last backup as unix timestamp", []string{"full_name", "name", "group", "model"}, nil),
		deviceLastBackupEndMetric:             prometheus.NewDesc("oxidized_device_last_backup_end", "End time of last backup as unix timestamp", []string{"full_name", "name", "group", "model"}, nil),
		deviceLastBackupStatusMetric:          prometheus.NewDesc("oxidized_device_last_backup_status", "Status of last backup, 2 = success, 1 = error", []string{"full_name", "name", "group", "model"}, nil),
		deviceConfigSizeMetric:                prometheus.NewDesc("oxidized_device_config_size", "Size of the device config in bytes", []string{"full_name", "name", "group", "model"}, nil),
		deviceConfigLinesMetric:               prometheus.NewDesc("oxidized_device_config_lines", "Number of lines in the device config", []string{"full_name", "name", "group", "model"}, nil),
	}
}

// Describe sends the super-set of all possible descriptors of metrics
func (c *OxidizedCollector) Describe(ch chan<- *prometheus.Desc) {
	// Update this section with the each metric you create for a given collector
	ch <- c.oxidizedStatusMetric
	ch <- c.deviceStatusMetric
	ch <- c.oxidizedExporterCollectDurationMetric
	ch <- c.deviceLastBackupTimeMetric
	ch <- c.deviceLastBackupStartMetric
	ch <- c.deviceLastBackupEndMetric
	ch <- c.deviceLastBackupStatusMetric
	ch <- c.deviceConfigSizeMetric
	ch <- c.deviceConfigLinesMetric
}

// Collect is called by the Prometheus registry when collecting metrics
func (c *OxidizedCollector) Collect(ch chan<- prometheus.Metric) {
	slog.Info("Collecting oxidized metrics")
	start_time := time.Now()
	devices, err := c.oxidizedClient.GetDevices()
	if err != nil {
		slog.Error("Could not get devices from oxidized", "error", err)
		// oxidized not reachable
		ch <- prometheus.MustNewConstMetric(c.oxidizedStatusMetric, prometheus.GaugeValue, 0)
		return
	}

	slog.Info("Got devices from oxidized", "count", len(devices))

	// oxidized reachable
	ch <- prometheus.MustNewConstMetric(c.oxidizedStatusMetric, prometheus.GaugeValue, 1)

	onlyDefaultGroup := c.oxidizedClient.OnlyDefaultGroup(devices)
	if onlyDefaultGroup {
		slog.Info("Oxidized has only devices of group default")
	}

	semaphore := make(chan struct{}, 100)
	wg := sync.WaitGroup{}
	for _, device := range devices {
		wg.Add(1)
		go func(device Device) {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
				wg.Done()
			}()

			var deviceStatus float64
			switch device.Status {
			case DeviceStatusSuccess:
				deviceStatus = MetricSuccess
			case DeviceStatusNever:
				deviceStatus = MetricNever
			case DeviceStatusNoConnection:
				deviceStatus = MetricError
			default:
				slog.Error("Unknown device status", "status", device.Status)
				deviceStatus = MetricError
			}
			ch <- prometheus.MustNewConstMetric(c.deviceStatusMetric, prometheus.GaugeValue, deviceStatus, device.FullName, device.Name, device.Group, device.Model)

			// try to send last backup metrics as unix timestamp
			if device.Last.Start != "" {
				unixTime, err := ConvertOixidzedTimeToUnix(device.Last.Start)
				if err != nil {
					slog.Error("Error parsing time", "error", err)
				} else {
					ch <- prometheus.MustNewConstMetric(c.deviceLastBackupStartMetric, prometheus.GaugeValue, float64(unixTime), device.FullName, device.Name, device.Group, device.Model)
				}
			} else {
				slog.Debug("Device has no last backup start time", "device", device.FullName)
			}
			if device.Last.End != "" {
				unixTime, err := ConvertOixidzedTimeToUnix(device.Last.End)
				if err != nil {
					slog.Error("Error parsing time", "error", err)
				} else {
					ch <- prometheus.MustNewConstMetric(c.deviceLastBackupEndMetric, prometheus.GaugeValue, float64(unixTime), device.FullName, device.Name, device.Group, device.Model)
				}
			} else {
				slog.Debug("Device has no last backup end time", "device", device.FullName)
			}

			if device.Last.Time != 0 {
				ch <- prometheus.MustNewConstMetric(c.deviceLastBackupTimeMetric, prometheus.GaugeValue, float64(device.Last.Time), device.FullName, device.Name, device.Group, device.Model)
			} else {
				slog.Debug("Device has no last backup time", "device", device.FullName)
			}

			if device.Last.Status != "" {
				var deviceLastBackupStatus float64
				switch device.Last.Status {
				case DeviceStatusSuccess:
					deviceLastBackupStatus = MetricSuccess
				case DeviceStatusNever:
					deviceLastBackupStatus = MetricNever
				case DeviceStatusNoConnection:
					deviceLastBackupStatus = MetricError
				default:
					slog.Error("Unknown device last backup status", "status", device.Last.Status, "device", device.FullName)
					deviceLastBackupStatus = MetricError
				}
				ch <- prometheus.MustNewConstMetric(c.deviceLastBackupStatusMetric, prometheus.GaugeValue, deviceLastBackupStatus, device.FullName, device.Name, device.Group, device.Model)
			}

			configStat, err := c.oxidizedClient.GetConfigStats(device.Group, device.Name, onlyDefaultGroup)
			if err != nil {
				slog.Error("Could not get config stats", "error", err, "device", device.FullName)
			} else {
				ch <- prometheus.MustNewConstMetric(c.deviceConfigSizeMetric, prometheus.GaugeValue, float64(configStat.Size), device.FullName, device.Name, device.Group, device.Model)
				ch <- prometheus.MustNewConstMetric(c.deviceConfigLinesMetric, prometheus.GaugeValue, float64(configStat.Lines), device.FullName, device.Name, device.Group, device.Model)
			}
		}(device)
	}
	wg.Wait()

	elapsed := time.Since(start_time)
	ch <- prometheus.MustNewConstMetric(c.oxidizedExporterCollectDurationMetric, prometheus.GaugeValue, float64(elapsed.Milliseconds()))
	slog.Info("Finished collecting oxidized metrics", "duration", elapsed)
}
