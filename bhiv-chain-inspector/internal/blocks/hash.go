package blocks

import (
    "crypto/sha256"
    "encoding/hex"
    "strconv"
)

func ComputeHash(height int, prevHash string, data string, timestamp int64) string {
    record := strconv.Itoa(height) + prevHash + data + strconv.FormatInt(timestamp, 10)
    h := sha256.New()
    h.Write([]byte(record))
    hashed := h.Sum(nil)
    return hex.EncodeToString(hashed)
}
