package app

import (
	"encoding/json"
	"time"

	"shione/host/internal/easytier"
	"shione/host/internal/frpembed"
)

type Snapshot struct {
	FRP      frpembed.Status `json:"frp"`
	EasyTier easytier.Status `json:"easytier"`
}

type App struct {
	frp      *frpembed.Manager
	easytier easytier.Adapter
}

func NewDefault() *App {
	return &App{
		frp:      frpembed.NewManager(frpembed.NewLocalBuilder()),
		easytier: easytier.NewDefaultAdapter(),
	}
}

func (a *App) ValidateFRP(configPath string) error {
	return a.frp.Validate(configPath)
}

func (a *App) StartFRP(configPath string) error {
	return a.frp.Start(configPath)
}

func (a *App) StopFRP(graceful time.Duration, wait time.Duration) error {
	return a.frp.Stop(graceful, wait)
}

func (a *App) StartEasyTier(configPath string) error {
	return a.easytier.Start(configPath)
}

func (a *App) StopEasyTier() error {
	return a.easytier.Stop()
}

func (a *App) Snapshot() Snapshot {
	return Snapshot{
		FRP:      a.frp.Status(),
		EasyTier: a.easytier.Status(),
	}
}

func (a *App) SnapshotJSON() ([]byte, error) {
	return json.MarshalIndent(a.Snapshot(), "", "  ")
}
