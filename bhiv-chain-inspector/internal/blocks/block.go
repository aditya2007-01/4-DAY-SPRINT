package blocks

type Block struct {
    Height    int    `json:"height"`
    Hash      string `json:"hash"`
    PrevHash  string `json:"prev_hash"`
    Data      string `json:"data"`
    Timestamp int64  `json:"timestamp"`
}
