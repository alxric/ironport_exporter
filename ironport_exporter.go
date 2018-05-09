package main

import (
	"fmt"
	"io/ioutil"
	"itops/ironport_exporter/collector"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	yaml "gopkg.in/yaml.v2"
)

const (
	//VERSION of the exporter
	VERSION = "0.1.0"
)

var (
	listenAddress = kingpin.Flag(
		"web.listen-address",
		"Address to listen on for web interface and telemetry.",
	).Default(":9113").String()
	metricsPath = kingpin.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics.",
	).Default("/metrics").String()

	configPath = kingpin.Flag(
		"config.file",
		"Path to the config file.",
	).Default("config.yml").String()
)

func main() {
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("ironport_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	log.Infoln("Starting Ironport exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Ironport Exporter</title></head>
			<body>
			<h1> Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	http.HandleFunc(
		*metricsPath,
		prometheus.InstrumentHandlerFunc("metrics", handler),
	)
	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query()["target"]
	config := r.URL.Query()["config"]
	if len(config) > 0 {
		configPath = &config[0]
	}
	log.Debugln("collect query:", target)
	cfg, err := readCfg()
	if err != nil {
		log.Fatalf("Could not read config file: %v\n", err)
	}
	if len(target) == 0 {
		log.Warnln("No target provided", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No target provided!"))
		return
	}

	nc, err := collector.New(target[0], cfg.Username, cfg.Password)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}

	h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})

	h.ServeHTTP(w, r)
}

type config struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func readCfg() (*config, error) {
	b, err := ioutil.ReadFile(*configPath)
	if err != nil {
		return nil, err
	}
	c := &config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}
	return c, nil

}

func filter(filters map[string]bool, name string, flag bool) bool {
	if len(filters) > 0 {
		return flag && filters[name]
	}
	return flag
}

func init() {
	prometheus.MustRegister(version.NewCollector("ironport_exporter"))
}
