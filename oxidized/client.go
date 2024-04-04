package oxidized

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	DeviceStatusSuccess      = "success"
	DeviceStatusNever        = "never"
	DeviceStatusNoConnection = "no_connection"
)

type Device struct {
	// FullName is the full name of the device, e.g. "netzwerk/leaf-netzwerk-01"
	FullName string `json:"full_name"`
	// Name is the name of the device, e.g. "leaf-netzwerk-01"
	Name string `json:"name"`
	// Group is the group of the device, e.g. "netzwerk"
	Group string `json:"group"`
	// Ip is the FQDN or IP of the device
	Ip string `json:"ip"`
	// Model is the model of the device, e.g. "cisco_ios"
	Model string `json:"model"`
	// Status is either "success", "never" or "no_connection"
	Status string `json:"status"`

	// Last is the last backup of the device
	Last struct {
		// Start is the start time of the last backup
		Start string `json:"start"`
		// End is the end time of the last backup
		End string `json:"end"`
		// Status is the status of the last backup
		Status string `json:"status"`
		// Time is the time of the last backup in seconds
		Time float32 `json:"time"`
	}
}

type DeviceStat struct {
	Name string `json:"name"`
}

type ConfigStat struct {
	// Size of the config in bytes
	Size int
	// Number of lines in the config
	Lines int
}

type OxidizedClient struct {
	Url      string
	Username string
	Password string
}

func NewOxidizedClient(url, username, password string) *OxidizedClient {
	return &OxidizedClient{
		Url:      url,
		Username: username,
		Password: password,
	}
}

// get makes a GET request to the given path and decodes the response into v.
func (c *OxidizedClient) get(path string, v interface{}) error {
	req, err := http.NewRequest("GET", c.Url+"/"+path+"?format=json", nil)
	if err != nil {
		return err
	}

	resp, err := c.request(*req)
	if err != nil {
		return err
	}

	slog.Debug("Got response from oxidized", "status_code", resp.StatusCode)

	defer resp.Body.Close()
	// debug json response
	b, _ := io.ReadAll(resp.Body)
	slog.Debug("Read response content", "body", string(b))

	// decode json response from b
	return json.Unmarshal(b, v)
}

func (c *OxidizedClient) request(req http.Request) (*http.Response, error) {
	slog.Debug("Sending request to oxidized", "url", req.URL.String(), "method", req.Method)
	if c.Username != "" && c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	return http.DefaultClient.Do(&req)
}

// GetDevices returns a list of devices from the oxidized API.
func (c *OxidizedClient) GetDevices() ([]Device, error) {
	var devices []Device
	err := c.get("nodes", &devices)
	if err != nil {
		return nil, err
	}

	slog.Debug("Got devices", "count", len(devices))
	return devices, nil
}

// GetDeviceStats returns the stats from the oxidized API.
func (c *OxidizedClient) GetStatus() ([]DeviceStat, error) {
	var stats []DeviceStat
	err := c.get("nodes/stats", &stats)
	if err != nil {
		return nil, err
	}

	slog.Debug("Got device stats", "count", len(stats))
	return stats, nil
}

func (c *OxidizedClient) GetConfigStats(group string, name string) (*ConfigStat, error) {
	req, err := http.NewRequest("GET", c.Url+"/"+"node/fetch/"+group+"/"+name, nil)
	if err != nil {
		return nil, err
	}

	slog.Debug("Getting config of device", "group", group, "name", name)
	resp, err := c.request(*req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch config for %s/%s: %s", group, name, resp.Status)
	}

	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	return &ConfigStat{
		Size:  len(b),
		Lines: bytes.Count(b, []byte("\n")),
	}, nil
}

// ConvertOixidzedTimeTo8601 converts from 2019-11-19 14:00:00 CET
// to UnixTimeStamp
func ConvertOixidzedTimeToUnix(t string) (int64, error) {
	parsed, err := time.Parse("2006-01-02 15:04:05 MST", t)
	if err != nil {
		return 0, err
	}
	unix := parsed.Unix()
	return unix, nil
}
