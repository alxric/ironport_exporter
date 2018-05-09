package collector

// Status is the xml response we get from the Ironport api
type Status struct {
	Build    string `xml:"build,attr"`
	Counters struct {
		Counters []Counter `xml:"counter"`
	} `xml:"counters"`
	Gauges struct {
		Gauges []Gauge `xml:"gauge"`
	} `xml:"gauges"`
}

// Counter is an ironport counter metric
type Counter struct {
	Name     string `xml:"name,attr"`
	Reset    string `xml:"reset,attr"`
	Uptime   string `xml:"uptime,attr"`
	Lifetime string `xml:"lifetime,attr"`
}

// Gauge is an ironport gauge metric
type Gauge struct {
	Name    string `xml:"name,attr"`
	Current string `xml:"current,attr"`
}
