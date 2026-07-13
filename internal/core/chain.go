package core

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"time"
)

func (s *State) Validate() error {
	_, _, err := s.replay(true)
	return err
}

func (s *State) ConfirmedUTXO() (UTXO, error) {
	utxo, _, err := s.replay(false)
	return utxo, err
}

func (s *State) SpendableUTXO() (UTXO, error) {
	utxo, _, err := s.replay(true)
	return utxo, err
}

func (s *State) Balance(address string, includePending bool) (uint64, error) {
	if err := ValidateAddress(address); err != nil {
		return 0, err
	}
	var (
		utxo UTXO
		err  error
	)
	if includePending {
		utxo, err = s.SpendableUTXO()
	} else {
		utxo, err = s.ConfirmedUTXO()
	}
	if err != nil {
		return 0, err
	}
	total := uint64(0)
	for _, output := range utxo {
		if output.Address != address {
			continue
		}
		total, err = safeAdd(total, output.Amount)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (s *State) Balances(address string) (uint64, uint64, error) {
	if err := ValidateAddress(address); err != nil {
		return 0, 0, err
	}
	utxo, seen, err := s.replay(false)
	if err != nil {
		return 0, 0, err
	}
	confirmed, err := balanceFromUTXO(utxo, address)
	if err != nil {
		return 0, 0, err
	}
	if len(s.Pending) > MaxPendingTransactions {
		return 0, 0, fmt.Errorf("pending transaction limit exceeded")
	}
	for index, tx := range s.Pending {
		if _, exists := seen[tx.ID]; exists {
			return 0, 0, fmt.Errorf("pending transaction %d is duplicated", index)
		}
		if _, err := validateRegularTransaction(tx, utxo); err != nil {
			return 0, 0, fmt.Errorf("pending transaction %d: %w", index, err)
		}
		if err := applyRegularTransaction(tx, utxo); err != nil {
			return 0, 0, err
		}
		seen[tx.ID] = struct{}{}
	}
	spendable, err := balanceFromUTXO(utxo, address)
	if err != nil {
		return 0, 0, err
	}
	return confirmed, spendable, nil
}

func balanceFromUTXO(utxo UTXO, address string) (uint64, error) {
	total := uint64(0)
	for _, output := range utxo {
		if output.Address != address {
			continue
		}
		var err error
		total, err = safeAdd(total, output.Amount)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (s *State) AddPending(tx Transaction) error {
	if len(s.Pending) >= MaxPendingTransactions {
		return fmt.Errorf("pending transaction limit reached")
	}
	utxo, seen, err := s.replay(true)
	if err != nil {
		return err
	}
	if _, exists := seen[tx.ID]; exists {
		return fmt.Errorf("transaction %s already exists", tx.ID)
	}
	if _, err := validateRegularTransaction(tx, utxo); err != nil {
		return err
	}
	if err := applyRegularTransaction(tx, utxo); err != nil {
		return err
	}
	s.Pending = append(s.Pending, tx)
	return nil
}

func (s *State) Mine(ctx context.Context, address string) (Block, error) {
	confirmed, seen, err := s.replay(false)
	if err != nil {
		return Block{}, err
	}
	if err := ValidateAddress(address); err != nil {
		return Block{}, err
	}

	fees := uint64(0)
	selected := make([]Transaction, 0, MaxBlockTransactions-1)
	blockBytes := 512
	for _, tx := range s.Pending {
		transactionBytes := tx.encodedSize()
		if len(selected) >= MaxBlockTransactions-1 || blockBytes+transactionBytes > MaxBlockBytes {
			break
		}
		if _, exists := seen[tx.ID]; exists {
			return Block{}, fmt.Errorf("duplicate pending transaction %s", tx.ID)
		}
		fee, err := validateRegularTransaction(tx, confirmed)
		if err != nil {
			return Block{}, fmt.Errorf("pending transaction %s: %w", tx.ID, err)
		}
		fees, err = safeAdd(fees, fee)
		if err != nil {
			return Block{}, err
		}
		if err := applyRegularTransaction(tx, confirmed); err != nil {
			return Block{}, err
		}
		seen[tx.ID] = struct{}{}
		selected = append(selected, tx)
		blockBytes += transactionBytes
	}

	height := uint64(len(s.Blocks))
	reward, err := safeAdd(Subsidy(height), fees)
	if err != nil {
		return Block{}, err
	}
	coinbase, err := NewCoinbase(address, height, reward)
	if err != nil {
		return Block{}, err
	}
	transactions := make([]Transaction, 0, len(selected)+1)
	transactions = append(transactions, coinbase)
	transactions = append(transactions, selected...)
	previous := s.Blocks[len(s.Blocks)-1]
	block := Block{
		Version:      StateVersion,
		Height:       height,
		Timestamp:    nextTimestamp(s.Blocks),
		PreviousHash: previous.Hash,
		MerkleRoot:   merkleRoot(transactions),
		Difficulty:   expectedDifficulty(s.Blocks, height),
		Transactions: transactions,
	}
	mined, err := mineBlock(ctx, block)
	if err != nil {
		return Block{}, err
	}

	oldPending := s.Pending
	s.Blocks = append(s.Blocks, mined)
	s.Pending = append([]Transaction(nil), s.Pending[len(selected):]...)
	if err := s.Validate(); err != nil {
		s.Blocks = s.Blocks[:len(s.Blocks)-1]
		s.Pending = oldPending
		return Block{}, fmt.Errorf("mined invalid block: %w", err)
	}
	return mined, nil
}

func (s *State) TotalIssued() (uint64, error) {
	if _, _, err := s.replay(false); err != nil {
		return 0, err
	}
	return MintedThrough(uint64(len(s.Blocks) - 1)), nil
}

func (s *State) CumulativeWork() *big.Int {
	work := new(big.Int)
	for _, block := range s.Blocks[1:] {
		work.Add(work, new(big.Int).Lsh(big.NewInt(1), uint(block.Difficulty)))
	}
	return work
}

func (s *State) replay(includePending bool) (UTXO, map[string]struct{}, error) {
	if s.Version != StateVersion || s.Name != ChainName || s.Symbol != ChainSymbol {
		return nil, nil, fmt.Errorf("unsupported chain identity or state version")
	}
	if len(s.Blocks) == 0 || !reflect.DeepEqual(s.Blocks[0], GenesisBlock()) {
		return nil, nil, fmt.Errorf("genesis block mismatch")
	}

	utxo := make(UTXO)
	seen := make(map[string]struct{})
	for index := 1; index < len(s.Blocks); index++ {
		block := s.Blocks[index]
		previous := s.Blocks[index-1]
		if err := validateBlockHeader(block, previous, s.Blocks[:index]); err != nil {
			return nil, nil, fmt.Errorf("block %d: %w", block.Height, err)
		}
		if len(block.Transactions) == 0 || !block.Transactions[0].Coinbase {
			return nil, nil, fmt.Errorf("block %d: first transaction must be coinbase", block.Height)
		}

		fees := uint64(0)
		for txIndex, tx := range block.Transactions[1:] {
			if tx.Coinbase {
				return nil, nil, fmt.Errorf("block %d: multiple coinbase transactions", block.Height)
			}
			if _, exists := seen[tx.ID]; exists {
				return nil, nil, fmt.Errorf("block %d: duplicate transaction %s", block.Height, tx.ID)
			}
			fee, err := validateRegularTransaction(tx, utxo)
			if err != nil {
				return nil, nil, fmt.Errorf("block %d transaction %d: %w", block.Height, txIndex+1, err)
			}
			fees, err = safeAdd(fees, fee)
			if err != nil {
				return nil, nil, err
			}
			if err := applyRegularTransaction(tx, utxo); err != nil {
				return nil, nil, err
			}
			seen[tx.ID] = struct{}{}
		}

		coinbase := block.Transactions[0]
		if _, exists := seen[coinbase.ID]; exists {
			return nil, nil, fmt.Errorf("block %d: duplicate coinbase transaction", block.Height)
		}
		maximum, err := safeAdd(Subsidy(block.Height), fees)
		if err != nil {
			return nil, nil, err
		}
		if err := validateCoinbase(coinbase, maximum); err != nil {
			return nil, nil, fmt.Errorf("block %d: %w", block.Height, err)
		}
		if err := applyOutputs(coinbase, utxo); err != nil {
			return nil, nil, err
		}
		seen[coinbase.ID] = struct{}{}
	}

	if includePending {
		if len(s.Pending) > MaxPendingTransactions {
			return nil, nil, fmt.Errorf("pending transaction limit exceeded")
		}
		for index, tx := range s.Pending {
			if _, exists := seen[tx.ID]; exists {
				return nil, nil, fmt.Errorf("pending transaction %d is duplicated", index)
			}
			if _, err := validateRegularTransaction(tx, utxo); err != nil {
				return nil, nil, fmt.Errorf("pending transaction %d: %w", index, err)
			}
			if err := applyRegularTransaction(tx, utxo); err != nil {
				return nil, nil, err
			}
			seen[tx.ID] = struct{}{}
		}
	}
	return utxo, seen, nil
}

func validateBlockHeader(block, previous Block, priorBlocks []Block) error {
	if block.Version != StateVersion {
		return fmt.Errorf("unsupported block version")
	}
	if block.Height != previous.Height+1 {
		return fmt.Errorf("height does not follow previous block")
	}
	if block.PreviousHash != previous.Hash {
		return fmt.Errorf("previous hash mismatch")
	}
	if block.Timestamp <= medianTimePast(priorBlocks) {
		return fmt.Errorf("timestamp must exceed median time past")
	}
	if block.Timestamp > time.Now().Unix()+MaxFutureSeconds {
		return fmt.Errorf("timestamp is too far in the future")
	}
	if block.Difficulty != expectedDifficulty(priorBlocks, block.Height) {
		return fmt.Errorf("unexpected difficulty")
	}
	if len(block.Transactions) == 0 || len(block.Transactions) > MaxBlockTransactions {
		return fmt.Errorf("block transaction count is outside consensus limits")
	}
	encodedSize := 256
	for _, tx := range block.Transactions {
		transactionBytes := tx.encodedSize()
		if transactionBytes > MaxTransactionBytes {
			return fmt.Errorf("block contains oversized transaction")
		}
		encodedSize += transactionBytes
		if encodedSize > MaxBlockBytes {
			return fmt.Errorf("block exceeds encoded size limit")
		}
	}
	if block.MerkleRoot != merkleRoot(block.Transactions) {
		return fmt.Errorf("Merkle root mismatch")
	}
	if block.Hash != block.ComputeHash() {
		return fmt.Errorf("block hash mismatch")
	}
	if !block.HasValidWork() {
		return fmt.Errorf("proof of work is insufficient")
	}
	return nil
}

func applyRegularTransaction(tx Transaction, utxo UTXO) error {
	for _, input := range tx.Inputs {
		delete(utxo, Outpoint{TxID: input.TxID, Index: input.OutputIndex})
	}
	return applyOutputs(tx, utxo)
}

func applyOutputs(tx Transaction, utxo UTXO) error {
	for index, output := range tx.Outputs {
		outpoint := Outpoint{TxID: tx.ID, Index: uint32(index)}
		if _, exists := utxo[outpoint]; exists {
			return fmt.Errorf("duplicate output %s:%d", tx.ID, index)
		}
		utxo[outpoint] = output
	}
	return nil
}
