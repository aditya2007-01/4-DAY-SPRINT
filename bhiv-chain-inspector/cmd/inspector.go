package main

import (
    "flag"
    "fmt"
    "os"
    "time"

    "bhiv-chain-inspector/internal/blocks"
    "bhiv-chain-inspector/internal/db"
    "bhiv-chain-inspector/internal/errors"
)

const version = "1.0.0"

func main() {
    dbPath := flag.String("db", "./leveldb-data", "Path to LevelDB database")
    db1Path := flag.String("db1", "./node1-data", "Path to first database")
    db2Path := flag.String("db2", "./node2-data", "Path to second database")
    cmd := flag.String("cmd", "scan-errors", "Command: load, scan-errors, compare")
    numBlocks := flag.Int("blocks", 10, "Number of blocks to load")
    jsonOutput := flag.Bool("json", false, "Output in JSON format")
    showVersion := flag.Bool("version", false, "Show version")
    
    flag.Parse()

    if *showVersion {
        fmt.Printf("BHIV Chain Inspector v%s\n", version)
        return
    }

    switch *cmd {
    case "load":
        loadSampleData(*dbPath, *numBlocks)

    case "scan-errors":
        runScan(*dbPath, *jsonOutput)

    case "compare":
        runCompare(*db1Path, *db2Path, *jsonOutput)

    default:
        printUsage()
    }
}

func loadSampleData(dbPath string, numBlocks int) {
    storage, err := db.NewStorage(dbPath)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer storage.Close()

    fmt.Printf("Loading %d sample blocks into %s...\n", numBlocks, dbPath)

    prevHash := "0"
    for i := 0; i < numBlocks; i++ {
        timestamp := time.Now().Unix() + int64(i*10)
        data := fmt.Sprintf("Transaction data for block %d", i)
        hash := blocks.ComputeHash(i, prevHash, data, timestamp)

        block := &blocks.Block{
            Height:    i,
            Hash:      hash,
            PrevHash:  prevHash,
            Data:      data,
            Timestamp: timestamp,
        }

        if err := storage.SaveBlock(block); err != nil {
            fmt.Printf("Error saving block %d: %v\n", i, err)
            os.Exit(1)
        }

        fmt.Printf("✔ Block %d stored\n", i)
        prevHash = hash
    }

    fmt.Println("\nData loading complete!")
}

func runScan(dbPath string, jsonMode bool) {
    storage, err := db.NewStorage(dbPath)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer storage.Close()

    result := errors.ScanErrors(storage, dbPath)
    errors.OutputScanResult(result, jsonMode)
}

func runCompare(db1Path, db2Path string, jsonMode bool) {
    storage1, err := db.NewStorage(db1Path)
    if err != nil {
        fmt.Printf("Error opening Node1: %v\n", err)
        os.Exit(1)
    }
    defer storage1.Close()

    storage2, err := db.NewStorage(db2Path)
    if err != nil {
        fmt.Printf("Error opening Node2: %v\n", err)
        os.Exit(1)
    }
    defer storage2.Close()

    result := errors.CompareNodes(storage1, storage2, db1Path, db2Path)
    errors.OutputComparisonResult(result, jsonMode)
}

func printUsage() {
    fmt.Println("\nBHIV Blockchain Inspector CLI")
    fmt.Println("\nUsage:")
    fmt.Println("  inspector -cmd <command> [options]")
    fmt.Println("\nCommands:")
    fmt.Println("  load        Load sample blockchain data")
    fmt.Println("  scan-errors Scan blockchain for errors")
    fmt.Println("  compare     Compare two blockchain nodes")
    fmt.Println("\nExamples:")
    fmt.Println("  inspector -cmd load -db ./data -blocks 50")
    fmt.Println("  inspector -cmd scan-errors -db ./data")
    fmt.Println("  inspector -cmd scan-errors -db ./data --json")
    fmt.Println("  inspector -cmd compare -db1 ./node1 -db2 ./node2")
}
