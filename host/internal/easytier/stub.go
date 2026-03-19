package easytier

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type Status struct {
	Supported  bool            `json:"supported"`
	Loaded     bool            `json:"loaded"`
	State      string          `json:"state"`
	DLLPath    string          `json:"dllPath,omitempty"`
	ConfigPath string          `json:"configPath,omitempty"`
	StartedAt  time.Time       `json:"startedAt,omitempty"`
	LastError  string          `json:"lastError,omitempty"`
	Infos      json.RawMessage `json:"infos,omitempty"`
}

type Adapter interface {
	Start(configPath string) error
	Stop() error
	Status() Status
}

type StubAdapter struct {
	mu     sync.RWMutex
	status Status
}

func NewStubAdapter() *StubAdapter {
	return &StubAdapter{
		status: Status{
			Supported: false,
			Loaded:    false,
			State:     "stub",
		},
	}
}

func (s *StubAdapter) Start(configPath string) error {
	s.mu.RLock()
	lastError := s.status.LastError
	s.mu.RUnlock()

	if lastError == "" {
		lastError = "EasyTier adapter is unavailable"
	}

	err := errors.New(lastError)
	s.setLastError(err)
	return err
}

func (s *StubAdapter) Stop() error {
	return nil
}

func (s *StubAdapter) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *StubAdapter) setLastError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.status.LastError = err.Error()
	}
}
