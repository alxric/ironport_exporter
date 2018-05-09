package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	exporter        = "exporter"
	namespace       = "ironport"
	defaultEnabled  = true
	defaultDisabled = false
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"ironport_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"ironport_exporter: Whether a collector succeeded.",
		[]string{"collector"},
		nil,
	)
	ironportUp = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"ironport_exporter: Whether the ironport instance is reachable or not.",
		[]string{"collector"},
		nil,
	)
	factories      = make(map[string]func() (Collector, error))
	collectorState = make(map[string]*bool)
)

// Exporter collects ironport metrics. It implements prometheus.Collector.
type Exporter struct {
	Collectors map[string]Collector
	Target     *target
}

// Collector is the interface a collector has to implement.
type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Update(ch chan<- prometheus.Metric, target *target) error
}

func registerCollector(collector string, isDefaultEnabled bool, factory func() (Collector, error)) {
	var helpDefaultState string
	if isDefaultEnabled {
		helpDefaultState = "enabled"
	} else {
		helpDefaultState = "disabled"
	}

	flagName := fmt.Sprintf("collector.%s", collector)
	flagHelp := fmt.Sprintf("Enable the %s collector (default: %s).", collector, helpDefaultState)
	defaultValue := fmt.Sprintf("%v", isDefaultEnabled)

	flag := kingpin.Flag(flagName, flagHelp).Default(defaultValue).Bool()
	collectorState[collector] = flag

	factories[collector] = factory
}

type target struct {
	Hostname string
	Username string
	Password string
}

//New returns a new exporter for the provided ASE
func New(hostname string, username string, password string) (*Exporter, error) {
	f := make(map[string]bool)
	collectors := make(map[string]Collector)
	for key, enabled := range collectorState {
		if *enabled {
			collector, err := factories[key]()
			if err != nil {
				return nil, err
			}
			if len(f) == 0 || f[key] {
				collectors[key] = collector
			}
		}
	}
	target := &target{
		Hostname: hostname,
		Username: username,
		Password: password,
	}
	return &Exporter{Collectors: collectors, Target: target}, nil
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {

	wg := sync.WaitGroup{}
	wg.Add(len(e.Collectors))
	for name, c := range e.Collectors {
		go func(name string, c Collector) {
			e.execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func (e *Exporter) execute(name string, c Collector, ch chan<- prometheus.Metric) {
	log.Info("executing ", name)
	begin := time.Now()
	var err error
	err = c.Update(ch, e.Target)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}
