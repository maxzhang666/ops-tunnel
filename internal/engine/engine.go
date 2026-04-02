package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

type Engine interface {
	StartTunnel(ctx context.Context, id string) error
	StopTunnel(ctx context.Context, id string) error
	RestartTunnel(ctx context.Context, id string) error
	GetStatus(id string) (TunnelStatus, bool)
	ListStatus() []TunnelStatus
	Events() EventBus
	Shutdown(ctx context.Context) error
}

type eng struct {
	cfg  *config.Config
	bus  EventBus
	mu   sync.RWMutex
	sups map[string]*tunnelSupervisor
}

func NewEngine(cfg *config.Config, bus EventBus) Engine {
	return &eng{
		cfg:  cfg,
		bus:  bus,
		sups: make(map[string]*tunnelSupervisor),
	}
}

func (e *eng) findTunnel(id string) (*config.Tunnel, error) {
	for i := range e.cfg.Tunnels {
		if e.cfg.Tunnels[i].ID == id {
			return &e.cfg.Tunnels[i], nil
		}
	}
	return nil, fmt.Errorf("tunnel '%s' not found", id)
}

func (e *eng) getOrCreateSupervisor(t *config.Tunnel) *tunnelSupervisor {
	if sup, ok := e.sups[t.ID]; ok {
		return sup
	}
	sup := newSupervisor(*t, e.bus)
	e.sups[t.ID] = sup
	return sup
}

func (e *eng) StartTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}
	return e.getOrCreateSupervisor(t).Start(ctx)
}

func (e *eng) StopTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}
	return e.getOrCreateSupervisor(t).Stop(ctx)
}

func (e *eng) RestartTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}
	sup := e.getOrCreateSupervisor(t)
	if err := sup.Stop(ctx); err != nil {
		return err
	}
	return sup.Start(ctx)
}

func (e *eng) GetStatus(id string) (TunnelStatus, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if sup, ok := e.sups[id]; ok {
		return sup.Status(), true
	}
	for _, t := range e.cfg.Tunnels {
		if t.ID == id {
			return TunnelStatus{ID: id, State: StateStopped, Chain: []HopStatus{}, Mappings: []MappingStatus{}}, true
		}
	}
	return TunnelStatus{}, false
}

func (e *eng) ListStatus() []TunnelStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	statuses := make([]TunnelStatus, 0, len(e.cfg.Tunnels))
	for _, t := range e.cfg.Tunnels {
		if sup, ok := e.sups[t.ID]; ok {
			statuses = append(statuses, sup.Status())
		} else {
			statuses = append(statuses, TunnelStatus{ID: t.ID, State: StateStopped, Chain: []HopStatus{}, Mappings: []MappingStatus{}})
		}
	}
	return statuses
}

func (e *eng) Events() EventBus { return e.bus }

func (e *eng) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, sup := range e.sups {
		sup.Stop(ctx)
	}
	e.sups = make(map[string]*tunnelSupervisor)
	return nil
}
