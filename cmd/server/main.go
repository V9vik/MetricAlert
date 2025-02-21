package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type MemStorage struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MemStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

var (
	storage = NewMemStorage()
)

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 4 || pathParts[0] != "update" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	metricType := pathParts[1]
	metricName := pathParts[2]
	metricValue := pathParts[3]

	if metricName == "" {
		http.Error(w, "Metric name required", http.StatusNotFound)
		return
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid value", http.StatusBadRequest)
			return
		}
		storage.UpdateGauge(metricName, value)
		w.WriteHeader(http.StatusOK)

	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid value", http.StatusBadRequest)
			return
		}
		storage.UpdateCounter(metricName, value)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Bad metric type", http.StatusBadRequest)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", updateHandler)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
