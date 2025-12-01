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

// ComparisonResult tracks differences between two nodes
type ComparisonResult struct {
    Node1Path           string
    Node2Path           string
    Node1Height         int
    Node2Height         int
    MatchingBlocks      int
    MismatchedBlocks    []int
    Node1OnlyBlocks     []int
    Node2OnlyBlocks     []int
    DivergencePoint     int
    HashMismatches      []string
    DataMismatches      []string
    TimestampMismatches []string
    Recommendations     []string
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

// CompareNodes performs comprehensive comparison between two blockchain nodes
func CompareNodes(db1, db2 *leveldb.DB, db1Path, db2Path string) {
    fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
    fmt.Println("║        BLOCKCHAIN NODE COMPARISON & DIFF ANALYSIS             ║")
    fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")

    result := ComparisonResult{
        Node1Path:       db1Path,
        Node2Path:       db2Path,
        DivergencePoint: -1,
    }

    // Get chain heights
    result.Node1Height = GetMaxHeight(db1)
    result.Node2Height = GetMaxHeight(db2)

    fmt.Printf("Node 1: %s\n", db1Path)
    fmt.Printf("  └─ Height: %d blocks\n\n", result.Node1Height+1)
    
    fmt.Printf("Node 2: %s\n", db2Path)
    fmt.Printf("  └─ Height: %d blocks\n\n", result.Node2Height+1)

    maxHeight := result.Node1Height
    if result.Node2Height > maxHeight {
        maxHeight = result.Node2Height
    }

    fmt.Println("Starting block-by-block comparison...\n")
    fmt.Println(strings.Repeat("─", 66))

    // =====================================================
    // BLOCK-BY-BLOCK COMPARISON
    // =====================================================
    for i := 0; i <= maxHeight; i++ {
        block1, err1 := LoadBlock(db1, i)
        block2, err2 := LoadBlock(db2, i)

        // Both blocks missing
        if err1 != nil && err2 != nil {
            fmt.Printf("⚠ Block %d: MISSING on BOTH nodes\n", i)
            continue
        }

        // Block only on Node 2
        if err1 != nil && err2 == nil {
            result.Node2OnlyBlocks = append(result.Node2OnlyBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            fmt.Printf("✖ Block %d: MISSING on Node1 (exists on Node2)\n", i)
            fmt.Printf("  └─ Node2 hash: %s...\n", block2.Hash[:16])
            continue
        }

        // Block only on Node 1
        if err1 == nil && err2 != nil {
            result.Node1OnlyBlocks = append(result.Node1OnlyBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            fmt.Printf("✖ Block %d: MISSING on Node2 (exists on Node1)\n", i)
            fmt.Printf("  └─ Node1 hash: %s...\n", block1.Hash[:16])
            continue
        }

        // Both blocks exist - compare them
        hasMismatch := false

        // Compare hashes
        if block1.Hash != block2.Hash {
            result.MismatchedBlocks = append(result.MismatchedBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            
            errMsg := fmt.Sprintf("Block %d: Hash mismatch", i)
            result.HashMismatches = append(result.HashMismatches, errMsg)
            hasMismatch = true
            
            fmt.Printf("✖ Block %d: HASH MISMATCH\n", i)
            fmt.Printf("  ├─ Node1: %s\n", block1.Hash)
            fmt.Printf("  └─ Node2: %s\n", block2.Hash)
        }

        // Compare data
        if block1.Data != block2.Data {
            errMsg := fmt.Sprintf("Block %d: Data differs", i)
            result.DataMismatches = append(result.DataMismatches, errMsg)
            
            if !hasMismatch {
                fmt.Printf("⚠ Block %d: DATA MISMATCH (but hash matches)\n", i)
            }
            fmt.Printf("  ├─ Node1 data: %s\n", block1.Data)
            fmt.Printf("  └─ Node2 data: %s\n", block2.Data)
            hasMismatch = true
        }

        // Compare timestamps
        if block1.Timestamp != block2.Timestamp {
            timeDiff := block1.Timestamp - block2.Timestamp
            errMsg := fmt.Sprintf("Block %d: Timestamp differs by %d seconds", i, timeDiff)
            result.TimestampMismatches = append(result.TimestampMismatches, errMsg)
            
            if !hasMismatch {
                fmt.Printf("⚠ Block %d: TIMESTAMP MISMATCH\n", i)
            }
            fmt.Printf("  ├─ Node1: %s (Unix: %d)\n", 
                time.Unix(block1.Timestamp, 0).Format("2006-01-02 15:04:05"), 
                block1.Timestamp)
            fmt.Printf("  └─ Node2: %s (Unix: %d)\n", 
                time.Unix(block2.Timestamp, 0).Format("2006-01-02 15:04:05"), 
                block2.Timestamp)
        }

        // Compare prevHash
        if block1.PrevHash != block2.PrevHash {
            if !hasMismatch {
                fmt.Printf("⚠ Block %d: PREVHASH MISMATCH\n", i)
            }
            fmt.Printf("  ├─ Node1 prevHash: %s\n", block1.PrevHash)
            fmt.Printf("  └─ Node2 prevHash: %s\n", block2.PrevHash)
            hasMismatch = true
        }

        if !hasMismatch {
            fmt.Printf("✔ Block %d: MATCH\n", i)
            result.MatchingBlocks++
        }
    }

    fmt.Println(strings.Repeat("─", 66))

    // =====================================================
    // GENERATE RECOMMENDATIONS
    // =====================================================
    result.Recommendations = generateComparisonRecommendations(result)

    // =====================================================
    // DISPLAY COMPREHENSIVE SUMMARY
    // =====================================================
    displayComparisonSummary(result)
}

func generateComparisonRecommendations(result ComparisonResult) []string {
    recs := []string{}

    heightDiff := result.Node1Height - result.Node2Height

    // Height-based recommendations
    if heightDiff > 0 {
        recs = append(recs, 
            fmt.Sprintf("Node2 is %d blocks behind - sync from Node1", heightDiff))
    } else if heightDiff < 0 {
        recs = append(recs, 
            fmt.Sprintf("Node1 is %d blocks behind - sync from Node2", -heightDiff))
    }

    // Divergence recommendations
    if result.DivergencePoint >= 0 {
        recs = append(recs, 
            fmt.Sprintf("Chains diverge at block %d - investigate fork cause", result.DivergencePoint))
        
        if result.DivergencePoint < 10 {
            recs = append(recs, "Early divergence detected - possible genesis block issue")
        }
    }

    // Hash mismatch recommendations
    if len(result.HashMismatches) > 0 {
        recs = append(recs, 
            fmt.Sprintf("Found %d hash mismatches - possible data corruption or fork", 
                len(result.HashMismatches)))
    }

    // Missing block recommendations
    if len(result.Node1OnlyBlocks) > 0 {
        recs = append(recs, 
            fmt.Sprintf("Node2 missing %d blocks - sync required", len(result.Node1OnlyBlocks)))
    }
    
    if len(result.Node2OnlyBlocks) > 0 {
        recs = append(recs, 
            fmt.Sprintf("Node1 missing %d blocks - sync required", len(result.Node2OnlyBlocks)))
    }

    // Timestamp recommendations
    if len(result.TimestampMismatches) > 3 {
        recs = append(recs, "Multiple timestamp mismatches - check node time synchronization")
    }

    // Perfect sync
    if len(recs) == 0 {
        recs = append(recs, "✔ Nodes are perfectly synchronized - no action needed")
    }

    return recs
}

func displayComparisonSummary(result ComparisonResult) {
    fmt.Println("\n" + strings.Repeat("═", 66))
    fmt.Println("COMPARISON SUMMARY")
    fmt.Println(strings.Repeat("═", 66))

    // Node info
    fmt.Println("\n📊 NODE INFORMATION:")
    fmt.Printf("  Node1: %s (Height: %d)\n", result.Node1Path, result.Node1Height)
    fmt.Printf("  Node2: %s (Height: %d)\n", result.Node2Path, result.Node2Height)

    heightDiff := result.Node1Height - result.Node2Height
    if heightDiff > 0 {
        fmt.Printf("  └─ Node1 is %d blocks ahead\n", heightDiff)
    } else if heightDiff < 0 {
        fmt.Printf("  └─ Node2 is %d blocks ahead\n", -heightDiff)
    } else {
        fmt.Printf("  └─ Both nodes at same height\n")
    }

    // Match statistics
    fmt.Println("\n🔍 COMPARISON RESULTS:")
    fmt.Printf("  ✔ Matching blocks:        %d\n", result.MatchingBlocks)
    fmt.Printf("  ✖ Mismatched blocks:      %d\n", len(result.MismatchedBlocks))
    fmt.Printf("  ⚠ Node1-only blocks:      %d\n", len(result.Node1OnlyBlocks))
    fmt.Printf("  ⚠ Node2-only blocks:      %d\n", len(result.Node2OnlyBlocks))

    // Divergence point
    if result.DivergencePoint >= 0 {
        fmt.Printf("\n🔀 DIVERGENCE DETECTED:\n")
        fmt.Printf("  └─ Divergence point: Block %d\n", result.DivergencePoint)
    } else {
        fmt.Printf("\n✔ NO DIVERGENCE: Chains are synchronized\n")
    }

    // Detailed differences
    fmt.Println("\n📋 DETAILED DIFFERENCES:")
    
    if len(result.HashMismatches) > 0 {
        fmt.Printf("  • Hash mismatches:      %d\n", len(result.HashMismatches))
        if len(result.HashMismatches) <= 5 {
            for _, err := range result.HashMismatches {
                fmt.Printf("    - %s\n", err)
            }
        }
    }
    
    if len(result.DataMismatches) > 0 {
        fmt.Printf("  • Data mismatches:      %d\n", len(result.DataMismatches))
    }
    
    if len(result.TimestampMismatches) > 0 {
        fmt.Printf("  • Timestamp mismatches: %d\n", len(result.TimestampMismatches))
    }

    if len(result.Node1OnlyBlocks) > 0 {
        fmt.Printf("  • Blocks only on Node1: %v\n", result.Node1OnlyBlocks)
    }
    
    if len(result.Node2OnlyBlocks) > 0 {
        fmt.Printf("  • Blocks only on Node2: %v\n", result.Node2OnlyBlocks)
    }

    // Sync percentage
    maxHeight := result.Node1Height
    if result.Node2Height > maxHeight {
        maxHeight = result.Node2Height
    }
    
    syncPercentage := float64(0)
    if maxHeight >= 0 {
        syncPercentage = (float64(result.MatchingBlocks) / float64(maxHeight+1)) * 100
    }
    
    fmt.Printf("\n📈 SYNCHRONIZATION: %.1f%%\n", syncPercentage)

    // Recommendations
    fmt.Println("\n🔧 RECOMMENDATIONS:")
    for i, rec := range result.Recommendations {
        fmt.Printf("  %d. %s\n", i+1, rec)
    }

    fmt.Println("\n" + strings.Repeat("═", 66))
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

func main() {
    dbPath := flag.String("db", "./leveldb-data", "Path to LevelDB database")
    db1Path := flag.String("db1", "./node1-data", "Path to first database (for compare)")
    db2Path := flag.String("db2", "./node2-data", "Path to second database (for compare)")
    cmd := flag.String("cmd", "compare", "Command: load, view, compare")
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
            fmt.Println("Usage: -cmd view -db <path> <block_height>")
            return
        }
        height, err := strconv.Atoi(flag.Arg(0))
        if err != nil {
            fmt.Println("Invalid block height")
            return
        }
        ViewBlock(db, height)

    case "compare":
        db1, err1 := leveldb.OpenFile(*db1Path, nil)
        if err1 != nil {
            fmt.Printf("Failed to open Node1 database: %v\n", err1)
            return
        }
        defer db1.Close()

        db2, err2 := leveldb.OpenFile(*db2Path, nil)
        if err2 != nil {
            fmt.Printf("Failed to open Node2 database: %v\n", err2)
            return
        }
        defer db2.Close()

        CompareNodes(db1, db2, *db1Path, *db2Path)

    default:
        fmt.Printf("Unknown command: %s\n", *cmd)
        fmt.Println("\nAvailable commands:")
        fmt.Println("  load    - Load sample blockchain data")
        fmt.Println("  view    - View a specific block")
        fmt.Println("  compare - Compare two blockchain nodes")
        fmt.Println("\nExamples:")
        fmt.Println("  go run main.go -cmd load -db ./node1-data -blocks 50")
        fmt.Println("  go run main.go -cmd load -db ./node2-data -blocks 45")
        fmt.Println("  go run main.go -cmd compare -db1 ./node1-data -db2 ./node2-data")
        fmt.Println("  go run main.go -cmd view -db ./node1-data 10")
    }
}
