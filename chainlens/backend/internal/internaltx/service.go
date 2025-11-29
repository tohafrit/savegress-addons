package internaltx

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"getchainlens.com/chainlens/backend/internal/tracer"
)

// Service provides internal transaction business logic
type Service struct {
	repo       *Repository
	tracer     *tracer.Tracer
	rpcURLs    map[string]string
	stopCh     chan struct{}
	wg         sync.WaitGroup
	maxRetries int
	batchSize  int
}

// NewService creates a new internal transaction service
func NewService(repo *Repository, tracer *tracer.Tracer) *Service {
	return &Service{
		repo:       repo,
		tracer:     tracer,
		rpcURLs:    make(map[string]string),
		stopCh:     make(chan struct{}),
		maxRetries: 3,
		batchSize:  10,
	}
}

// SetRPCURL sets the RPC URL for a network
func (s *Service) SetRPCURL(network, rpcURL string) {
	s.rpcURLs[network] = rpcURL
}

// Start starts background trace processing workers
func (s *Service) Start() {
	s.wg.Add(1)
	go s.traceProcessingWorker()
}

// Stop stops background workers
func (s *Service) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// traceProcessingWorker processes pending trace jobs
func (s *Service) traceProcessingWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processPendingTraces()
		}
	}
}

// processPendingTraces processes a batch of pending traces
func (s *Service) processPendingTraces() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for network, rpcURL := range s.rpcURLs {
		jobs, err := s.repo.GetPendingTraces(ctx, network, s.batchSize, s.maxRetries)
		if err != nil {
			log.Printf("Error getting pending traces for %s: %v", network, err)
			continue
		}

		if len(jobs) == 0 {
			continue
		}

		// Mark as processing
		txHashes := make([]string, len(jobs))
		for i, job := range jobs {
			txHashes[i] = job.TxHash
		}
		if err := s.repo.MarkAsProcessing(ctx, network, txHashes); err != nil {
			log.Printf("Error marking traces as processing: %v", err)
			continue
		}

		// Process each job
		for _, job := range jobs {
			if err := s.TraceAndStore(ctx, network, job.TxHash, rpcURL); err != nil {
				errMsg := err.Error()
				s.repo.UpdateProcessingStatus(ctx, network, job.TxHash, StatusFailed, &errMsg)
				log.Printf("Error tracing %s on %s: %v", job.TxHash, network, err)
			} else {
				s.repo.UpdateProcessingStatus(ctx, network, job.TxHash, StatusCompleted, nil)
			}
		}
	}
}

// TraceAndStore traces a transaction and stores the internal transactions
func (s *Service) TraceAndStore(ctx context.Context, network, txHash, rpcURL string) error {
	// Get trace from RPC
	traceResult, err := s.tracer.TraceTransaction(ctx, rpcURL, txHash)
	if err != nil {
		return fmt.Errorf("trace transaction: %w", err)
	}

	// Convert trace calls to internal transactions
	internalTxs := s.convertTraceCalls(network, txHash, traceResult)

	if len(internalTxs) == 0 {
		return nil
	}

	// Store in batch
	if err := s.repo.InsertInternalTransactionsBatch(ctx, internalTxs); err != nil {
		return fmt.Errorf("insert internal transactions: %w", err)
	}

	return nil
}

// convertTraceCalls converts tracer.InternalCall to InternalTransaction records
func (s *Service) convertTraceCalls(network, txHash string, trace *tracer.TraceResult) []*InternalTransaction {
	if trace == nil || len(trace.Calls) == 0 {
		return nil
	}

	var result []*InternalTransaction
	timestamp := time.Now().UTC() // Will be updated from block data

	var flattenCalls func(calls []tracer.InternalCall, parentIdx *int, depth int)
	traceIndex := 0

	flattenCalls = func(calls []tracer.InternalCall, parentIdx *int, depth int) {
		for _, call := range calls {
			idx := traceIndex
			traceIndex++

			tx := &InternalTransaction{
				Network:          network,
				TxHash:           txHash,
				TraceIndex:       idx,
				BlockNumber:      int64(trace.BlockNumber),
				ParentTraceIndex: parentIdx,
				Depth:            depth,
				TraceType:        NormalizeTraceType(call.Type),
				FromAddress:      call.From,
				Value:            call.Value,
				GasUsed:          int64Ptr(int64(call.GasUsed)),
				Reverted:         call.Error != "",
				Timestamp:        timestamp,
			}

			if call.To != "" {
				tx.ToAddress = &call.To
			}

			if call.Input != "" && call.Input != "0x" {
				tx.InputData = &call.Input
			}

			if call.Output != "" && call.Output != "0x" {
				tx.OutputData = &call.Output
			}

			if call.Error != "" {
				tx.Error = &call.Error
			}

			// For CREATE operations, the 'to' becomes created_contract
			if IsContractCreation(tx.TraceType) && call.To != "" {
				tx.CreatedContract = &call.To
				tx.ToAddress = nil
			}

			result = append(result, tx)

			// Process children
			if len(call.Children) > 0 {
				flattenCalls(call.Children, &idx, depth+1)
			}
		}
	}

	flattenCalls(trace.Calls, nil, 0)

	return result
}

