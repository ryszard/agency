package cache

import (
	"time"

	"github.com/boltdb/bolt"
)

const bucketName = "cache"

// BoltDBCache is a BoltDB backed key-value store.
type BoltDBCache struct {
	db *bolt.DB
}

// BoltDB returns a BoltDBCache, which is BoltDB backed key-value store.
func BoltDB(filepath string) (*BoltDBCache, error) {
	db, err := bolt.Open(filepath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &BoltDBCache{
		db: db,
	}, nil
}

// Set adds a key-value pair to the cache.
func (c *BoltDBCache) Set(key []byte, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Put(key, value)
	})
}

// Get retrieves a key-value pair from the cache.
func (c *BoltDBCache) Get(key []byte) (value []byte, ok bool, err error) {

	err = c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		value = b.Get(key)
		ok = value != nil

		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return value, ok, nil
}

// Close releases all database resources.
func (c *BoltDBCache) Close() error {
	return c.db.Close()
}
