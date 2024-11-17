package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

type LiteClient struct {
	ctx                 context.Context
	api                 ton.APIClientWrapped
	previousMasterBlock *ton.BlockIDExt
}

func New() *LiteClient {
	client := liteclient.NewConnectionPool()

	// cfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/global.config.json")
	filepath := "./globalconfig.json"
	cfg, err := liteclient.GetConfigFromFile(filepath)
	if err != nil {
		log.Fatalln("get config err: ", err.Error())
		return nil
	}

	err = client.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return nil
	}

	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(cfg)
	return &LiteClient{
		api: api,
		ctx: context.Background(),
	}
}

func main() {

	// Initialize the LiteClient
	liteClient := New()

	// Start processing
	liteClient.Start()
}

func (l *LiteClient) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := l.ProcessBlocks()
			if err != nil {
				log.Printf("Error processing blocks: %v", err)
			}
		case <-l.ctx.Done():
			return
		}
	}
}

func (l *LiteClient) ProcessBlocks() error {
	ctx := l.ctx
	api := l.api

	// Get the current masterchain block
	currentMasterBlock, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get masterchain info: %v", err)
	}

	// If this is the first run, set the previousMasterBlock and return
	if l.previousMasterBlock == nil {
		l.previousMasterBlock = currentMasterBlock
		log.Println("First run")
		return nil
	}

	// Collect masterchain blocks between previous and current masterchain blocks
	masterBlocks, err := l.collectMasterchainBlocks(ctx, currentMasterBlock, l.previousMasterBlock)
	if err != nil {
		return fmt.Errorf("failed to collect masterchain blocks: %v", err)
	}

	var blocks []*ton.BlockIDExt
	blocksMap := make(map[string]*ton.BlockIDExt)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, masterBlock := range masterBlocks {
		// Get shard blocks for each masterchain block
		shardBlocks, err := api.GetBlockShardsInfo(ctx, masterBlock)
		if err != nil {
			return fmt.Errorf("failed to get shard blocks: %v", err)
		}

		var workchain0Shards []*ton.BlockIDExt
		for _, shard := range shardBlocks {
			if shard.Workchain == 0 {
				workchain0Shards = append(workchain0Shards, shard)
			}
		}

		if len(workchain0Shards) == 0 {
			return fmt.Errorf("no workchain 0 shard blocks found at masterchain seqno %d", masterBlock.SeqNo)
		}

		resultCh := make(chan blockResult, len(workchain0Shards))

		for _, shardBlock := range workchain0Shards {
			wg.Add(1)
			go func(shardBlock *ton.BlockIDExt) {
				defer wg.Done()
				shardBlocksCollected, err := l.collectShardBlocks(ctx, shardBlock, l.previousMasterBlock.SeqNo)
				if err != nil {
					resultCh <- blockResult{nil, err}
					return
				}
				resultCh <- blockResult{shardBlocksCollected, nil}
			}(shardBlock)
		}

		wg.Wait()
		close(resultCh)

		for res := range resultCh {
			if res.err != nil {
				return fmt.Errorf("error collecting shard blocks: %v", res.err)
			}
			mu.Lock()
			for _, blk := range res.blocks {
				blockKey := fmt.Sprintf("%d:%d:%d", blk.Workchain, blk.Shard, blk.SeqNo)
				if _, exists := blocksMap[blockKey]; !exists {
					blocksMap[blockKey] = blk
					blocks = append(blocks, blk)
				}
			}
			mu.Unlock()
		}
	}

	// Update the previous master block for the next iteration
	l.previousMasterBlock = currentMasterBlock

	// Process the collected blocks as needed
	processBlocks(blocks)

	return nil
}

type blockResult struct {
	blocks []*ton.BlockIDExt
	err    error
}

/*
func (l *LiteClient) collectMasterchainBlocks(ctx context.Context, startBlock, endBlock *ton.BlockIDExt) ([]*ton.BlockIDExt, error) {
	var blocks []*ton.BlockIDExt
	queue := []*ton.BlockIDExt{startBlock}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		currentBlockID := queue[0]
		queue = queue[1:]

		blockKey := fmt.Sprintf("%d:%d:%d", currentBlockID.Workchain, currentBlockID.Shard, currentBlockID.SeqNo)
		if visited[blockKey] {
			continue
		}
		visited[blockKey] = true

		blocks = append(blocks, currentBlockID)

		// Termination condition
		if currentBlockID.SeqNo <= endBlock.SeqNo {
			if currentBlockID.Workchain == endBlock.Workchain &&
				currentBlockID.Shard == endBlock.Shard &&
				currentBlockID.SeqNo == endBlock.SeqNo &&
				bytes.Equal(currentBlockID.RootHash, endBlock.RootHash) &&
				bytes.Equal(currentBlockID.FileHash, endBlock.FileHash) {
				// Reached the end block
				break
			} else {
				continue
			}
		}

		// Fetch the block data
		blockData, err := l.api.GetBlockData(ctx, currentBlockID)
		if err != nil {
			return nil, fmt.Errorf("failed to get block data for block %d: %w", currentBlockID.SeqNo, err)
		}

		// Get previous block IDs
		prevBlockIDs, err := getPrevBlocks(&blockData.BlockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous blocks for block %d: %w", currentBlockID.SeqNo, err)
		}

		// Add previous blocks to the queue
		queue = append(queue, prevBlockIDs...)
	}

	return blocks, nil
}*/