// QueueForTracing adds a transaction to the trace processing queue
func (s *Service) QueueForTracing(ctx context.Context, network, txHash string, blockNumber int64) error {
	_, err := s.repo.GetOrCreateProcessingStatus(ctx, network, txHash, blockNumber)
	return err
}

// GetInternalTransactions returns internal transactions for a transaction hash
func (s *Service) GetInternalTransactions(ctx context.Context, network, txHash string) ([]*InternalTransaction, error) {
	return s.repo.GetByTxHash(ctx, network, txHash)
}

// GetInternalTransactionTree returns internal transactions as a tree structure
func (s *Service) GetInternalTransactionTree(ctx context.Context, network, txHash string) (*TraceTree, error) {
	txs, err := s.repo.GetByTxHash(ctx, network, txHash)
	if err != nil {
		return nil, err
	}

	if len(txs) == 0 {
		return nil, nil
	}

	// Build tree from flat list
	return buildTraceTree(txs), nil
}

// GetAddressInternalTransactions returns internal transactions for an address
func (s *Service) GetAddressInternalTransactions(ctx context.Context, network, address string, page, pageSize int) ([]*InternalTransaction, error) {
	filter := &InternalTxFilter{
		Network:     network,
		FromAddress: address,
		ToAddress:   address,
		Page:        page,
		PageSize:    pageSize,
	}
	return s.repo.GetByAddress(ctx, filter)
}

// GetCallStats returns statistics about internal calls for a transaction
func (s *Service) GetCallStats(ctx context.Context, network, txHash string) (*CallStats, error) {
	return s.repo.GetCallStats(ctx, network, txHash)
}

// GetCreatedContracts returns contracts created via internal transactions
func (s *Service) GetCreatedContracts(ctx context.Context, network string, limit, offset int) ([]*InternalTransaction, error) {
	return s.repo.GetCreatedContracts(ctx, network, limit, offset)
}

// GetProcessingStatus returns the trace processing status for a transaction
func (s *Service) GetProcessingStatus(ctx context.Context, network, txHash string) (*TraceProcessingStatus, error) {
	return s.repo.GetProcessingStatus(ctx, network, txHash)
}

// GetProcessingStats returns processing statistics for a network
func (s *Service) GetProcessingStats(ctx context.Context, network string) (map[string]int64, error) {
	return s.repo.CountByStatus(ctx, network)
}

// TraceTransactionOnDemand traces a transaction immediately (not queued)
func (s *Service) TraceTransactionOnDemand(ctx context.Context, network, txHash string) ([]*InternalTransaction, error) {
	rpcURL, ok := s.rpcURLs[network]
	if !ok {
		return nil, fmt.Errorf("no RPC URL configured for network %s", network)
	}

	// Check if already traced
	existing, err := s.repo.GetByTxHash(ctx, network, txHash)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return existing, nil
	}

	// Trace and store
	if err := s.TraceAndStore(ctx, network, txHash, rpcURL); err != nil {
		return nil, err
	}

	// Return stored results
	return s.repo.GetByTxHash(ctx, network, txHash)
}

// Helper functions

func buildTraceTree(txs []*InternalTransaction) *TraceTree {
	if len(txs) == 0 {
		return nil
	}

	// Create map of trace_index -> node
	nodeMap := make(map[int]*TraceTree)
	for _, tx := range txs {
		nodeMap[tx.TraceIndex] = &TraceTree{
			Call:     tx,
			Children: nil,
		}
	}

	// Build tree by linking children to parents
	var root *TraceTree
	for _, tx := range txs {
		node := nodeMap[tx.TraceIndex]
		if tx.ParentTraceIndex == nil {
			root = node
		} else {
			if parent, ok := nodeMap[*tx.ParentTraceIndex]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return root
}

func int64Ptr(v int64) *int64 {
	return &v
}
