package node

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"entropy/internal/core"
	"entropy/internal/ledger"
)

const (
	maxDirectExtensionBlocks = maxBlockBatch
	maxStagedForkBytes       = int64(512 << 20)
	maxBlockDownloadBatch    = 8
)

func cleanupStaleStagingFiles(directory string) error {
	paths, err := filepath.Glob(filepath.Join(directory, "incoming-chain-*.tmp"))
	if err != nil {
		return fmt.Errorf("scan stale incoming chain files: %w", err)
	}
	for _, path := range paths {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect stale incoming chain file: %w", err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove stale incoming chain file: %w", err)
		}
	}
	return nil
}

type stagedBlockFile struct {
	file       *os.File
	reader     *bufio.Reader
	path       string
	nextIndex  int
	blockCount int
	release    func()
	closeOnce  sync.Once
	closeErr   error
}

type remoteBlockSource interface {
	requestBlocks(context.Context, blocksRequest) (blocksResponse, error)
}

type httpRemoteSource struct {
	service *Service
	peer    string
}

func (h httpRemoteSource) requestHeaders(ctx context.Context, request headersRequest) (headersResponse, error) {
	var response headersResponse
	err := h.service.postJSON(ctx, h.peer+"/v2/headers", request, maxProtocolBytes, &response)
	return response, err
}

func (h httpRemoteSource) requestBlocks(ctx context.Context, request blocksRequest) (blocksResponse, error) {
	var response blocksResponse
	maximum := int64(len(request.Hashes))*maxBlockBodyBytes + 64<<10
	err := h.service.postJSON(ctx, h.peer+"/v2/blocks", request, maximum, &response)
	return response, err
}

func (s *Service) stageBlocks(ctx context.Context, peer string, headers []core.Block) (*stagedBlockFile, error) {
	return s.stageBlocksWithBudget(ctx, peer, headers, maxStagedForkBytes)
}

func (s *Service) stageBlocksWithBudget(ctx context.Context, peer string, headers []core.Block, budget int64) (*stagedBlockFile, error) {
	return s.stageBlocksFromSourceWithBudget(ctx, httpRemoteSource{service: s, peer: peer}, headers, budget)
}

func (s *Service) stageBlocksFromSource(
	ctx context.Context,
	source remoteBlockSource,
	headers []core.Block,
) (*stagedBlockFile, error) {
	return s.stageBlocksFromSourceWithBudget(ctx, source, headers, maxStagedForkBytes)
}

func (s *Service) stageBlocksFromSourceWithBudget(
	ctx context.Context,
	source remoteBlockSource,
	headers []core.Block,
	budget int64,
) (*stagedBlockFile, error) {
	if len(headers) == 0 || len(headers) > maxHeadersPerSync {
		return nil, fmt.Errorf("staged block count is outside sync limits")
	}
	if budget <= 0 || budget > maxStagedForkBytes {
		return nil, fmt.Errorf("staging budget is outside safe limits")
	}
	select {
	case s.stagingSlot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	release := sync.OnceFunc(func() { <-s.stagingSlot })
	file, err := os.CreateTemp(filepath.Dir(s.ledger.Path()), "incoming-chain-*.tmp")
	if err != nil {
		release()
		return nil, fmt.Errorf("create incoming chain staging file: %w", err)
	}
	staged := &stagedBlockFile{file: file, path: file.Name(), blockCount: len(headers), release: release}
	keep := false
	defer func() {
		if !keep {
			_ = staged.Close()
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return nil, fmt.Errorf("protect incoming chain staging file: %w", err)
	}
	writer := bufio.NewWriterSize(file, 1<<20)
	totalBytes := int64(0)
	for start := 0; start < len(headers); start += maxBlockDownloadBatch {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(start+maxBlockDownloadBatch, len(headers))
		hashes := make([]string, 0, end-start)
		for _, header := range headers[start:end] {
			hashes = append(hashes, header.Hash)
		}
		response, err := source.requestBlocks(ctx, blocksRequest{Hashes: hashes})
		if err != nil {
			return nil, err
		}
		if response.Protocol != ledger.ProtocolName || len(response.Blocks) != len(hashes) {
			return nil, fmt.Errorf("peer returned an incomplete block batch")
		}
		for index, block := range response.Blocks {
			header := headers[start+index]
			if !sameBlockHeader(block, header) {
				return nil, fmt.Errorf("block body does not match requested header %s", header.Hash)
			}
			encoded, err := json.Marshal(block)
			if err != nil {
				return nil, fmt.Errorf("encode staged block %s: %w", block.Hash, err)
			}
			if int64(len(encoded)) > maxBlockBodyBytes {
				return nil, fmt.Errorf("block %s exceeds staging size limit", block.Hash)
			}
			totalBytes += int64(4 + len(encoded))
			if totalBytes > budget {
				return nil, fmt.Errorf("candidate branch exceeds %d-byte staging budget", budget)
			}
			if err := binary.Write(writer, binary.BigEndian, uint32(len(encoded))); err != nil {
				return nil, fmt.Errorf("write staged block length: %w", err)
			}
			if _, err := writer.Write(encoded); err != nil {
				return nil, fmt.Errorf("write staged block: %w", err)
			}
		}
	}
	if err := writer.Flush(); err != nil {
		return nil, fmt.Errorf("flush incoming chain staging file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("sync incoming chain staging file: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind incoming chain staging file: %w", err)
	}
	staged.reader = bufio.NewReaderSize(file, 1<<20)
	keep = true
	return staged, nil
}

func (s *stagedBlockFile) BlockAt(index int) (core.Block, error) {
	if s == nil || s.file == nil || s.reader == nil {
		return core.Block{}, fmt.Errorf("incoming chain staging file is closed")
	}
	if index != s.nextIndex || index < 0 || index >= s.blockCount {
		return core.Block{}, fmt.Errorf("staged block requested out of order")
	}
	var length uint32
	if err := binary.Read(s.reader, binary.BigEndian, &length); err != nil {
		return core.Block{}, err
	}
	if length == 0 || int64(length) > maxBlockBodyBytes {
		return core.Block{}, fmt.Errorf("staged block length is invalid")
	}
	data := make([]byte, int(length))
	if _, err := io.ReadFull(s.reader, data); err != nil {
		return core.Block{}, err
	}
	var block core.Block
	if err := decodeLimitedJSONBytes(data, &block); err != nil {
		return core.Block{}, err
	}
	s.nextIndex++
	return block, nil
}

func decodeLimitedJSONBytes(data []byte, value any) error {
	return decodeLimitedJSON(bytes.NewReader(data), int64(len(data)), value)
}

func (s *stagedBlockFile) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		var closeErr error
		if s.file != nil {
			closeErr = s.file.Close()
			s.file = nil
		}
		removeErr := os.Remove(s.path)
		if os.IsNotExist(removeErr) {
			removeErr = nil
		}
		s.closeErr = errors.Join(closeErr, removeErr)
		if s.release != nil {
			s.release()
		}
	})
	return s.closeErr
}
