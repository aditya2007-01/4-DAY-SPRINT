package errors

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "bhiv-chain-inspector/internal/blocks"
    "bhiv-chain-inspector/internal/db"
)

type ErrorScanResult struct {
    ScanTime                string   `json:"scan_time"`
    DatabasePath            string   `json:"database_path"`
    TotalBlocks             int      `json:"total_blocks"`
    BlocksScanned           int      `json:"blocks_scanned"`
    TotalErrors             int      `json:"total_errors"`
    CorruptedJSON           []string `json:"corrupted_json"`
    BadHash                 []string `json:"bad_hash"`
    TimestampFuture         []string `json:"timestamp_future"`
    TimestampPast           []string `json:"timestamp_past"`
    TimestampNotIncreasing  []string `json:"timestamp_not_increasing"`
    DuplicateHashes         []string `json:"duplicate_hashes"`
    EmptyBlocks             []string `json:"empty_blocks"`
    PrevHashErrors          []string `json:"prevhash_errors"`
    HeightErrors            []string `json:"height_errors"`
    MissingBlocks           []int    `json:"missing_blocks"`
    OutOfOrderBlocks        []string `json:"out_of_order_blocks"`
    HealthScore             int      `json:"health_score"`
    Status                  string   `json:"status"`
}

func ScanErrors(storage *db.Storage, dbPath string) *ErrorScanResult {
    result := &ErrorScanResult{
        ScanTime:     time.Now().Format("2006-01-02 15:04:05"),
        DatabasePath: dbPath,
    }

    height := storage.GetMaxHeight()
    if height < 0 {
        result.Status = "ERROR: Empty database"
        result.HealthScore = 0
        return result
    }

    result.TotalBlocks = height + 1
    seenHashes := make(map[string]int)
    var prevBlock *blocks.Block
    expectedHeight := 0
    currentTime := time.Now().Unix()

    for i := 0; i <= height+10; i++ {
        rawData, rawErr := storage.LoadBlockRaw(i)
        
        if rawErr != nil {
            if i <= height {
                result.MissingBlocks = append(result.MissingBlocks, i)
                result.TotalErrors++
            }
            if i > height {
                break
            }
            continue
        }

        var block blocks.Block
        err := json.Unmarshal(rawData, &block)
        if err != nil {
            errMsg := fmt.Sprintf("Block %d: Corrupted JSON - %v", i, err)
            result.CorruptedJSON = append(result.CorruptedJSON, errMsg)
            result.TotalErrors++
            continue
        }

        result.BlocksScanned++

        // Hash validation
        computedHash := blocks.ComputeHash(block.Height, block.PrevHash, block.Data, block.Timestamp)
        if block.Hash != computedHash {
            errMsg := fmt.Sprintf("Block %d: Bad hash", i)
            result.BadHash = append(result.BadHash, errMsg)
            result.TotalErrors++
        }

        // Duplicate detection
        if firstHeight, exists := seenHashes[block.Hash]; exists {
            errMsg := fmt.Sprintf("Block %d duplicates hash from Block %d", i, firstHeight)
            result.DuplicateHashes = append(result.DuplicateHashes, errMsg)
            result.TotalErrors++
        } else {
            seenHashes[block.Hash] = i
        }

        // Timestamp future
        if block.Timestamp > currentTime+300 {
            errMsg := fmt.Sprintf("Block %d: Timestamp in future", i)
            result.TimestampFuture = append(result.TimestampFuture, errMsg)
            result.TotalErrors++
        }

        // Timestamp past
        tenYearsAgo := currentTime - (10 * 365 * 24 * 60 * 60)
        if block.Timestamp < tenYearsAgo {
            errMsg := fmt.Sprintf("Block %d: Timestamp too old", i)
            result.TimestampPast = append(result.TimestampPast, errMsg)
            result.TotalErrors++
        }

        // Timestamp not increasing
        if prevBlock != nil && block.Timestamp <= prevBlock.Timestamp {
            errMsg := fmt.Sprintf("Block %d: Timestamp not increasing", i)
            result.TimestampNotIncreasing = append(result.TimestampNotIncreasing, errMsg)
            result.TotalErrors++
        }

        // Empty blocks
        if block.Data == "" || len(strings.TrimSpace(block.Data)) == 0 {
            errMsg := fmt.Sprintf("Block %d: Empty block", i)
            result.EmptyBlocks = append(result.EmptyBlocks, errMsg)
            result.TotalErrors++
        }

        // PrevHash validation
        if i == 0 {
            if block.PrevHash != "0" {
                errMsg := fmt.Sprintf("Block 0: Invalid genesis prevHash")
                result.PrevHashErrors = append(result.PrevHashErrors, errMsg)
                result.TotalErrors++
            }
        } else if prevBlock != nil && block.PrevHash != prevBlock.Hash {
            errMsg := fmt.Sprintf("Block %d: PrevHash linkage broken", i)
            result.PrevHashErrors = append(result.PrevHashErrors, errMsg)
            result.TotalErrors++
        }

        // Height validation
        if block.Height != expectedHeight {
            errMsg := fmt.Sprintf("Block %d: Height mismatch", i)
            result.HeightErrors = append(result.HeightErrors, errMsg)
            result.TotalErrors++
        }

        // Out of order
        if block.Height < expectedHeight {
            errMsg := fmt.Sprintf("Block %d: Out of order", i)
            result.OutOfOrderBlocks = append(result.OutOfOrderBlocks, errMsg)
            result.TotalErrors++
        }

        prevBlock = &block
        expectedHeight++
    }

    // Calculate health score
    if result.BlocksScanned > 0 {
        result.HealthScore = ((result.BlocksScanned - result.TotalErrors) * 100) / result.BlocksScanned
        if result.HealthScore < 0 {
            result.HealthScore = 0
        }
    }

    if result.TotalErrors == 0 {
        result.Status = "HEALTHY"
    } else {
        result.Status = "ERRORS_FOUND"
    }

    return result
}
