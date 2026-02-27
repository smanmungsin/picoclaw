package brain

import (
	"github.com/dgraph-io/badger/v3"
)

type BadgerMemory struct {
	db *badger.DB
}

func NewBadgerMemory(path string) (*BadgerMemory, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}
	return &BadgerMemory{db: db}, nil
}

func (m *BadgerMemory) Remember(key string, value any) error {
	valBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), valBytes)
	})
}

func (m *BadgerMemory) Recall(key string) (any, error) {
	var val any
	err := m.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(v []byte) error {
			return json.Unmarshal(v, &val)
		})
	})
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (m *BadgerMemory) Search(query string) ([]any, error) {
	results := []any{}
	err := m.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if string(k) == query {
				var val any
				item.Value(func(v []byte) error {
					return json.Unmarshal(v, &val)
				})
				results = append(results, val)
			}
		}
		return nil
	})
	return results, err
}

func (m *BadgerMemory) Forget(key string) error {
	return m.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (m *BadgerMemory) ListKeys() ([]string, error) {
	keys := []string{}
	err := m.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			keys = append(keys, string(item.Key()))
		}
		return nil
	})
	return keys, err
}

func (m *BadgerMemory) Close() error {
	return m.db.Close()
}
