package db

import (
    "encoding/json"
    "fmt"

    "bhiv-chain-inspector/internal/blocks"
    "github.com/syndtr/goleveldb/leveldb"
)

type Storage struct {
    db *leveldb.DB
}

func NewStorage(dbPath string) (*Storage, error) {
    database, err := leveldb.OpenFile(dbPath, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    return &Storage{db: database}, nil
}

func (s *Storage) Close() error {
    return s.db.Close()
}

func (s *Storage) LoadBlock(height int) (*blocks.Block, error) {
    key := []byte(fmt.Sprintf("block-%d", height))
    data, err := s.db.Get(key, nil)
    if err != nil {
        return nil, err
    }

    var block blocks.Block
    if err := json.Unmarshal(data, &block); err != nil {
        return nil, err
    }
    return &block, nil
}

func (s *Storage) LoadBlockRaw(height int) ([]byte, error) {
    key := []byte(fmt.Sprintf("block-%d", height))
    return s.db.Get(key, nil)
}

func (s *Storage) SaveBlock(block *blocks.Block) error {
    key := []byte(fmt.Sprintf("block-%d", block.Height))
    data, err := json.Marshal(block)
    if err != nil {
        return err
    }
    return s.db.Put(key, data, nil)
}

func (s *Storage) GetMaxHeight() int {
    height := 0
    for {
        _, err := s.LoadBlock(height)
        if err != nil {
            if height == 0 {
                return -1
            }
            return height - 1
        }
        height++
    }
}
