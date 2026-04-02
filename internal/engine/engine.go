package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/maxzhang666/ops-tunnel/internal/config"
	tunnelssh "github.com/maxzhang666/ops-tunnel/internal/ssh"
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
	cfg      *config.Config
	bus      EventBus
	hostKeys tunnelssh.HostKeyStore
	mu       sync.RWMutex
	sups     map[string]*tunnelSupervisor
}

func NewEngine(cfg *config.Config, bus EventBus, hostKeys tunnelssh.HostKeyStore) Engine {
	return &eng{
		cfg:      cfg,
		bus:      bus,
		hostKeys: hostKeys,
		sups:     make(map[string]*tunnelSupervisor),
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

func (e *eng) resolveChain(t *config.Tunnel) ([]config.SSHConnection, error) {
	conns := make([]config.SSHConnection, 0, len(t.Chain))
	for _, id := range t.Chain {
		found := false
		for _, c := range e.cfg.SSHConnections {
			if c.ID == id {
				conns = append(conns, c)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("SSH connection '%s' not found", id)
		}
	}
	return conns, nil
}

func (e *eng) getOrCreateSupervisor(t *config.Tunnel) (*tunnelSupervisor, error) {
	if sup, ok := e.sups[t.ID]; ok {
		return sup, nil
	}
	conns, err := e.resolveChain(t)
	if err != nil {
		return nil, err
	}
	sup := newSupervisor(*t, conns, e.bus, e.hostKeys)
	e.sups[t.ID] = sup
	return sup, nil
}

func (e *eng) StartTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}
	sup, err := e.getOrCreateSupervisor(t)
	if err != nil {
		return err
	}
	return sup.Start(ctx)
}

func (e *eng) StopTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}
	sup, err := e.getOrCreateSupervisor(t)
	if err != nil {
		return err
	}
	return sup.Stop(ctx)
}

func (e *eng) RestartTunnel(ctx context.Context, id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	t, err := e.findTunnel(id)
	if err != nil {
		return err
	}
	sup, err := e.getOrCreateSupervisor(t)
	if err != nil {
		return err
	}
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
