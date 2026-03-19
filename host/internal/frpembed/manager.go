package frpembed

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

type State string

const (
	StateIdle     State = "idle"
	StateRunning  State = "running"
	StateStopping State = "stopping"
)

type Status struct {
	State      State     `json:"state"`
	Running    bool      `json:"running"`
	ConfigPath string    `json:"configPath,omitempty"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	StoppedAt  time.Time `json:"stoppedAt,omitempty"`
	LastError  string    `json:"lastError,omitempty"`
	LastExit   string    `json:"lastExit,omitempty"`
}

type Manager struct {
	builder Builder

	mu     sync.RWMutex
	status Status
	svc    RunnableService
	cancel context.CancelFunc
	done   chan struct{}
}

func NewManager(builder Builder) *Manager {
	if builder == nil {
		builder = NewLocalBuilder()
	}
	return &Manager{
		builder: builder,
		status: Status{
			State: StateIdle,
		},
	}
}

func (m *Manager) Validate(configPath string) error {
	return m.builder.Validate(configPath)
}

func (m *Manager) Start(configPath string) error {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("resolve frp config path: %w", err)
	}

	m.mu.Lock()
	if m.status.Running || m.status.State == StateStopping {
		m.mu.Unlock()
		return fmt.Errorf("frp client is already running")
	}
	m.mu.Unlock()

	svc, builtConfigPath, err := m.builder.Build(absConfigPath)
	if err != nil {
		m.mu.Lock()
		m.status.LastError = err.Error()
		m.status.LastExit = err.Error()
		m.mu.Unlock()
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	m.mu.Lock()
	m.svc = svc
	m.cancel = cancel
	m.done = done
	m.status = Status{
		State:      StateRunning,
		Running:    true,
		ConfigPath: builtConfigPath,
		StartedAt:  time.Now(),
	}
	m.mu.Unlock()

	go m.runService(runCtx, svc, done)
	return nil
}

func (m *Manager) Stop(graceful time.Duration, wait time.Duration) error {
	m.mu.Lock()
	if !m.status.Running || m.svc == nil {
		m.mu.Unlock()
		return nil
	}

	svc := m.svc
	cancel := m.cancel
	done := m.done
	m.status.State = StateStopping
	m.mu.Unlock()

	svc.GracefulClose(graceful)
	if cancel != nil {
		cancel()
	}

	if wait <= 0 {
		return nil
	}

	select {
	case <-done:
		return nil
	case <-time.After(wait):
		return fmt.Errorf("timed out waiting for frp client to stop")
	}
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) runService(runCtx context.Context, svc RunnableService, done chan struct{}) {
	err := svc.Run(runCtx)

	m.mu.Lock()
	if m.svc == svc {
		m.svc = nil
		m.cancel = nil
		m.done = nil
		m.status.Running = false
		m.status.State = StateIdle
		m.status.StoppedAt = time.Now()
		if err != nil && !errors.Is(err, context.Canceled) {
			m.status.LastError = err.Error()
			m.status.LastExit = err.Error()
		} else {
			m.status.LastExit = ""
		}
	}
	m.mu.Unlock()

	close(done)
}
