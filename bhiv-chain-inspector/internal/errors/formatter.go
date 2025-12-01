package errors

import (
    "encoding/json"
    "fmt"
    "strings"
)

func OutputScanResult(result *ErrorScanResult, jsonMode bool) {
    if jsonMode {
        outputJSON(result)
    } else {
        outputScanText(result)
    }
}

func OutputComparisonResult(result *ComparisonResult, jsonMode bool) {
    if jsonMode {
        outputJSON(result)
    } else {
        outputComparisonText(result)
    }
}

func outputJSON(data interface{}) {
    jsonData, _ := json.MarshalIndent(data, "", "  ")
    fmt.Println(string(jsonData))
}

func outputScanText(result *ErrorScanResult) {
    fmt.Println("\n" + strings.Repeat("═", 66))
    fmt.Println("BLOCKCHAIN ERROR SCAN SUMMARY")
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
        fmt.Println("\n⚠️  Errors detected.")
    }
    fmt.Println(strings.Repeat("═", 66))
}

func outputComparisonText(result *ComparisonResult) {
    fmt.Println("\n" + strings.Repeat("═", 66))
    fmt.Println("NODE COMPARISON SUMMARY")
    fmt.Println(strings.Repeat("═", 66))
    fmt.Printf("\n📊 NODE INFO:\n")
    fmt.Printf("  Node1: %s (Height: %d)\n", result.Node1Path, result.Node1Height)
    fmt.Printf("  Node2: %s (Height: %d)\n", result.Node2Path, result.Node2Height)
    
    fmt.Println("\n🔍 RESULTS:")
    fmt.Printf("  Matching Blocks:    %d\n", result.MatchingBlocks)
    fmt.Printf("  Mismatched Blocks:  %d\n", len(result.MismatchedBlocks))
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
