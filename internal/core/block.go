package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const genesisTimestamp int64 = 1783900800

var zeroHash = strings.Repeat("0", 64)

func NewState() *State {
	return &State{
		Version: StateVersion,
		Name:    ChainName,
		Symbol:  ChainSymbol,
		Blocks:  []Block{GenesisBlock()},
		Pending: []Transaction{},
	}
}

func GenesisBlock() Block {
	block := Block{
		Version:      StateVersion,
		Height:       0,
		Timestamp:    genesisTimestamp,
		PreviousHash: zeroHash,
		MerkleRoot:   merkleRoot(nil),
		Difficulty:   0,
		Transactions: []Transaction{},
	}
	block.Hash = block.ComputeHash()
	return block
}

func (b Block) ComputeHash() string {
	var e encoder
	e.uint32(b.Version)
	e.uint64(b.Height)
	e.int64(b.Timestamp)
	e.string(b.PreviousHash)
	e.string(b.MerkleRoot)
	e.uint8(b.Difficulty)
	e.uint64(b.Nonce)
	return hashHex(e.Bytes())
}

func (b Block) HasValidWork() bool {
	decoded, err := decodeHash(b.Hash)
	if err != nil {
		return false
	}
	return leadingZeroBits(decoded) >= int(b.Difficulty)
}

func mineBlock(ctx context.Context, block Block) (Block, error) {
	for nonce := uint64(0); ; nonce++ {
		if nonce%100_000 == 0 {
			select {
			case <-ctx.Done():
				return Block{}, ctx.Err()
			default:
			}
		}
		block.Nonce = nonce
		block.Hash = block.ComputeHash()
		if block.HasValidWork() {
			return block, nil
		}
		if nonce == math.MaxUint64 {
			return Block{}, fmt.Errorf("nonce space exhausted")
		}
	}
}

func expectedDifficulty(blocks []Block, nextHeight uint64) uint8 {
	if nextHeight == 1 {
		return InitialDifficulty
	}
	previous := blocks[len(blocks)-1].Difficulty
	if nextHeight < FirstAdjustment || nextHeight%AdjustmentBlocks != 0 {
		return previous
	}

	lastTime := medianTimeAt(blocks, len(blocks)-1)
	firstTime := medianTimeAt(blocks, len(blocks)-1-AdjustmentBlocks)
	actualSeconds := lastTime - firstTime
	targetSeconds := int64(AdjustmentBlocks) * TargetBlockSeconds
	if actualSeconds <= targetSeconds/4 {
		return clampDifficulty(int(previous) + 2)
	}
	if actualSeconds < targetSeconds/2 {
		return clampDifficulty(int(previous) + 1)
	}
	if actualSeconds >= targetSeconds*4 {
		return clampDifficulty(int(previous) - 2)
	}
	if actualSeconds > targetSeconds*2 {
		return clampDifficulty(int(previous) - 1)
	}
	return previous
}

func clampDifficulty(difficulty int) uint8 {
	if difficulty < MinimumDifficulty {
		return MinimumDifficulty
	}
	if difficulty > MaximumDifficulty {
		return MaximumDifficulty
	}
	return uint8(difficulty)
}

func medianTimePast(blocks []Block) int64 {
	return medianTimeAt(blocks, len(blocks)-1)
}

func medianTimeAt(blocks []Block, end int) int64 {
	start := end - MedianTimeBlocks + 1
	if start < 0 {
		start = 0
	}
	timestamps := make([]int64, 0, end-start+1)
	for index := start; index <= end; index++ {
		timestamps = append(timestamps, blocks[index].Timestamp)
	}
	sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })
	return timestamps[len(timestamps)/2]
}

func merkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		empty := sha256.Sum256(nil)
		return hex.EncodeToString(empty[:])
	}
	level := make([][]byte, 0, len(transactions))
	for _, tx := range transactions {
		decoded, err := decodeHash(tx.ID)
		if err != nil {
			return ""
		}
		level = append(level, decoded)
	}
	for len(level) > 1 {
		next := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[i]
			if i+1 < len(level) {
				right = level[i+1]
			}
			combined := append(append([]byte(nil), level[i]...), right...)
			hash := sha256.Sum256(combined)
			next = append(next, hash[:])
		}
		level = next
	}
	return hex.EncodeToString(level[0])
}

func leadingZeroBits(value []byte) int {
	count := 0
	for _, b := range value {
		if b == 0 {
			count += 8
			continue
		}
		for mask := byte(0x80); mask != 0 && b&mask == 0; mask >>= 1 {
			count++
		}
		break
	}
	return count
}

func nextTimestamp(blocks []Block) int64 {
	now := time.Now().Unix()
	minimum := medianTimePast(blocks) + 1
	if now < minimum {
		return minimum
	}
	return now
}
