package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type Metrics struct {
	mu       sync.RWMutex
	gauge    map[string]float64
	counters map[string]int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		gauge:    make(map[string]float64),
		counters: make(map[string]int64),
	}
}

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	url            = "http://localhost:8080"
)

func (m Metrics) UpdateRuntimeMetrics() {
	for {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		m.mu.Lock()
		m.gauge["Alloc"] = float64(memStats.Alloc)
		m.gauge["BuckHashSys"] = float64(memStats.BuckHashSys)
		m.gauge["Frees"] = float64(memStats.Frees)
		m.gauge["GCCPUFraction"] = memStats.GCCPUFraction
		m.gauge["GCSys"] = float64(memStats.GCSys)
		m.gauge["HeapAlloc"] = float64(memStats.HeapAlloc)
		m.gauge["HeapIdle"] = float64(memStats.HeapIdle)
		m.gauge["HeapInuse"] = float64(memStats.HeapInuse)
		m.gauge["HeapObjects"] = float64(memStats.HeapObjects)
		m.gauge["HeapReleased"] = float64(memStats.HeapReleased)
		m.gauge["HeapSys"] = float64(memStats.HeapSys)
		m.gauge["LastGC"] = float64(memStats.LastGC)
		m.gauge["Lookups"] = float64(memStats.Lookups)
		m.gauge["MCacheInuse"] = float64(memStats.MCacheInuse)
		m.gauge["MCacheSys"] = float64(memStats.MCacheSys)
		m.gauge["MSpanInuse"] = float64(memStats.MSpanInuse)
		m.gauge["MSpanSys"] = float64(memStats.MSpanSys)
		m.gauge["Mallocs"] = float64(memStats.Mallocs)
		m.gauge["NextGC"] = float64(memStats.NextGC)
		m.gauge["NumForcedGC"] = float64(memStats.NumForcedGC)
		m.gauge["NumGC"] = float64(memStats.NumGC)
		m.gauge["OtherSys"] = float64(memStats.OtherSys)
		m.gauge["PauseTotalNs"] = float64(memStats.PauseTotalNs)
		m.gauge["StackInuse"] = float64(memStats.StackInuse)
		m.gauge["StackSys"] = float64(memStats.StackSys)
		m.gauge["Sys"] = float64(memStats.Sys)
		m.gauge["TotalAlloc"] = float64(memStats.TotalAlloc)

		m.counters["PollCount"]++
		m.gauge["RandomValue"] = float64(rand.Intn(100))
		time.Sleep(pollInterval)
	}
}
func (m *Metrics) Snapshot() (map[string]float64, map[string]int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gauges := make(map[string]float64, len(m.gauge))
	for k, v := range m.gauge {
		gauges[k] = v
	}

	counters := make(map[string]int64, len(m.counters))
	for k, v := range m.counters {
		counters[k] = v
	}

	return gauges, counters
}

type Sender struct {
	client    *http.Client
	serverURL string
}

func NewSender(serverURL string) *Sender {
	return &Sender{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		serverURL: serverURL,
	}
}

func (s *Sender) SendMetric(metricType, name string, value float64) error {
	url := fmt.Sprintf("%s/update/%s/%s/%v", s.serverURL, metricType, name, value)
	resp, err := s.client.Post(url, "text/plain", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func main() {
	metrix := NewMetrics()
	Sen := NewSender(url)
	ticker := time.NewTicker(pollInterval)
	for range ticker.C {
		go metrix.UpdateRuntimeMetrics()
	}
	reportTicket := time.NewTicker(reportInterval)
	for range reportTicket.C {
		counters, gauges := metrix.Snapshot()
		var wg sync.WaitGroup
		for name, value := range gauges {
			wg.Add(1)
			go func(n string, v int64) {
				defer wg.Done()
				if err := Sen.SendMetric("gauge", n, float64(v)); err != nil {
					log.Printf("Error sending gauge %s: %v", n, err)
				}
			}(name, value)
		}
		for name, value := range counters {
			wg.Add(1)
			var wg sync.WaitGroup
			go func(n string, v int64) {
				defer wg.Done()
				if err := Sen.SendMetric("counter", n, float64(v)); err != nil {
					log.Printf("Error sending counter %s: %v", n, err)
				}
			}(name, int64(value))
		}
		wg.Wait()
		fmt.Println("Metrics sent")
	}

}
