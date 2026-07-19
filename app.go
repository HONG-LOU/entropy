package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"entropy/internal/core"
	"entropy/internal/ledger"
	"entropy/internal/node"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	mu      sync.RWMutex
	startMu sync.Mutex
	wait    sync.WaitGroup
	service *node.Service
	start   error
	ctx     context.Context
	cancel  context.CancelFunc
	closing bool
}

type ActionResult struct {
	ID      string `json:"id,omitempty"`
	Message string `json:"message"`
}

type StartupState struct {
	Ready   bool   `json:"ready"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) focusWindow() {
	a.mu.RLock()
	ctx := a.ctx
	closing := a.closing
	a.mu.RUnlock()
	if ctx == nil || closing {
		return
	}
	wailsruntime.WindowUnminimise(ctx)
	wailsruntime.WindowShow(ctx)
}

func (a *App) startup(ctx context.Context) {
	nodeContext, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.ctx = nodeContext
	a.cancel = cancel
	a.mu.Unlock()
	a.wait.Add(1)
	go func() {
		defer a.wait.Done()
		a.startNode(nodeContext)
	}()
}

func (a *App) startNode(ctx context.Context) {
	a.startMu.Lock()
	defer a.startMu.Unlock()
	a.mu.RLock()
	alreadyRunning := a.service != nil && a.start == nil
	closing := a.closing
	a.mu.RUnlock()
	if alreadyRunning || closing {
		return
	}
	service, err := node.NewContext(ctx, node.Config{
		FallbackPort:          true,
		InitialPruneDepth:     20_000,
		BootstrapManifestURLs: node.DefaultBootstrapManifestURLs(),
	})
	if err == nil {
		err = service.Start(ctx)
	}
	if err != nil && service != nil {
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = service.Close(shutdown)
		cancel()
		service = nil
	}
	a.mu.Lock()
	if a.closing {
		a.mu.Unlock()
		if service != nil {
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = service.Close(shutdown)
			cancel()
		}
		return
	}
	if a.service != nil && a.start == nil {
		a.mu.Unlock()
		if service != nil {
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = service.Close(shutdown)
			cancel()
		}
		return
	}
	a.service = service
	a.start = err
	a.mu.Unlock()
}

func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	a.closing = true
	cancelNode := a.cancel
	a.mu.Unlock()
	if cancelNode != nil {
		cancelNode()
	}
	a.wait.Wait()
	a.mu.RLock()
	service := a.service
	a.mu.RUnlock()
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

func (a *App) GetStartupState() StartupState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.service != nil && a.start == nil {
		return StartupState{Ready: true, Code: "ready", Message: "Entropy node is running"}
	}
	if errors.Is(a.start, node.ErrLegacyWalletMigrationRequired) {
		return StartupState{Code: "wallet_migration_required", Message: a.start.Error()}
	}
	if a.start != nil {
		return StartupState{Code: "startup_failed", Message: a.start.Error()}
	}
	return StartupState{Code: "starting", Message: "Entropy node is starting"}
}

func (a *App) RetryStartup() (StartupState, error) {
	a.mu.Lock()
	if a.closing {
		a.mu.Unlock()
		return StartupState{}, fmt.Errorf("application is shutting down")
	}
	if a.service != nil && a.start == nil {
		a.mu.Unlock()
		return a.GetStartupState(), nil
	}
	ctx := a.ctx
	a.wait.Add(1)
	a.mu.Unlock()
	defer a.wait.Done()
	a.startNode(ctx)
	state := a.GetStartupState()
	if !state.Ready && state.Code == "startup_failed" {
		return state, fmt.Errorf("%s", state.Message)
	}
	return state, nil
}

func (a *App) SendTransaction(to, amount string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	tx, fee, err := service.SendRecommended(to, amount)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{
		ID:      tx.ID,
		Message: fmt.Sprintf("Transaction added to local pending pool with %s ENT fee", core.FormatAmount(fee)),
	}, nil
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

func (a *App) RemovePeer(peer string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if err := service.RemovePeer(peer); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: "Peer removed"}, nil
}

func (a *App) GetTransactionHistory(limit int) ([]node.TransactionSummary, error) {
	service, err := a.readyService()
	if err != nil {
		return nil, err
	}
	return service.TransactionHistory(limit)
}

func (a *App) PruneLedger(retainRecent uint64) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	height, err := service.PruneLedger(retainRecent)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: fmt.Sprintf("Ledger pruned through block %d", height)}, nil
}

func (a *App) GetHealthEvents(activeOnly bool, limit int) ([]ledger.HealthEvent, error) {
	service, err := a.readyService()
	if err != nil {
		return nil, err
	}
	return service.HealthEvents(activeOnly, limit)
}

func (a *App) ResolveHealthEvent(id int64) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if err := service.ResolveHealthEvent(id); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: "Health event resolved"}, nil
}

func (a *App) GetRecoveryPhrase() (string, error) {
	service, err := a.readyService()
	if err != nil {
		return "", err
	}
	return service.RecoveryPhrase()
}

func (a *App) ConfirmWalletRecovery() (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if err := service.ConfirmWalletRecovery(); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: "Wallet recovery phrase confirmed"}, nil
}

func (a *App) ExportWalletBackup(password string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	path, err := wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
		Title:           "Export encrypted Entropy wallet",
		DefaultFilename: "entropy-wallet.entwallet",
		Filters:         []wailsruntime.FileFilter{{DisplayName: "Entropy wallet (*.entwallet)", Pattern: "*.entwallet"}},
	})
	if err != nil {
		return ActionResult{}, err
	}
	if path == "" {
		return ActionResult{Message: "Backup cancelled"}, nil
	}
	if err := service.ExportWalletBackup(path, password); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: "Encrypted wallet backup exported"}, nil
}

func (a *App) RestoreWalletBackup(password string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	path, err := wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:   "Restore encrypted Entropy wallet",
		Filters: []wailsruntime.FileFilter{{DisplayName: "Entropy wallet (*.entwallet)", Pattern: "*.entwallet"}},
	})
	if err != nil {
		return ActionResult{}, err
	}
	if path == "" {
		return ActionResult{Message: "Restore cancelled"}, nil
	}
	address, err := service.RestoreWalletBackup(path, password)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: address, Message: "Wallet imported and activated"}, nil
}

func (a *App) RestoreWalletMnemonic(phrase string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	address, err := service.RestoreWalletMnemonic(phrase)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: address, Message: "Wallet imported and activated"}, nil
}

func (a *App) CreateWallet() (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	address, err := service.CreateWallet()
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: address, Message: "New wallet created and activated"}, nil
}

func (a *App) SwitchWallet(address string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if err := service.SwitchWallet(address); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: address, Message: "Wallet activated"}, nil
}

func (a *App) RemoveWallet(address string) (ActionResult, error) {
	service, err := a.readyService()
	if err != nil {
		return ActionResult{}, err
	}
	if err := service.RemoveWallet(address); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{ID: address, Message: "Wallet removed from this device"}, nil
}

func (a *App) MigrateLegacyWallet(password string) (ActionResult, error) {
	a.mu.RLock()
	ctx := a.ctx
	startupErr := a.start
	a.mu.RUnlock()
	if !errors.Is(startupErr, node.ErrLegacyWalletMigrationRequired) {
		return ActionResult{}, fmt.Errorf("legacy wallet migration is not required")
	}
	path, err := wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
		Title:           "Create required legacy wallet backup",
		DefaultFilename: "entropy-legacy-wallet.entwallet",
		Filters:         []wailsruntime.FileFilter{{DisplayName: "Entropy wallet (*.entwallet)", Pattern: "*.entwallet"}},
	})
	if err != nil {
		return ActionResult{}, err
	}
	if path == "" {
		return ActionResult{Message: "Migration cancelled"}, nil
	}
	a.mu.Lock()
	if a.closing {
		a.mu.Unlock()
		return ActionResult{}, fmt.Errorf("application is shutting down")
	}
	a.wait.Add(1)
	a.mu.Unlock()
	defer a.wait.Done()
	if err := node.MigrateLegacyWallet("", path, password); err != nil {
		return ActionResult{}, err
	}
	a.startNode(ctx)
	if _, err := a.readyService(); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{Message: "Legacy wallet encrypted and node started"}, nil
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
