package frpembed

import (
	"context"
	"testing"
	"time"
)

type fakeBuilder struct {
	validateErr error
	buildErr    error
	svc         *fakeService
}

func (f *fakeBuilder) Validate(configPath string) error {
	return f.validateErr
}

func (f *fakeBuilder) Build(configPath string) (RunnableService, string, error) {
	if f.buildErr != nil {
		return nil, "", f.buildErr
	}
	return f.svc, configPath, nil
}

type fakeService struct {
	graceful []time.Duration
}

func (f *fakeService) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f *fakeService) GracefulClose(d time.Duration) {
	f.graceful = append(f.graceful, d)
}

func TestManagerStartStop(t *testing.T) {
	service := &fakeService{}
	manager := NewManager(&fakeBuilder{svc: service})

	if err := manager.Start("client.toml"); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	status := manager.Status()
	if !status.Running || status.State != StateRunning {
		t.Fatalf("expected running status, got %+v", status)
	}

	if err := manager.Stop(250*time.Millisecond, time.Second); err != nil {
		t.Fatalf("stop failed: %v", err)
	}

	status = manager.Status()
	if status.Running || status.State != StateIdle {
		t.Fatalf("expected idle status after stop, got %+v", status)
	}
	if len(service.graceful) != 1 || service.graceful[0] != 250*time.Millisecond {
		t.Fatalf("expected graceful stop duration to be recorded, got %#v", service.graceful)
	}
}

func TestManagerRejectsDoubleStart(t *testing.T) {
	service := &fakeService{}
	manager := NewManager(&fakeBuilder{svc: service})

	if err := manager.Start("client.toml"); err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	if err := manager.Start("client.toml"); err == nil {
		t.Fatal("expected second start to fail")
	}

	_ = manager.Stop(0, time.Second)
}

func TestManagerValidateDelegatesToBuilder(t *testing.T) {
	expectedErr := context.Canceled
	manager := NewManager(&fakeBuilder{validateErr: expectedErr})

	if err := manager.Validate("client.toml"); err != expectedErr {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
