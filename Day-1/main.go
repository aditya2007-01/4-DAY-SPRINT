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

// ValidationErrors tracks all types of errors found
type ValidationErrors struct {
    HashMismatches      []string
    PrevHashErrors      []string
    MissingBlocks       []int
    DuplicateHashes     []string
    HeightMismatches    []string
    TimestampAnomalies  []string
    OutOfOrderBlocks    []string
    TotalErrors         int
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

// VerifyChainComplete performs comprehensive end-to-end validation
func VerifyChainComplete(db *leveldb.DB) error {
    fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
    fmt.Println("║      COMPLETE END-TO-END BLOCKCHAIN VERIFICATION              ║")
    fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")

    height := GetMaxHeight(db)
    
    if height < 0 {
        fmt.Println("✖ No blocks found in database")
        return fmt.Errorf("empty database")
    }

    fmt.Printf("Starting comprehensive validation of %d blocks...\n\n", height+1)

    // Initialize error tracking
    errors := ValidationErrors{}
    
    // Track seen hashes for duplicate detection
    seenHashes := make(map[string]int)
    
    var prevBlock *Block
    expectedHeight := 0

    // =====================================================
    // COMPREHENSIVE VALIDATION LOOP
    // =====================================================
    for i := 0; i <= height+10; i++ { // Check a few extra heights for gaps
        block, err := LoadBlock(db, i)
        
        // =====================================================
        // 1. DETECT MISSING BLOCKS
        // =====================================================
        if err != nil {
            if i <= height {
                errors.MissingBlocks = append(errors.MissingBlocks, i)
                errors.TotalErrors++
                fmt.Printf("✖ Block %d: MISSING BLOCK\n", i)
            }
            
            // Stop checking beyond reasonable range
            if i > height {
                break
            }
            continue
        }

        // =====================================================
        // 2. VALIDATE HASH == SHA256(blockData)
        // =====================================================
        computedHash := ComputeHash(block.Height, block.PrevHash, block.Data, block.Timestamp)
        if block.Hash != computedHash {
            errMsg := fmt.Sprintf("Block %d: Expected hash %s, got %s", 
                i, computedHash[:16]+"...", block.Hash[:16]+"...")
            errors.HashMismatches = append(errors.HashMismatches, errMsg)
            errors.TotalErrors++
            
            fmt.Printf("✖ Block %d: HASH MISMATCH\n", i)
            fmt.Printf("   Computed: %s\n", computedHash)
            fmt.Printf("   Stored:   %s\n", block.Hash)
        }

        // =====================================================
        // 3. DETECT DUPLICATE HASHES
        // =====================================================
        if firstHeight, exists := seenHashes[block.Hash]; exists {
            errMsg := fmt.Sprintf("Block %d duplicates hash from Block %d (hash: %s...)", 
                i, firstHeight, block.Hash[:16])
            errors.DuplicateHashes = append(errors.DuplicateHashes, errMsg)
            errors.TotalErrors++
            
            fmt.Printf("✖ Block %d: DUPLICATE HASH (also in Block %d)\n", i, firstHeight)
            fmt.Printf("   Hash: %s\n", block.Hash)
        } else {
            seenHashes[block.Hash] = i
        }

        // =====================================================
        // 4. VALIDATE PREVHASH LINK END-TO-END
        // =====================================================
        if i == 0 {
            // Genesis block validation
            if block.PrevHash != "0" {
                errMsg := fmt.Sprintf("Block 0 (genesis): Invalid prevHash '%s', expected '0'", 
                    block.PrevHash)
                errors.PrevHashErrors = append(errors.PrevHashErrors, errMsg)
                errors.TotalErrors++
                
                fmt.Printf("✖ Block 0: GENESIS BLOCK INVALID PREVHASH\n")
                fmt.Printf("   Expected: 0\n")
                fmt.Printf("   Got:      %s\n", block.PrevHash)
            }
        } else {
            // Validate chain linkage
            if prevBlock != nil && block.PrevHash != prevBlock.Hash {
                errMsg := fmt.Sprintf("Block %d: PrevHash mismatch - expected %s, got %s", 
                    i, prevBlock.Hash[:16]+"...", block.PrevHash[:16]+"...")
                errors.PrevHashErrors = append(errors.PrevHashErrors, errMsg)
                errors.TotalErrors++
                
                fmt.Printf("✖ Block %d: PREVHASH LINKAGE BROKEN\n", i)
                fmt.Printf("   Expected (Block %d hash): %s\n", i-1, prevBlock.Hash)
                fmt.Printf("   Got:                      %s\n", block.PrevHash)
            }
        }

        // =====================================================
        // 5. DETECT HEIGHT MISMATCHES
        // =====================================================
        if block.Height != expectedHeight {
            errMsg := fmt.Sprintf("Block at position %d has height %d (mismatch)", 
                i, block.Height)
            errors.HeightMismatches = append(errors.HeightMismatches, errMsg)
            errors.TotalErrors++
            
            fmt.Printf("✖ Block %d: HEIGHT MISMATCH\n", i)
            fmt.Printf("   Expected height: %d\n", expectedHeight)
            fmt.Printf("   Stored height:   %d\n", block.Height)
        }

        // =====================================================
        // 6. DETECT TIMESTAMP ANOMALIES
        // =====================================================
        if prevBlock != nil {
            // Check timestamps are strictly increasing
            if block.Timestamp <= prevBlock.Timestamp {
                errMsg := fmt.Sprintf("Block %d: Timestamp not increasing (%d <= %d)", 
                    i, block.Timestamp, prevBlock.Timestamp)
                errors.TimestampAnomalies = append(errors.TimestampAnomalies, errMsg)
                errors.TotalErrors++
                
                fmt.Printf("✖ Block %d: TIMESTAMP NOT INCREASING\n", i)
                fmt.Printf("   Block %d time: %s (Unix: %d)\n", 
                    i-1, 
                    time.Unix(prevBlock.Timestamp, 0).Format("2006-01-02 15:04:05"), 
                    prevBlock.Timestamp)
                fmt.Printf("   Block %d time: %s (Unix: %d)\n", 
                    i, 
                    time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"), 
                    block.Timestamp)
            }
            
            // Detect unrealistic timestamp jumps (e.g., > 1 year)
            timeDiff := block.Timestamp - prevBlock.Timestamp
            if timeDiff > 31536000 { // 1 year in seconds
                errMsg := fmt.Sprintf("Block %d: Suspicious timestamp jump of %d seconds", 
                    i, timeDiff)
                errors.TimestampAnomalies = append(errors.TimestampAnomalies, errMsg)
                errors.TotalErrors++
                
                fmt.Printf("⚠ Block %d: SUSPICIOUS TIMESTAMP JUMP\n", i)
                fmt.Printf("   Time difference: %d seconds (%.1f days)\n", 
                    timeDiff, float64(timeDiff)/86400)
            }
        }

        // =====================================================
        // 7. DETECT OUT-OF-ORDER BLOCKS
        // =====================================================
        if block.Height < expectedHeight {
            errMsg := fmt.Sprintf("Block %d appears out of order (height %d < expected %d)", 
                i, block.Height, expectedHeight)
            errors.OutOfOrderBlocks = append(errors.OutOfOrderBlocks, errMsg)
            errors.TotalErrors++
            
            fmt.Printf("✖ Block %d: OUT OF ORDER\n", i)
            fmt.Printf("   Block height %d found at position %d\n", block.Height, i)
        }

        // Print OK if no errors for this block
        if block.Hash == computedHash && 
           (i == 0 || (prevBlock != nil && block.PrevHash == prevBlock.Hash)) &&
           block.Height == expectedHeight &&
           (prevBlock == nil || block.Timestamp > prevBlock.Timestamp) {
            fmt.Printf("✔ Block %d: OK (hash: %s...)\n", i, block.Hash[:16])
        }

        prevBlock = block
        expectedHeight++
    }

    // =====================================================
    // DETAILED ERROR SUMMARY
    // =====================================================
    fmt.Println("\n" + strings.Repeat("═", 66))
    fmt.Println("COMPREHENSIVE VALIDATION SUMMARY")
    fmt.Println(strings.Repeat("═", 66))
    
    fmt.Printf("\n📊 BLOCKS ANALYZED: %d\n", height+1)
    
    fmt.Println("\n🔍 VALIDATION RESULTS:")
    fmt.Println(strings.Repeat("-", 66))
    
    // Hash validation
    if len(errors.HashMismatches) == 0 {
        fmt.Println("✔ Hash Validation:              PASSED (0 errors)")
    } else {
        fmt.Printf("✖ Hash Validation:              FAILED (%d errors)\n", len(errors.HashMismatches))
        for _, err := range errors.HashMismatches {
            fmt.Printf("   • %s\n", err)
        }
    }
    
    // PrevHash validation
    if len(errors.PrevHashErrors) == 0 {
        fmt.Println("✔ PrevHash Linkage:             PASSED (0 errors)")
    } else {
        fmt.Printf("✖ PrevHash Linkage:             FAILED (%d errors)\n", len(errors.PrevHashErrors))
        for _, err := range errors.PrevHashErrors {
            fmt.Printf("   • %s\n", err)
        }
    }
    
    // Missing blocks
    if len(errors.MissingBlocks) == 0 {
        fmt.Println("✔ Missing Block Detection:      PASSED (0 missing)")
    } else {
        fmt.Printf("✖ Missing Block Detection:      FAILED (%d missing)\n", len(errors.MissingBlocks))
        fmt.Printf("   • Missing heights: %v\n", errors.MissingBlocks)
    }
    
    // Duplicate hashes
    if len(errors.DuplicateHashes) == 0 {
        fmt.Println("✔ Duplicate Hash Detection:     PASSED (0 duplicates)")
    } else {
        fmt.Printf("✖ Duplicate Hash Detection:     FAILED (%d duplicates)\n", len(errors.DuplicateHashes))
        for _, err := range errors.DuplicateHashes {
            fmt.Printf("   • %s\n", err)
        }
    }
    
    // Height validation
    if len(errors.HeightMismatches) == 0 {
        fmt.Println("✔ Height Validation:            PASSED (0 mismatches)")
    } else {
        fmt.Printf("✖ Height Validation:            FAILED (%d mismatches)\n", len(errors.HeightMismatches))
        for _, err := range errors.HeightMismatches {
            fmt.Printf("   • %s\n", err)
        }
    }
    
    // Timestamp validation
    if len(errors.TimestampAnomalies) == 0 {
        fmt.Println("✔ Timestamp Validation:         PASSED (0 anomalies)")
    } else {
        fmt.Printf("✖ Timestamp Validation:         FAILED (%d anomalies)\n", len(errors.TimestampAnomalies))
        for _, err := range errors.TimestampAnomalies {
            fmt.Printf("   • %s\n", err)
        }
    }
    
    // Out-of-order detection
    if len(errors.OutOfOrderBlocks) == 0 {
        fmt.Println("✔ Block Order Validation:       PASSED (0 out-of-order)")
    } else {
        fmt.Printf("✖ Block Order Validation:       FAILED (%d out-of-order)\n", len(errors.OutOfOrderBlocks))
        for _, err := range errors.OutOfOrderBlocks {
            fmt.Printf("   • %s\n", err)
        }
    }
    
    fmt.Println(strings.Repeat("-", 66))
    fmt.Printf("\n📈 TOTAL ERRORS FOUND: %d\n", errors.TotalErrors)

    // Final verdict
    if errors.TotalErrors == 0 {
        fmt.Println("\n" + strings.Repeat("═", 66))
        fmt.Println("🎉 BLOCKCHAIN VERIFICATION PASSED!")
        fmt.Println("   All blocks are valid and properly linked.")
        fmt.Println("   Chain integrity: 100%")
        fmt.Println(strings.Repeat("═", 66))
        return nil
    } else {
        fmt.Println("\n" + strings.Repeat("═", 66))
        fmt.Println("⚠️  BLOCKCHAIN VERIFICATION FAILED!")
        fmt.Printf("   Found %d integrity issues across %d blocks.\n", errors.TotalErrors, height+1)
        fmt.Printf("   Chain integrity: %.1f%%\n", 
            float64(height+1-errors.TotalErrors)*100/float64(height+1))
        fmt.Println(strings.Repeat("═", 66))
        return fmt.Errorf("verification failed with %d errors", errors.TotalErrors)
    }
}

// ViewBlock displays details of a specific block
func ViewBlock(db *leveldb.DB, height int) {
    block, err := LoadBlock(db, height)
    if err != nil {
        fmt.Printf("Error loading block %d: %v\n", height, err)
        return
    }

    fmt.Printf("\n=== Block %d ===\n", block.Height)
    fmt.Printf("Hash:      %s\n", block.Hash)
    fmt.Printf("PrevHash:  %s\n", block.PrevHash)
    fmt.Printf("Timestamp: %s (Unix: %d)\n", 
        time.Unix(block.Timestamp, 0).UTC(), block.Timestamp)
    fmt.Printf("Data:      %s\n\n", block.Data)
}

// GetBlockchainStats displays blockchain statistics
func GetBlockchainStats(db *leveldb.DB) {
    height := 0
    blockCount := 0
    seenHashes := make(map[string]int)
    duplicates := []string{}
    missingHeights := []int{}
    var totalTimeDiff int64 = 0
    var prevTimestamp int64 = 0
    var latestHeight int = -1
    firstBlock := true

    fmt.Println("\n=== Blockchain Stats ===\n")
    fmt.Println("Scanning blocks...")

    for {
        block, err := LoadBlock(db, height)
        if err != nil {
            if blockCount > 0 && height < latestHeight+10 {
                missingHeights = append(missingHeights, height)
                height++
                continue
            }
            break
        }

        if firstHeight, exists := seenHashes[block.Hash]; exists {
            duplicates = append(duplicates,
                fmt.Sprintf("Block %d duplicates hash from Block %d", block.Height, firstHeight))
        } else {
            seenHashes[block.Hash] = block.Height
        }

        if !firstBlock {
            totalTimeDiff += (block.Timestamp - prevTimestamp)
        }
        prevTimestamp = block.Timestamp
        latestHeight = block.Height
        firstBlock = false

        blockCount++
        height++
    }

    avgBlockTime := float64(0)
    if blockCount > 1 {
        avgBlockTime = float64(totalTimeDiff) / float64(blockCount-1)
    }

    fmt.Println("\n--- Results ---")
    fmt.Printf("Height: %d\n", latestHeight)
    fmt.Printf("Total Blocks: %d\n", blockCount)
    fmt.Printf("Average Block Time: %.2f seconds\n", avgBlockTime)

    fmt.Println("\n--- Gap Detection ---")
    if len(missingHeights) > 0 {
        fmt.Printf("⚠ Gaps detected at heights: %v\n", missingHeights)
    } else {
        fmt.Println("✔ No gaps detected")
    }

    fmt.Println("\n--- Duplicate Hash Detection ---")
    if len(duplicates) > 0 {
        for _, dup := range duplicates {
            fmt.Printf("⚠ %s\n", dup)
        }
    } else {
        fmt.Println("✔ No duplicate hashes detected")
    }
    fmt.Println()
}

func main() {
    dbPath := flag.String("db", "./leveldb-data", "Path to LevelDB database")
    cmd := flag.String("cmd", "verify", "Command: load, view, stats, verify")
    numBlocks := flag.Int("blocks", 10, "Number of blocks to load")
    flag.Parse()

    switch *cmd {
    case "load":
        LoadSampleData(*dbPath, *numBlocks)

    case "view":
        db, err := leveldb.OpenFile(*dbPath, nil)
        if err != nil {
            fmt.Printf("Failed to open database: %v\n", err)
            return
        }
        defer db.Close()

        if flag.NArg() < 1 {
            fmt.Println("Usage: -cmd view <block_height>")
            return
        }
        height, err := strconv.Atoi(flag.Arg(0))
        if err != nil {
            fmt.Println("Invalid block height")
            return
        }
        ViewBlock(db, height)

    case "stats":
        db, err := leveldb.OpenFile(*dbPath, nil)
        if err != nil {
            fmt.Printf("Failed to open database: %v\n", err)
            return
        }
        defer db.Close()
        GetBlockchainStats(db)

    case "verify":
        db, err := leveldb.OpenFile(*dbPath, nil)
        if err != nil {
            fmt.Printf("Failed to open database: %v\n", err)
            return
        }
        defer db.Close()
        VerifyChainComplete(db)

    default:
        fmt.Printf("Unknown command: %s\n", *cmd)
        fmt.Println("\nAvailable commands:")
        fmt.Println("  load   - Load sample blockchain data")
        fmt.Println("  view   - View a specific block")
        fmt.Println("  stats  - Display blockchain statistics")
        fmt.Println("  verify - Complete chain verification")
        fmt.Println("\nExamples:")
        fmt.Println("  go run main.go -cmd load -db ./leveldb-data -blocks 50")
        fmt.Println("  go run main.go -cmd view -db ./leveldb-data 5")
        fmt.Println("  go run main.go -cmd stats -db ./leveldb-data")
        fmt.Println("  go run main.go -cmd verify -db ./leveldb-data")
    }
}
