package anypoint

import (
	"encoding/json"
	"time"
)

// Monitoring Archive API metrics data definitions
type APIMonitoringMetric struct {
	Time   time.Time
	Events []APISummaryMetricEvent
}

type DataFile struct {
	ID   string    `json:"id"`
	Time time.Time `json:"time"`
	Size int       `json:"size"`
}

type DataFileResources struct {
	Resources []DataFile `json:"resources"`
}

type APISummaryMetricEvent struct {
	APIName            string `json:"api_name"`
	APIVersion         string `json:"api_version"`
	APIVersionID       string `json:"api_version_id"`
	ClientID           string `json:"client_id"`
	Method             string `json:"method"`
	StatusCode         string `json:"status_code"`
	ResponseSizeCount  int    `json:"response_size.count"`
	ResponseSizeMax    int    `json:"response_size.max"`
	ResponseSizeMin    int    `json:"response_size.min"`
	ResponseSizeSos    int    `json:"response_size.sos"`
	ResponseSizeSum    int    `json:"response_size.sum"`
	ResponseTimeCount  int    `json:"response_time.count"`
	ResponseTimeMax    int    `json:"response_time.max"`
	ResponseTimeMin    int    `json:"response_time.min"`
	ResponseTimeSos    int    `json:"response_time.sos"`
	ResponseTimeSum    int    `json:"response_time.sum"`
	RequestSizeCount   int    `json:"request_size.count"`
	RequestSizeMax     int    `json:"request_size.max"`
	RequestSizeMin     int    `json:"request_size.min"`
	RequestSizeSos     int    `json:"request_size.sos"`
	RequestSizeSum     int    `json:"request_size.sum"`
	RequestDisposition string `json:"request_disposition"`
}

type MetricData struct {
	Format   string                 `json:"format"`
	Time     int64                  `json:"time"`
	Type     string                 `json:"type"`
	Metadata map[string]interface{} `json:"metadata"`
	Commons  map[string]interface{} `json:"commons"`
	Events   []APISummaryMetricEvent
}

// Influx DB based metric data definitions
type MonitoringBootInfo struct {
	Settings MonitoringBootSetting `json:"Settings"`
}

type MonitoringBootSetting struct {
	DataSource MonitoringDataSource `json:"datasources"`
}

type MonitoringDataSource struct {
	InfluxDB InfluxDB `json:"influxdb"`
}

type InfluxDB struct {
	ID       int    `json:"id"`
	Database string `json:"database"`
}

type MetricResponse struct {
	Results []*MetricResult `json:"results"`
}

type MetricResult struct {
	StatementID int             `json:"statement_id"`
	Series      []*MetricSeries `json:"series"`
}

type MetricTag struct {
	ClientID   string `json:"client_id"`
	StatusCode string `json:"status_code"`
}

type MetricSeries struct {
	Name        string      `json:"name"`
	Tags        *MetricTag  `json:"tags"`
	Columns     []string    `json:"columns"`
	Values      [][]float64 `json:"values"`
	Time        time.Time   `json:"-"`
	Count       int64       `json:"-"`
	ResponseMax int64       `json:"-"`
	ResponseMin int64       `json:"-"`
}

func (ms *MetricSeries) UnmarshalJSON(data []byte) error {
	type alias MetricSeries
	v := &struct{ *alias }{
		alias: (*alias)(ms),
	}

	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	tm := ms.getValue("time")
	ms.Time = time.UnixMilli(tm)
	ms.Count = ms.getValue("request_count")
	ms.ResponseMax = ms.getValue("response_max")
	ms.ResponseMin = ms.getValue("response_min")

	return nil
}

func (ms *MetricSeries) getValue(columnName string) int64 {
	return ms.getMetricSeriesIndexValue(ms.getMetricSeriesColumnIndex(columnName))
}

func (ms *MetricSeries) getMetricSeriesIndexValue(index int) int64 {
	if len(ms.Values) > 0 {
		val := ms.Values[0]
		if index >= 0 && index < len(val) {
			return int64(val[index])
		}
	}

	return 0
}

func (ms *MetricSeries) getMetricSeriesColumnIndex(column string) int {
	for n := 0; n < len(ms.Columns); n++ {
		if ms.Columns[n] == column {
			return n
		}
	}
	return -1
}
