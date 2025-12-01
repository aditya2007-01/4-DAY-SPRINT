package main

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "flag"
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/syndtr/goleveldb/leveldb"
)

// Block represents a blockchain block
type Block struct {
    Height    int    `json:"height"`
    Hash      string `json:"hash"`
    PrevHash  string `json:"prev_hash"`
    Data      string `json:"data"`
    Timestamp int64  `json:"timestamp"`
}

// ErrorScanResult contains all detected errors with classification
type ErrorScanResult struct {
    ScanTime            string   `json:"scan_time"`
    DatabasePath        string   `json:"database_path"`
    TotalBlocks         int      `json:"total_blocks"`
    BlocksScanned       int      `json:"blocks_scanned"`
    TotalErrors         int      `json:"total_errors"`
    
    // Error classification
    CorruptedJSON       []string `json:"corrupted_json"`
    BadHash             []string `json:"bad_hash"`
    TimestampFuture     []string `json:"timestamp_future"`
    TimestampPast       []string `json:"timestamp_past"`
    TimestampNotIncreasing []string `json:"timestamp_not_increasing"`
    DuplicateHashes     []string `json:"duplicate_hashes"`
    EmptyBlocks         []string `json:"empty_blocks"`
    
    // Additional errors
    PrevHashErrors      []string `json:"prevhash_errors"`
    HeightErrors        []string `json:"height_errors"`
    MissingBlocks       []int    `json:"missing_blocks"`
    OutOfOrderBlocks    []string `json:"out_of_order_blocks"`
    
    HealthScore         int      `json:"health_score"`
    Status              string   `json:"status"`
}

// ComparisonResult tracks differences between two nodes
type ComparisonResult struct {
    ScanTime            string   `json:"scan_time"`
    Node1Path           string   `json:"node1_path"`
    Node2Path           string   `json:"node2_path"`
    Node1Height         int      `json:"node1_height"`
    Node2Height         int      `json:"node2_height"`
    MatchingBlocks      int      `json:"matching_blocks"`
    MismatchedBlocks    []int    `json:"mismatched_blocks"`
    Node1OnlyBlocks     []int    `json:"node1_only_blocks"`
    Node2OnlyBlocks     []int    `json:"node2_only_blocks"`
    DivergencePoint     int      `json:"divergence_point"`
    HashMismatches      []string `json:"hash_mismatches"`
    DataMismatches      []string `json:"data_mismatches"`
    TimestampMismatches []string `json:"timestamp_mismatches"`
    SyncPercentage      float64  `json:"sync_percentage"`
    Recommendations     []string `json:"recommendations"`
}

// ComputeHash calculates SHA256 hash for a block
func ComputeHash(height int, prevHash string, data string, timestamp int64) string {
    record := strconv.Itoa(height) + prevHash + data + strconv.FormatInt(timestamp, 10)
    h := sha256.New()
    h.Write([]byte(record))
    hashed := h.Sum(nil)
    return hex.EncodeToString(hashed)
}

// LoadBlock retrieves a single block from the database
func LoadBlock(db *leveldb.DB, height int) (*Block, error) {
    key := []byte(fmt.Sprintf("block-%d", height))
    data, err := db.Get(key, nil)
    if err != nil {
        return nil, err
    }

    var block Block
    if err := json.Unmarshal(data, &block); err != nil {
        return nil, err
    }
    return &block, nil
}

// LoadBlockRaw retrieves raw block data (for JSON corruption detection)
func LoadBlockRaw(db *leveldb.DB, height int) ([]byte, error) {
    key := []byte(fmt.Sprintf("block-%d", height))
    return db.Get(key, nil)
}

// GetMaxHeight finds the highest block height in a database
func GetMaxHeight(db *leveldb.DB) int {
    height := 0
    for {
        _, err := LoadBlock(db, height)
        if err != nil {
            if height == 0 {
                return -1
            }
            return height - 1
        }
        height++
    }
}

