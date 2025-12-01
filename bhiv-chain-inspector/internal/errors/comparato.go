package errors

import (
    "fmt"
    "time"

    "bhiv-chain-inspector/internal/db"
)

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

func CompareNodes(storage1, storage2 *db.Storage, db1Path, db2Path string) *ComparisonResult {
    result := &ComparisonResult{
        ScanTime:        time.Now().Format("2006-01-02 15:04:05"),
        Node1Path:       db1Path,
        Node2Path:       db2Path,
        DivergencePoint: -1,
    }

    result.Node1Height = storage1.GetMaxHeight()
    result.Node2Height = storage2.GetMaxHeight()

    maxHeight := result.Node1Height
    if result.Node2Height > maxHeight {
        maxHeight = result.Node2Height
    }

    for i := 0; i <= maxHeight; i++ {
        block1, err1 := storage1.LoadBlock(i)
        block2, err2 := storage2.LoadBlock(i)

        if err1 != nil && err2 != nil {
            continue
        }

        if err1 != nil && err2 == nil {
            result.Node2OnlyBlocks = append(result.Node2OnlyBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
            }
            continue
        }

        if err1 == nil && err2 != nil {
            result.Node1OnlyBlocks = append(result.Node1OnlyBlocks, i)
            if result.DivergencePoint == -1 {
                result.DivergencePoint = i
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
        } else {
            result.MatchingBlocks++
        }

        if block1.Data != block2.Data {
            errMsg := fmt.Sprintf("Block %d: Data differs", i)
            result.DataMismatches = append(result.DataMismatches, errMsg)
        }

        if block1.Timestamp != block2.Timestamp {
            errMsg := fmt.Sprintf("Block %d: Timestamp differs", i)
            result.TimestampMismatches = append(result.TimestampMismatches, errMsg)
        }
    }

    if maxHeight >= 0 {
        result.SyncPercentage = (float64(result.MatchingBlocks) / float64(maxHeight+1)) * 100
    }

    result.Recommendations = generateRecommendations(result)

    return result
}

func generateRecommendations(result *ComparisonResult) []string {
    recs := []string{}

    heightDiff := result.Node1Height - result.Node2Height
    if heightDiff > 0 {
        recs = append(recs, fmt.Sprintf("Node2 is %d blocks behind - sync from Node1", heightDiff))
    } else if heightDiff < 0 {
        recs = append(recs, fmt.Sprintf("Node1 is %d blocks behind - sync from Node2", -heightDiff))
    }

    if result.DivergencePoint >= 0 {
        recs = append(recs, fmt.Sprintf("Chains diverge at block %d", result.DivergencePoint))
    }

    if len(recs) == 0 {
        recs = append(recs, "Nodes are perfectly synchronized")
    }

    return recs
}
