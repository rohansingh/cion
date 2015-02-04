package cion

import (
	"code.google.com/p/snappy-go/snappy"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
)

var (
	JobsBucket    = []byte("jobs")
	JobRefsBucket = []byte("jobrefs")
)

type BoltJobStore struct {
	db *bolt.DB
}

type BoltJobLogger struct {
	db  *bolt.DB
	ref boltJobRef
}

// buckets provides references to the various Bolt buckets where job data is stored.
type buckets struct {
	Jobs  *bolt.Bucket
	Owner *bolt.Bucket
	Repo  *bolt.Bucket
	Logs  *bolt.Bucket
}

// boltJobRef is a reference to a job in a specific owner/repo/branch bucket.
type boltJobRef struct {
	Owner  string
	Repo   string
	Number uint64
}

func NewBoltJobStore(path string) (*BoltJobStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	return &BoltJobStore{db: db}, nil
}

func (s *BoltJobStore) GetByNumber(owner, repo string, number uint64) (*Job, error) {
	j := &Job{}

	ref := boltJobRef{
		Owner:  owner,
		Repo:   repo,
		Number: number,
	}

	err := s.db.View(func(tx *bolt.Tx) error {
		b, err := getBuckets(ref, tx)
		if err != nil {
			return err
		}

		key := Uint64ToBytes(number)
		return json.Unmarshal(b.Repo.Get(key), j)
	})
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (s *BoltJobStore) ListOwners() ([]string, error) {
	var l []string

	if err := s.db.View(func(tx *bolt.Tx) error {
		jb := tx.Bucket(JobsBucket)
		if jb == nil {
			return errors.New("jobs bucket doesn't exist")
		}

		return jb.ForEach(func(key, val []byte) error {
			l = append(l, string(key))
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return l, nil
}

func (s *BoltJobStore) ListRepos(owner string) ([]string, error) {
	var l []string

	if err := s.db.View(func(tx *bolt.Tx) error {
		jb := tx.Bucket(JobsBucket)
		if jb == nil {
			return errors.New("jobs bucket doesn't exist")
		}

		ob := jb.Bucket([]byte(owner))
		if ob == nil {
			return errors.New("owner bucket doesn't exist")
		}

		return ob.ForEach(func(key, val []byte) error {
			l = append(l, string(key))
			return nil
		})
	}); err != nil {
		return nil, err
	}

	return l, nil
}

func (s *BoltJobStore) List(owner, repo string) ([]*Job, error) {
	var l []*Job

	ref := boltJobRef{
		Owner: owner,
		Repo:  repo,
	}

	if err := s.db.View(func(tx *bolt.Tx) error {
		b, err := getBuckets(ref, tx)
		if err != nil {
			return err
		}

		c := b.Repo.Cursor()
		for key, val := c.Last(); key != nil; key, val = c.Prev() {
			j := &Job{}

			if len(val) > 0 {
				if err := json.Unmarshal(val, j); err != nil {
					return err
				}

				l = append(l, j)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return l, nil
}

func (s *BoltJobStore) GetLogger(j *Job) JobLogger {
	return BoltJobLogger{
		db: s.db,
		ref: boltJobRef{
			Owner:  j.Owner,
			Repo:   j.Repo,
			Number: j.Number,
		},
	}
}

func (l BoltJobLogger) Write(p []byte) (int, error) {
	if err := l.db.Update(func(tx *bolt.Tx) error {
		b, err := getBuckets(l.ref, tx)
		if err != nil {
			return err
		}

		i, err := b.Logs.NextSequence()
		if err != nil {
			return err
		}

		key := Uint64ToBytes(i)
		val, err := snappy.Encode(nil, p)
		if err != nil {
			return err
		}

		return b.Logs.Put(key, val)
	}); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (l BoltJobLogger) WriteTo(w io.Writer) (int64, error) {
	var n int64

	return n, l.db.View(func(tx *bolt.Tx) error {
		b, err := getBuckets(l.ref, tx)
		if err != nil {
			return err
		}

		return b.Logs.ForEach(func(key, val []byte) error {
			s, err := snappy.Decode(nil, val)
			if err != nil {
				return err
			}

			c, err := w.Write(s)
			n = n + int64(c)

			return err
		})
	})
}

func (l BoltJobLogger) WriteStep(name string) error {
	s := fmt.Sprintln("---", name, "---")
	_, err := l.Write([]byte(s))
	return err
}

// Save writes job data to various buckets in the Bolt database. We use this nesting pattern
// for buckets:
//
//    jobs -> (owner) -> (repo) -> logs_(job number)
//
// The actual data for a job is saved to the bucket for its repo, and its logs are stored in
// the logs_<number> sub-bucket.
func (s *BoltJobStore) Save(j *Job) error {
	ref := boltJobRef{
		Owner:  j.Owner,
		Repo:   j.Repo,
		Number: j.Number,
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := getBuckets(ref, tx)
		if err != nil {
			return err
		}

		if j.Number == 0 {
			// assign the job an incrementing build number
			n, err := b.Repo.NextSequence()
			if err != nil {
				return err
			}

			j.Number = n
		}

		key := Uint64ToBytes(j.Number)
		val, err := json.Marshal(j)
		if err != nil {
			return err
		}

		return b.Repo.Put(key, val)
	})
}

func getBuckets(ref boltJobRef, tx *bolt.Tx) (*buckets, error) {
	var jb, ob, rb, lb *bolt.Bucket
	var err error

	if tx.Writable() {
		jb, err = tx.CreateBucketIfNotExists(JobsBucket)
	} else {
		jb = tx.Bucket(JobsBucket)
	}
	if err != nil {
		return nil, err
	} else if jb == nil {
		return nil, errors.New("jobs bucket doesn't exist")
	}

	if tx.Writable() {
		ob, err = jb.CreateBucketIfNotExists([]byte(ref.Owner))
	} else {
		ob = jb.Bucket([]byte(ref.Owner))
	}
	if err != nil {
		return nil, err
	} else if ob == nil {
		return nil, errors.New("owner bucket doesn't exist for ref: " + fmt.Sprint(ref))
	}

	if tx.Writable() {
		rb, err = ob.CreateBucketIfNotExists([]byte(ref.Repo))
	} else {
		rb = ob.Bucket([]byte(ref.Repo))
	}
	if err != nil {
		return nil, err
	} else if rb == nil {
		return nil, errors.New("repo bucket doesn't exist for ref: " + fmt.Sprint(ref))
	}

	if ref.Number != 0 {
		lbn := "logs_" + string(ref.Number)

		if tx.Writable() {
			lb, err = rb.CreateBucketIfNotExists([]byte(lbn))
		} else {
			lb = rb.Bucket([]byte(lbn))
		}
		if err != nil {
			return nil, err
		} else if lb == nil {
			return nil, errors.New("logs bucket doesn't exist for ref: " + fmt.Sprint(ref))
		}
	}

	return &buckets{
		Jobs:  jb,
		Owner: ob,
		Repo:  rb,
		Logs:  lb,
	}, nil
}

func Uint64ToBytes(x uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, x)

	return b
}
