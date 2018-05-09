package main

import (
	"io/ioutil"
	"itops/ironport_exporter/collector"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	yaml "gopkg.in/yaml.v2"
)

func TestMetricGen(t *testing.T) {
	tt := readMetrics(t)
	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			ch := make(chan prometheus.Metric)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()

				resp := <-ch
				m := &dto.Metric{}
				resp.Write(m)
				switch tc.Type {
				case "gauge":
					if m.GetGauge().GetValue() != tc.ExpectedOutput {
						t.Errorf("%s - Expected: %v, Got: %v", tc.Name, tc.ExpectedOutput, m.GetGauge().GetValue())
					}
				case "counter":
					if m.GetCounter().GetValue() != tc.ExpectedOutput {
						t.Errorf("%s - Expected: %v, Got: %v", tc.Name, tc.ExpectedOutput, m.GetCounter().GetValue())
					}
				default:
					t.Errorf("%s - Unknown metric type", tc.Name)
				}
			}()
			if err := collector.ParseXML(ch, []byte(tc.XML)); err != nil {
				t.Errorf("Could not Parse xml: %v", err)
			}
			wg.Wait()
		})
	}
}

type Metric struct {
	Name           string  `yaml:"name"`
	XML            string  `yaml:"xml"`
	Type           string  `yaml:"type"`
	ExpectedError  error   `yaml:"expected_error"`
	ExpectedOutput float64 `yaml:"expected_output"`
}

func readMetrics(t *testing.T) []Metric {
	b, err := ioutil.ReadFile("metric_cases.yml")
	if err != nil {
		t.Fatal("Could not open metric_cases.yml")
	}
	metrics := []Metric{}
	err = yaml.Unmarshal(b, &metrics)
	if err != nil {
		t.Fatal("Could not parse metric_cases.yml")
	}
	return metrics
}
