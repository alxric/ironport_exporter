package collector

import (
	"bytes"
	"encoding/xml"
	"math"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"golang.org/x/net/html/charset"
)

const (
	scrapeSubsystem = ""
)

//Metric describes a ironport metric
type Metric struct {
	Sources []Source `yaml:"sources"`
}

// Source describes a metric path
type Source struct {
	Path    string               `yaml:"path"`
	Objects map[string][]*Object `yaml:"objects"`
}

// Object describes a specific xml object
type Object struct {
	Name     string            `yaml:"name"`
	Desc     string            `yaml:"desc"`
	Property string            `yaml:"property"`
	Labels   map[string]string `yaml:"labels"`
	LabelMap map[string]string `yaml:"label_map"`
}

type metricCollector struct {
}

func init() {
	registerCollector("ironport_api_collector", defaultEnabled, MetricCollector)
}

//MetricCollector returns a new collector
func MetricCollector() (Collector, error) {
	return &metricCollector{}, nil
}

// Update implements Collector
func (c *metricCollector) Update(ch chan<- prometheus.Metric, target *target) error {
	b, err := APICall(target)
	if err != nil || b == nil {
		ch <- prometheus.MustNewConstMetric(ironportUp, prometheus.GaugeValue, 0, "ironport_up")
		return err
	}
	ch <- prometheus.MustNewConstMetric(ironportUp, prometheus.GaugeValue, 1, "ironport_up")
	err = ParseXML(ch, b)
	if err != nil {
		return err
	}
	return nil
}

// ParseXML will parse the result from the ironport endpoint and present it in prometheus readable format
func ParseXML(ch chan<- prometheus.Metric, b []byte) error {
	r := &Status{}
	decoder := xml.NewDecoder(bytes.NewReader(b))
	decoder.CharsetReader = charset.NewReaderLabel
	err := decoder.Decode(r)
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	for _, counter := range r.Counters.Counters {
		wg.Add(1)
		go func(counter Counter) {
			defer wg.Done()
			var fval float64
			var err error
			fval, err = strconv.ParseFloat(counter.Lifetime, 64)
			if err != nil {
				var m float64
				q := counter.Lifetime[len(counter.Lifetime)-1:]
				switch q {
				case "K":
					m = 1024
				case "M":
					m = math.Pow(1024, 2)
				case "G":
					m = math.Pow(1024, 3)
				case "T":
					m = math.Pow(1024, 4)
				}
				fval, err = strconv.ParseFloat(counter.Lifetime[0:len(counter.Lifetime)-1], 64)
				fval *= m
				if err != nil {
					log.Error(err)
					return
				}
			}
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, scrapeSubsystem, counter.Name),
					counter.Name, nil, nil,
				), prometheus.CounterValue, fval)
		}(counter)
	}
	for _, gauge := range r.Gauges.Gauges {
		wg.Add(1)
		go func(gauge Gauge) {
			defer wg.Done()
			var fval float64
			var err error
			fval, err = strconv.ParseFloat(gauge.Current, 64)
			if err != nil {
				var m float64
				q := gauge.Current[len(gauge.Current)-1:]
				switch q {
				case "K":
					m = 1024
				case "M":
					m = math.Pow(1024, 2)
				case "G":
					m = math.Pow(1024, 3)
				case "T":
					m = math.Pow(1024, 4)
				}
				fval, err = strconv.ParseFloat(gauge.Current[0:len(gauge.Current)-1], 64)
				fval *= m
				if err != nil {
					log.Error(err)
					return
				}
			}
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, scrapeSubsystem, gauge.Name),
					gauge.Name, nil, nil,
				), prometheus.GaugeValue, fval)
		}(gauge)
	}
	wg.Wait()
	return nil
}