// LoadSampleData loads sample blocks into the database
func LoadSampleData(dbPath string, numBlocks int) {
    db, err := leveldb.OpenFile(dbPath, nil)
    if err != nil {
        fmt.Printf("Failed to open database: %v\n", err)
        return
    }
    defer db.Close()

    fmt.Printf("Loading %d sample blocks into %s...\n", numBlocks, dbPath)

    prevHash := "0"

    for i := 0; i < numBlocks; i++ {
        timestamp := time.Now().Unix() + int64(i*10)
        data := fmt.Sprintf("Transaction data for block %d", i)
        hash := ComputeHash(i, prevHash, data, timestamp)

        block := Block{
            Height:    i,
            Hash:      hash,
            PrevHash:  prevHash,
            Data:      data,
            Timestamp: timestamp,
        }

        blockJSON, _ := json.Marshal(block)
        key := []byte(fmt.Sprintf("block-%d", i))
        db.Put(key, blockJSON, nil)

        fmt.Printf("✔ Block %d stored (hash: %s...)\n", i, hash[:16])
        prevHash = hash
    }

    fmt.Println("\nData loading complete!")
}

// ScanErrors performs comprehensive error scanning with classification
func ScanErrors(db *leveldb.DB, dbPath string, jsonOutput bool) {
    result := ErrorScanResult{
        ScanTime:     time.Now().Format("2006-01-02 15:04:05"),
        DatabasePath: dbPath,
    }

    height := GetMaxHeight(db)
    
    if height < 0 {
        result.Status = "ERROR: Empty database"
        result.HealthScore = 0
        outputResult(result, jsonOutput)
        return
    }

    result.TotalBlocks = height + 1

    if !jsonOutput {
        fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
        fmt.Println("║          BLOCKCHAIN ERROR SCANNER WITH CLASSIFICATION         ║")
        fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
        fmt.Printf("Database: %s\n", dbPath)
        fmt.Printf("Scan Time: %s\n", result.ScanTime)
        fmt.Printf("Total Blocks: %d\n\n", result.TotalBlocks)
        fmt.Println("Scanning for errors...")
        fmt.Println(strings.Repeat("─", 66))
    }

    seenHashes := make(map[string]int)
    var prevBlock *Block
    expectedHeight := 0
    currentTime := time.Now().Unix()

    // Scan all blocks
    for i := 0; i <= height+10; i++ {
        // Try to load raw data first for JSON corruption detection
        rawData, rawErr := LoadBlockRaw(db, i)
        
        if rawErr != nil {
            if i <= height {
                result.MissingBlocks = append(result.MissingBlocks, i)
                result.TotalErrors++
                if !jsonOutput {
                    fmt.Printf("✖ Block %d: MISSING\n", i)
                }
            }
            if i > height {
                break
            }
            continue
        }

        // =====================================================
        // 1. CORRUPTED JSON DETECTION
        // =====================================================
        var block Block
        err := json.Unmarshal(rawData, &block)
        if err != nil {
            errMsg := fmt.Sprintf("Block %d: Corrupted JSON - %v", i, err)
            result.CorruptedJSON = append(result.CorruptedJSON, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: CORRUPTED JSON\n", i)
                fmt.Printf("  └─ Error: %v\n", err)
            }
            continue
        }

        result.BlocksScanned++

        // =====================================================
        // 2. BAD HASH DETECTION
        // =====================================================
        computedHash := ComputeHash(block.Height, block.PrevHash, block.Data, block.Timestamp)
        if block.Hash != computedHash {
            errMsg := fmt.Sprintf("Block %d: Bad hash (expected: %s..., got: %s...)", 
                i, computedHash[:16], block.Hash[:16])
            result.BadHash = append(result.BadHash, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: BAD HASH\n", i)
                fmt.Printf("  ├─ Expected: %s\n", computedHash)
                fmt.Printf("  └─ Got:      %s\n", block.Hash)
            }
        }

        // =====================================================
        // 3. DUPLICATE HASH DETECTION
        // =====================================================
        if firstHeight, exists := seenHashes[block.Hash]; exists {
            errMsg := fmt.Sprintf("Block %d duplicates hash from Block %d (hash: %s...)", 
                i, firstHeight, block.Hash[:16])
            result.DuplicateHashes = append(result.DuplicateHashes, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: DUPLICATE HASH (also in Block %d)\n", i, firstHeight)
                fmt.Printf("  └─ Hash: %s\n", block.Hash)
            }
        } else {
            seenHashes[block.Hash] = i
        }

        // =====================================================
        // 4. TIMESTAMP FUTURE DETECTION
        // =====================================================
        // Allow 5 minute clock drift tolerance
        if block.Timestamp > currentTime+300 {
            timeDiff := block.Timestamp - currentTime
            errMsg := fmt.Sprintf("Block %d: Timestamp in future by %d seconds (%s)", 
                i, timeDiff, time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"))
            result.TimestampFuture = append(result.TimestampFuture, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: TIMESTAMP IN FUTURE\n", i)
                fmt.Printf("  ├─ Block time: %s (Unix: %d)\n", 
                    time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"),
                    block.Timestamp)
                fmt.Printf("  └─ Current:    %s (Unix: %d)\n", 
                    time.Unix(currentTime, 0).Format("2006-01-02 15:04:05"),
                    currentTime)
            }
        }

        // =====================================================
        // 5. TIMESTAMP TOO FAR IN PAST DETECTION
        // =====================================================
        // Flag timestamps older than 10 years as suspicious
        tenYearsAgo := currentTime - (10 * 365 * 24 * 60 * 60)
        if block.Timestamp < tenYearsAgo {
            errMsg := fmt.Sprintf("Block %d: Timestamp too far in past (%s)", 
                i, time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"))
            result.TimestampPast = append(result.TimestampPast, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: TIMESTAMP TOO OLD\n", i)
                fmt.Printf("  └─ Time: %s (Unix: %d)\n", 
                    time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"),
                    block.Timestamp)
            }
        }

        // =====================================================
        // 6. TIMESTAMP NOT INCREASING DETECTION
        // =====================================================
        if prevBlock != nil {
            if block.Timestamp <= prevBlock.Timestamp {
                errMsg := fmt.Sprintf("Block %d: Timestamp not increasing (%d <= %d)", 
                    i, block.Timestamp, prevBlock.Timestamp)
                result.TimestampNotIncreasing = append(result.TimestampNotIncreasing, errMsg)
                result.TotalErrors++
                if !jsonOutput {
                    fmt.Printf("✖ Block %d: TIMESTAMP NOT INCREASING\n", i)
                    fmt.Printf("  ├─ Block %d: %s (Unix: %d)\n", 
                        i-1,
                        time.Unix(prevBlock.Timestamp, 0).Format("2006-01-02 15:04:05"),
                        prevBlock.Timestamp)
                    fmt.Printf("  └─ Block %d: %s (Unix: %d)\n", 
                        i,
                        time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"),
                        block.Timestamp)
                }
            }
        }

        // =====================================================
        // 7. EMPTY BLOCK DETECTION
        // =====================================================
        if block.Data == "" || len(strings.TrimSpace(block.Data)) == 0 {
            errMsg := fmt.Sprintf("Block %d: Empty block (no data)", i)
            result.EmptyBlocks = append(result.EmptyBlocks, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("⚠ Block %d: EMPTY BLOCK\n", i)
            }
        }

        // =====================================================
        // 8. PREVHASH VALIDATION
        // =====================================================
        if i == 0 {
            if block.PrevHash != "0" {
                errMsg := fmt.Sprintf("Block 0: Invalid genesis prevHash '%s'", block.PrevHash)
                result.PrevHashErrors = append(result.PrevHashErrors, errMsg)
                result.TotalErrors++
                if !jsonOutput {
                    fmt.Printf("✖ Block 0: INVALID GENESIS PREVHASH\n")
                    fmt.Printf("  └─ Expected: 0, Got: %s\n", block.PrevHash)
                }
            }
        } else {
            if prevBlock != nil && block.PrevHash != prevBlock.Hash {
                errMsg := fmt.Sprintf("Block %d: PrevHash linkage broken", i)
                result.PrevHashErrors = append(result.PrevHashErrors, errMsg)
                result.TotalErrors++
                if !jsonOutput {
                    fmt.Printf("✖ Block %d: PREVHASH LINKAGE BROKEN\n", i)
                }
            }
        }

        // =====================================================
        // 9. HEIGHT VALIDATION
        // =====================================================
        if block.Height != expectedHeight {
            errMsg := fmt.Sprintf("Block %d: Height mismatch (expected: %d, got: %d)", 
                i, expectedHeight, block.Height)
            result.HeightErrors = append(result.HeightErrors, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: HEIGHT MISMATCH\n", i)
            }
        }

        // =====================================================
        // 10. OUT OF ORDER DETECTION
        // =====================================================
        if block.Height < expectedHeight {
            errMsg := fmt.Sprintf("Block %d: Out of order (height %d < expected %d)", 
                i, block.Height, expectedHeight)
            result.OutOfOrderBlocks = append(result.OutOfOrderBlocks, errMsg)
            result.TotalErrors++
            if !jsonOutput {
                fmt.Printf("✖ Block %d: OUT OF ORDER\n", i)
            }
        }

        // Print OK if no errors
        if !jsonOutput {
            hasErrors := false
            if len(result.BadHash) > 0 && result.BadHash[len(result.BadHash)-1] == fmt.Sprintf("Block %d: Bad hash (expected: %s..., got: %s...)", i, computedHash[:16], block.Hash[:16]) {
                hasErrors = true
            }
            // Check other recent errors...
            if !hasErrors {
                fmt.Printf("✔ Block %d: OK\n", i)
            }
        }

        prevBlock = &block
        expectedHeight++
    }

    // Calculate health score
    if result.BlocksScanned > 0 {
        errorWeight := result.TotalErrors
        result.HealthScore = ((result.BlocksScanned - errorWeight) * 100) / result.BlocksScanned
        if result.HealthScore < 0 {
            result.HealthScore = 0
        }
    }

    if result.TotalErrors == 0 {
        result.Status = "HEALTHY"
    } else {
        result.Status = "ERRORS_FOUND"
    }

    if !jsonOutput {
        fmt.Println(strings.Repeat("─", 66))
    }

    outputResult(result, jsonOutput)
}

// CompareNodes performs comprehensive comparison between two blockchain nodes
func CompareNodes(db1, db2 *leveldb.DB, db1Path, db2Path string, jsonOutput bool) {
    result := ComparisonResult{
        ScanTime:        time.Now().Format("2006-01-02 15:04:05"),
        Node1Path:       db1Path,
        Node2Path:       db2Path,
        DivergencePoint: -1,
    }

    result.Node1Height = GetMaxHeight(db1)
    result.Node2Height = GetMaxHeight(db2)

    if !jsonOutput {
        fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
        fmt.Println("║        BLOCKCHAIN NODE COMPARISON & DIFF ANALYSIS             ║")
        fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")
        fmt.Printf("Node 1: %s (Height: %d)\n", db1Path, result.Node1Height)
        fmt.Printf("Node 2: %s (Height: %d)\n\n", db2Path, result.Node2Height)
        fmt.Println("Comparing blocks...")
        fmt.Println(strings.Repeat("─", 66))
    }

    maxHeight := result.Node1Height
    if result.Node2Height > maxHeight {
        maxHeight = result.Node2Height
    }

    for i := 0; i <= maxHeight; i++ {
        block1, err1 := LoadBlock(db1, i)
        block2, err2 := LoadBlock(db2, i)

        if err1 != nil && err2 != nil {
            continue
        }

        if err1 != nil && err2 == nil {
            result.Node2OnlyBlocks = append(result.Node2OnlyBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            if !jsonOutput {
                fmt.Printf("✖ Block %d: Missing on Node1\n", i)
            }
            continue
        }

        if err1 == nil && err2 != nil {
            result.Node1OnlyBlocks = append(result.Node1OnlyBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            if !jsonOutput {
                fmt.Printf("✖ Block %d: Missing on Node2\n", i)
            }
            continue
        }

        if block1.Hash != block2.Hash {
            result.MismatchedBlocks = append(result.MismatchedBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            errMsg := fmt.Sprintf("Block %d: Hash mismatch", i)
            result.HashMismatches = append(result.HashMismatches, errMsg)
            if !jsonOutput {
                fmt.Printf("✖ Block %d: Hash mismatch\n", i)
            }
        } else {
            result.MatchingBlocks++
            if !jsonOutput {
                fmt.Printf("✔ Block %d: Match\n", i)
            }
        }

        if block1.Data != block2.Data {
            errMsg := fmt.Sprintf("Block %d: Data differs", i)
            result.DataMismatches = append(result.DataMismatches, errMsg)
        }

        if block1.Timestamp != block2.Timestamp {
            errMsg := fmt.Sprintf("Block %d: Timestamp differs by %d seconds", 
                i, block1.Timestamp-block2.Timestamp)
            result.TimestampMismatches = append(result.TimestampMismatches, errMsg)
        }
    }

    if maxHeight >= 0 {
        result.SyncPercentage = (float64(result.MatchingBlocks) / float64(maxHeight+1)) * 100
    }

    result.Recommendations = generateComparisonRecommendations(result)

    if !jsonOutput {
        fmt.Println(strings.Repeat("─", 66))
    }

    outputComparisonResult(result, jsonOutput)
}

func generateComparisonRecommendations(result ComparisonResult) []string {
    recs := []string{}

    heightDiff := result.Node1Height - result.Node2Height

    if heightDiff > 0 {
        recs = append(recs, 
            fmt.Sprintf("Node2 is %d blocks behind - sync from Node1", heightDiff))
    } else if heightDiff < 0 {
        recs = append(recs, 
            fmt.Sprintf("Node1 is %d blocks behind - sync from Node2", -heightDiff))
    }

    if result.DivergencePoint >= 0 {
        recs = append(recs, 
            fmt.Sprintf("Chains diverge at block %d", result.DivergencePoint))
    }

    if len(result.HashMismatches) > 0 {
        recs = append(recs, 
            fmt.Sprintf("Found %d hash mismatches", len(result.HashMismatches)))
    }

    if len(recs) == 0 {
        recs = append(recs, "Nodes are perfectly synchronized")
    }

    return recs
}

func outputResult(result ErrorScanResult, jsonOutput bool) {
    if jsonOutput {
        jsonData, _ := json.MarshalIndent(result, "", "  ")
        fmt.Println(string(jsonData))
    } else {
        fmt.Println("\n" + strings.Repeat("═", 66))
        fmt.Println("SCAN SUMMARY")
        fmt.Println(strings.Repeat("═", 66))
        fmt.Printf("\n📊 STATISTICS:\n")
        fmt.Printf("  Blocks Scanned:   %d\n", result.BlocksScanned)
        fmt.Printf("  Total Errors:     %d\n", result.TotalErrors)
        fmt.Printf("  Health Score:     %d%%\n", result.HealthScore)
        fmt.Printf("  Status:           %s\n", result.Status)
        
        fmt.Println("\n🔍 ERROR CLASSIFICATION:")
        fmt.Printf("  Corrupted JSON:           %d\n", len(result.CorruptedJSON))
        fmt.Printf("  Bad Hash:                 %d\n", len(result.BadHash))
        fmt.Printf("  Timestamp Future:         %d\n", len(result.TimestampFuture))
        fmt.Printf("  Timestamp Past:           %d\n", len(result.TimestampPast))
        fmt.Printf("  Timestamp Not Increasing: %d\n", len(result.TimestampNotIncreasing))
        fmt.Printf("  Duplicate Hashes:         %d\n", len(result.DuplicateHashes))
        fmt.Printf("  Empty Blocks:             %d\n", len(result.EmptyBlocks))
        fmt.Printf("  PrevHash Errors:          %d\n", len(result.PrevHashErrors))
        fmt.Printf("  Height Errors:            %d\n", len(result.HeightErrors))
        fmt.Printf("  Missing Blocks:           %d\n", len(result.MissingBlocks))
        fmt.Printf("  Out of Order:             %d\n", len(result.OutOfOrderBlocks))
        
        if result.TotalErrors == 0 {
            fmt.Println("\n🎉 No errors found! Blockchain is healthy.")
        } else {
            fmt.Println("\n⚠️  Errors detected. Review details above.")
        }
        fmt.Println(strings.Repeat("═", 66))
    }
}

func outputComparisonResult(result ComparisonResult, jsonOutput bool) {
    if jsonOutput {
        jsonData, _ := json.MarshalIndent(result, "", "  ")
        fmt.Println(string(jsonData))
    } else {
        fmt.Println("\n" + strings.Repeat("═", 66))
        fmt.Println("COMPARISON SUMMARY")
        fmt.Println(strings.Repeat("═", 66))
        fmt.Printf("\n📊 NODE INFO:\n")
        fmt.Printf("  Node1: %s (Height: %d)\n", result.Node1Path, result.Node1Height)
        fmt.Printf("  Node2: %s (Height: %d)\n", result.Node2Path, result.Node2Height)
        
        fmt.Println("\n🔍 RESULTS:")
        fmt.Printf("  Matching Blocks:    %d\n", result.MatchingBlocks)
        fmt.Printf("  Mismatched Blocks:  %d\n", len(result.MismatchedBlocks))
        fmt.Printf("  Node1-only:         %d\n", len(result.Node1OnlyBlocks))
        fmt.Printf("  Node2-only:         %d\n", len(result.Node2OnlyBlocks))
        fmt.Printf("  Sync Percentage:    %.1f%%\n", result.SyncPercentage)
        
        if result.DivergencePoint >= 0 {
            fmt.Printf("\n🔀 Divergence Point: Block %d\n", result.DivergencePoint)
        }
        
        fmt.Println("\n🔧 RECOMMENDATIONS:")
        for i, rec := range result.Recommendations {
            fmt.Printf("  %d. %s\n", i+1, rec)
        }
        fmt.Println(strings.Repeat("═", 66))
    }
}

func main() {
    dbPath := flag.String("db", "./leveldb-data", "Path to LevelDB database")
    db1Path := flag.String("db1", "./node1-data", "Path to first database")
    db2Path := flag.String("db2", "./node2-data", "Path to second database")
    cmd := flag.String("cmd", "scan-errors", "Command: load, scan-errors, compare")
    numBlocks := flag.Int("blocks", 10, "Number of blocks to load")
    jsonOutput := flag.Bool("json", false, "Output in JSON format")
    flag.Parse()

    switch *cmd {
    case "load":
        LoadSampleData(*dbPath, *numBlocks)

    case "scan-errors":
        db, err := leveldb.OpenFile(*dbPath, nil)
        if err != nil {
            fmt.Printf("Failed to open database: %v\n", err)
            return
        }
        defer db.Close()
        ScanErrors(db, *dbPath, *jsonOutput)

    case "compare":
        db1, err1 := leveldb.OpenFile(*db1Path, nil)
        if err1 != nil {
            fmt.Printf("Failed to open Node1: %v\n", err1)
            return
        }
        defer db1.Close()

        db2, err2 := leveldb.OpenFile(*db2Path, nil)
        if err2 != nil {
            fmt.Printf("Failed to open Node2: %v\n", err2)
            return
        }
        defer db2.Close()

        CompareNodes(db1, db2, *db1Path, *db2Path, *jsonOutput)

    default:
        fmt.Printf("Unknown command: %s\n", *cmd)
        fmt.Println("\nAvailable commands:")
        fmt.Println("  load        - Load sample blockchain data")
        fmt.Println("  scan-errors - Scan blockchain for errors with classification")
        fmt.Println("  compare     - Compare two blockchain nodes")
        fmt.Println("\nExamples:")
        fmt.Println("  go run main.go -cmd load -db ./leveldb-data -blocks 50")
        fmt.Println("  go run main.go -cmd scan-errors -db ./leveldb-data")
        fmt.Println("  go run main.go -cmd scan-errors -db ./leveldb-data --json")
        fmt.Println("  go run main.go -cmd compare -db1 ./node1-data -db2 ./node2-data")
        fmt.Println("  go run main.go -cmd compare -db1 ./node1-data -db2 ./node2-data --json")
    }
}
