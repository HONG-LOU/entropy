package node

import (
	"context"
	"fmt"

	"entropy/internal/ledger"
	"entropy/internal/store"
)

func migrateLegacyState(ctx context.Context, storage *store.Store, chain *ledger.Ledger) error {
	state, found, err := storage.LoadLegacyState()
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	tip, err := chain.Tip(ctx)
	if err != nil {
		return err
	}
	legacyTip := state.Blocks[len(state.Blocks)-1]
	if tip.Height == 0 {
		if err := chain.ImportState(ctx, state); err != nil {
			return fmt.Errorf("migrate legacy chain to SQLite: %w", err)
		}
	} else if tip.Height != legacyTip.Height || tip.Hash != legacyTip.Hash {
		return fmt.Errorf("legacy chain and SQLite ledger disagree; refusing to replace either copy")
	}
	verified, err := chain.Tip(ctx)
	if err != nil {
		return err
	}
	if verified.Height != legacyTip.Height || verified.Hash != legacyTip.Hash {
		return fmt.Errorf("verify migrated SQLite tip: got %d/%s, want %d/%s",
			verified.Height, verified.Hash, legacyTip.Height, legacyTip.Hash)
	}
	if err := storage.ArchiveLegacy("chain.json"); err != nil {
		return err
	}
	return nil
}

func migrateLegacyPeers(ctx context.Context, storage *store.Store, chain *ledger.Ledger) error {
	peers, found, err := storage.LoadLegacyPeers()
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	for _, peer := range peers {
		if err := chain.UpsertPeer(ctx, peer, true); err != nil {
			return fmt.Errorf("migrate legacy peer %q: %w", peer, err)
		}
	}
	if err := storage.ArchiveLegacy("peers.json"); err != nil {
		return err
	}
	return nil
}
