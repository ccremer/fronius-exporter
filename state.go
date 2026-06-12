package main

import (
	"sync"
	"time"

	"github.com/ccremer/fronius-exporter/pkg/fronius"
)

type exporterState struct {
	mu sync.RWMutex

	powerFlowData      *fronius.SymoData
	powerFlowUpdatedAt time.Time
	archiveData        map[string]fronius.InverterArchive
	archiveUpdatedAt   time.Time
	inverterRealtime   *fronius.SymoInverterRealtimeData
	inverterUpdatedAt  time.Time
	meterRealtime      *fronius.SymoMeterRealtimeData
	meterUpdatedAt     time.Time
	lastPollingError   time.Time
}

func newExporterState() *exporterState {
	return &exporterState{}
}

func (s *exporterState) SetPowerFlow(data *fronius.SymoData, updatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.powerFlowData = data
	s.powerFlowUpdatedAt = updatedAt
}

func (s *exporterState) SetArchive(data map[string]fronius.InverterArchive, updatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.archiveData = data
	s.archiveUpdatedAt = updatedAt
}

func (s *exporterState) SetInverterRealtime(data *fronius.SymoInverterRealtimeData, updatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inverterRealtime = data
	s.inverterUpdatedAt = updatedAt
}

func (s *exporterState) SetMeterRealtime(data *fronius.SymoMeterRealtimeData, updatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meterRealtime = data
	s.meterUpdatedAt = updatedAt
}

func (s *exporterState) MarkPollingError(at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPollingError = at
}

func (s *exporterState) IsFresh(options fronius.ClientOptions, maxAge time.Duration) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	if options.PowerFlowEnabled && isStale(s.powerFlowUpdatedAt, now, maxAge) {
		return false
	}
	if options.ArchiveEnabled && isStale(s.archiveUpdatedAt, now, maxAge) {
		return false
	}
	if options.InverterRealtimeEnabled && isStale(s.inverterUpdatedAt, now, maxAge) {
		return false
	}
	if options.MeterRealtimeEnabled && isStale(s.meterUpdatedAt, now, maxAge) {
		return false
	}
	return true
}

func isStale(updatedAt, now time.Time, maxAge time.Duration) bool {
	return updatedAt.IsZero() || now.Sub(updatedAt) > maxAge
}
