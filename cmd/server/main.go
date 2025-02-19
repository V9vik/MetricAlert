package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var storage = NewMemStorage()

type MemStorageImpl interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) float64
	GetCounter(name string) int
}
type MemStorage struct {
	mu      sync.Mutex
	gauge   map[string]float64
	counter map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauge:   make(map[string]float64),
		counter: make(map[string]int64),
	}
}

func (s *MemStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauge[name] = value
}
func (s *MemStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter[name] += value
}
func (s *MemStorage) GetGauge(name string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gauge[name]
}
func (s *MemStorage) GetCounter(name string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counter[name]
}
func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	pathParts := strings.Split(r.URL.Path, "/")

	if len(pathParts) != 4 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	metricType := pathParts[2]
	metricName := pathParts[3]
	metricValue := pathParts[4]
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
		return

	case "counter":
		value, err := strconv.Atoi(metricValue)
		if err != nil {
			http.Error(w, "Invalid value", http.StatusBadRequest)
		}
		storage.UpdateCounter(metricName, int64(value))
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, "Bad metric", http.StatusBadRequest)
		return
	}
}
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", updateHandler)
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
