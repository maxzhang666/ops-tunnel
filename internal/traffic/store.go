package traffic

import (
	"encoding/binary"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var bucketName = []byte("traffic")

// Sample is a persisted traffic record.
type Sample struct {
	TS       time.Time `json:"ts"`
	BytesIn  int64     `json:"bytesIn"`
	BytesOut int64     `json:"bytesOut"`
	Conns    int64     `json:"conns"`
}

// Store persists traffic samples in a BBolt database.
type Store struct {
	db *bolt.DB
}

// NewStore opens or creates the traffic database at the given path.
func NewStore(path string) (*Store, error) {
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open traffic db: %w", err)
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	}); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Record writes a traffic sample for the given tunnel.
func (s *Store) Record(tunnelID string, bytesIn, bytesOut, conns int64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		key := makeKey(tunnelID, time.Now().UTC())
		val := encodeValue(bytesIn, bytesOut, conns)
		return b.Put(key, val)
	})
}

// Query returns aggregated samples for the given time range and step.
func (s *Store) Query(from, to time.Time, step time.Duration) ([]Sample, error) {
	// Collect raw entries
	type raw struct {
		ts       time.Time
		bytesIn  int64
		bytesOut int64
		conns    int64
	}
	var raws []raw

	if err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()
		// Scan all keys; filter by timestamp portion
		for k, v := c.First(); k != nil; k, v = c.Next() {
			ts := extractTS(k)
			if ts.Before(from) || !ts.Before(to) {
				continue
			}
			bi, bo, cn := decodeValue(v)
			raws = append(raws, raw{ts: ts, bytesIn: bi, bytesOut: bo, conns: cn})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if len(raws) == 0 {
		return nil, nil
	}

	// Aggregate into step-sized buckets
	buckets := make(map[int64]*Sample)
	for _, r := range raws {
		bucketTS := r.ts.Truncate(step)
		key := bucketTS.Unix()
		if _, ok := buckets[key]; !ok {
			buckets[key] = &Sample{TS: bucketTS}
		}
		buckets[key].BytesIn += r.bytesIn
		buckets[key].BytesOut += r.bytesOut
		buckets[key].Conns += r.conns
	}

	// Sort by time
	result := make([]Sample, 0, len(buckets))
	for t := from.Truncate(step); t.Before(to); t = t.Add(step) {
		key := t.Unix()
		if s, ok := buckets[key]; ok {
			result = append(result, *s)
		} else {
			result = append(result, Sample{TS: t})
		}
	}
	return result, nil
}

// Prune removes entries older than the given duration.
func (s *Store) Prune(olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan)
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if extractTS(k).Before(cutoff) {
				if err := c.Delete(); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Key format: tunnelID (variable length) + ":" + unix_ts (8 bytes big-endian)
func makeKey(tunnelID string, ts time.Time) []byte {
	id := []byte(tunnelID)
	key := make([]byte, len(id)+1+8)
	copy(key, id)
	key[len(id)] = ':'
	binary.BigEndian.PutUint64(key[len(id)+1:], uint64(ts.Unix()))
	return key
}

func extractTS(key []byte) time.Time {
	if len(key) < 9 {
		return time.Time{}
	}
	tsBytes := key[len(key)-8:]
	return time.Unix(int64(binary.BigEndian.Uint64(tsBytes)), 0).UTC()
}

// Value format: bytesIn(8) + bytesOut(8) + conns(8) = 24 bytes
func encodeValue(bytesIn, bytesOut, conns int64) []byte {
	buf := make([]byte, 24)
	binary.BigEndian.PutUint64(buf[0:8], uint64(bytesIn))
	binary.BigEndian.PutUint64(buf[8:16], uint64(bytesOut))
	binary.BigEndian.PutUint64(buf[16:24], uint64(conns))
	return buf
}

func decodeValue(val []byte) (bytesIn, bytesOut, conns int64) {
	if len(val) < 24 {
		return
	}
	bytesIn = int64(binary.BigEndian.Uint64(val[0:8]))
	bytesOut = int64(binary.BigEndian.Uint64(val[8:16]))
	conns = int64(binary.BigEndian.Uint64(val[16:24]))
	return
}
