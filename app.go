package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"entropy/internal/node"
)

type App struct {
	mu      sync.RWMutex
	service *node.Service
	start   error
	ctx     context.Context
	cancel  context.CancelFunc
}

type ActionResult struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	nodeContext, cancel := context.WithCancel(ctx)
	service, err := node.New(node.Config{})
	if err == nil {
		err = service.Start(nodeContext)
	}
	a.mu.Lock()
	a.service = service
	a.start = err
	a.ctx = nodeContext
	a.cancel = cancel
	a.mu.Unlock()
}

func (a *App) shutdown(context.Context) {
	a.mu.RLock()
	service := a.service
	cancelNode := a.cancel
	a.mu.RUnlock()
	if cancelNode != nil {
		cancelNode()
	}
	if service == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.Close(ctx)
}

func (a *App) GetDashboard() (node.Dashboard, error) {
	service, err := a.readyService()
	if err != nil {
		return node.Dashboard{}, err
	}
	return service.Dashboard()
}

func (a *App) SendTransaction(to, amount, fee string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	tx, err := service.Send(to, amount, fee)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: tx.ID, Message: "Transaction added to local pending pool"}, nil
}

func (a *App) MineOneBlock() (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return ActionResult{}, fmt.Errorf("node is still starting")
	}
	block, err := service.MineOnce(ctx)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: block.Hash, Message: fmt.Sprintf("Block %d mined", block.Height)}, nil
}

func (a *App) SetMining(enabled bool) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if enabled {
		if err := service.StartMining(); err != nil {
			return ActionResult{}, err
		}
		return ActionResult{Message: "Mining started"}, nil
	}
	service.StopMining()
	return ActionResult{Message: "Mining stopping"}, nil
}

func (a *App) AddPeer(peer string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if err := service.AddPeer(peer); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: "Peer added"}, nil
}

func (a *App) readyService() (*node.Service, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.start != nil {
		return nil, a.start
	}
	if a.service == nil {
		return nil, fmt.Errorf("node is still starting")
	}
	return a.service, nil
}
