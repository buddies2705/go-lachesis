package topicsdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/Fantom-foundation/go-lachesis/kvdb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/table"
)

const MaxCount = 0xff

var ErrTooManyTopics = fmt.Errorf("Too many topics")

// Index is a specialized indexes for log records storing and fetching.
type Index struct {
	db    kvdb.KeyValueStore
	table struct {
		// topic+topicN+(blockN+TxHash+logIndex) -> topic_count
		Topic kvdb.KeyValueStore `table:"t"`
		// (blockN+TxHash+logIndex) + topicN -> topic
		Other kvdb.KeyValueStore `table:"o"`
		// (blockN+TxHash+logIndex) -> address, data
		Logrec kvdb.KeyValueStore `table:"r"`
	}

	fetchMethod func(topics [][]common.Hash) ([]*types.Log, error)
}

// New TopicsDb instance.
func New(db kvdb.KeyValueStore) *Index {
	tt := &Index{
		db: db,
	}

	tt.fetchMethod = tt.fetchAsync

	table.MigrateTables(&tt.table, tt.db)

	return tt
}

// Find log records by conditions.
func (tt *Index) Find(topics [][]common.Hash) ([]*types.Log, error) {
	return tt.fetchMethod(topics)
}

// MustPush calls Push() and panics if error.
func (tt *Index) MustPush(recs ...*types.Log) {
	err := tt.Push(recs...)
	if err != nil {
		panic(err)
	}
}

// Push log record to database.
func (tt *Index) Push(recs ...*types.Log) error {
	for _, rec := range recs {
		if len(rec.Topics) > MaxCount {
			return ErrTooManyTopics
		}
		count := posToBytes(uint8(len(rec.Topics)))

		id := NewID(rec.BlockNumber, rec.TxHash, rec.Index)

		for pos, topic := range rec.Topics {
			key := topicKey(topic, uint8(pos), id)
			err := tt.table.Topic.Put(key, count)
			if err != nil {
				return err
			}

			key = otherKey(id, uint8(pos))
			err = tt.table.Other.Put(key, topic.Bytes())
			if err != nil {
				return err
			}
		}

		err := tt.table.Logrec.Put(id.Bytes(), append(rec.Data, rec.Address[:]...))
		if err != nil {
			return err
		}
	}

	return nil
}