func (l *LiteClient) collectShardBlocks(ctx context.Context, startBlock *ton.BlockIDExt, endSeqNo uint32) ([]*ton.BlockIDExt, error) {
	var blocks []*ton.BlockIDExt
	queue := []*ton.BlockIDExt{startBlock}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		currentBlockID := queue[0]
		queue = queue[1:]

		blockKey := fmt.Sprintf("%d:%d:%d", currentBlockID.Workchain, currentBlockID.Shard, currentBlockID.SeqNo)
		if visited[blockKey] {
			continue
		}
		visited[blockKey] = true

		blocks = append(blocks, currentBlockID)

		// Termination condition
		if currentBlockID.SeqNo <= endSeqNo {
			break
		}

		// Fetch the block data
		blockData, err := l.api.GetBlockData(ctx, currentBlockID)
		if err != nil {
			return nil, fmt.Errorf("failed to get block data for block %d: %w", currentBlockID.SeqNo, err)
		}

		// Get previous block IDs
		prevBlockIDs, err := getPrevBlocks(&blockData.BlockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous blocks for block %d: %w", currentBlockID.SeqNo, err)
		}

		// Add previous blocks to the queue
		queue = append(queue, prevBlockIDs...)
	}

	return blocks, nil
}

func getPrevBlocks(blockInfo *tlb.BlockHeader) ([]*ton.BlockIDExt, error) {
	prevBlocks, err := blockInfo.GetParentBlocks()
	if err != nil {
		return nil, err
	}
	return prevBlocks, nil
}
func (l *LiteClient) collectMasterchainBlocks(ctx context.Context, startBlock, endBlock *ton.BlockIDExt) ([]*ton.BlockIDExt, error) {
	var blocks []*ton.BlockIDExt
	limit := 100
	queue := []*ton.BlockIDExt{startBlock}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		currentBlockID := queue[0]
		queue = queue[1:]

		blockKey := fmt.Sprintf("%d:%d:%d", currentBlockID.Workchain, currentBlockID.Shard, currentBlockID.SeqNo)
		if visited[blockKey] {
			continue
		}
		visited[blockKey] = true

		blocks = append(blocks, currentBlockID)

		// Check if we've reached the end block
		if currentBlockID.SeqNo <= endBlock.SeqNo {
			if currentBlockID.Workchain == endBlock.Workchain &&
				currentBlockID.Shard == endBlock.Shard &&
				currentBlockID.SeqNo == endBlock.SeqNo &&
				bytes.Equal(currentBlockID.RootHash, endBlock.RootHash) &&
				bytes.Equal(currentBlockID.FileHash, endBlock.FileHash) {
				// Reached the end block, terminate the function
				break
			}
		}

		// Check if we've reached the limit
		if len(blocks) >= limit {
			break
		}

		// Fetch the block data
		blockData, err := l.api.GetBlockData(ctx, currentBlockID)
		if err != nil {
			return nil, fmt.Errorf("failed to get block data for block %d: %w", currentBlockID.SeqNo, err)
		}

		// Get previous block IDs
		prevBlockIDs, err := getPrevBlocks(&blockData.BlockInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous blocks for block %d: %w", currentBlockID.SeqNo, err)
		}

		// Add previous blocks to the queue
		queue = append(queue, prevBlockIDs...)
	}

	return blocks, nil
}

func processBlocks(blocks []*ton.BlockIDExt) {
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].SeqNo == blocks[j].SeqNo {
			return blocks[i].Shard > blocks[j].Shard
		}
		return blocks[i].SeqNo > blocks[j].SeqNo
	})

	for _, blk := range blocks {
		fmt.Printf("Block: Workchain %d, Shard %d, SeqNo %d\n", blk.Workchain, blk.Shard, blk.SeqNo)
	}
}
