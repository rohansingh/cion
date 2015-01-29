package cion

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"io"
)

var (
	JobsBucket = []byte("jobs")
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
	Jobs   *bolt.Bucket
	Owner  *bolt.Bucket
	Repo   *bolt.Bucket
	Branch *bolt.Bucket
	Logs   *bolt.Bucket
}

// boltJobRef is a reference to a job in a specific owner/repo/branch bucket.
type boltJobRef struct {
	Owner  string
	Repo   string
	Branch string
	Number uint64
}

func NewBoltJobStore(path string) (*BoltJobStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	return &BoltJobStore{db: db}, nil
}

func (s *BoltJobStore) GetByID(id uint64) (*Job, error) {
	var ref boltJobRef

	err := s.db.View(func(tx *bolt.Tx) error {
		jb, err := tx.CreateBucketIfNotExists(JobsBucket)
		if err != nil {
			return err
		}

		key := Uint64ToBytes(id)
		return json.Unmarshal(jb.Get(key), &ref)
	})
	if err != nil {
		return nil, err
	}

	return s.GetByNumber(ref.Owner, ref.Repo, ref.Branch, ref.Number)
}

func (s *BoltJobStore) GetByNumber(owner, repo, branch string, number uint64) (*Job, error) {
	j := &Job{}

	ref := boltJobRef{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
		Number: number,
	}

	err := s.db.View(func(tx *bolt.Tx) error {
		b, err := getBuckets(ref, tx)
		if err != nil {
			return err
		}

		key := Uint64ToBytes(number)
		return json.Unmarshal(b.Branch.Get(key), j)
	})
	if err != nil {
		return nil, err
	}

	return j, nil
}

func (s *BoltJobStore) List(owner, repo, branch string) ([]*Job, error) {
	var l []*Job

	ref := boltJobRef{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
	}

	if err := s.db.View(func(tx *bolt.Tx) error {
		b, err := getBuckets(ref, tx)
		if err != nil {
			return err
		}

		return b.Branch.ForEach(func(key, val []byte) error {
			var j *Job
			if err := json.Unmarshal(val, j); err != nil {
				return err
			}

			l = append(l, j)
			return nil
		})
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
			Branch: j.Branch,
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

		return b.Logs.Put(Uint64ToBytes(i), p)
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
			c, err := w.Write(val)
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
//    Jobs -> (owner) -> (repo) -> (branch) -> (job number)_Logs
//
// The actual data for a job is saved to the bucket for its branch. The Jobs bucket contains
// mappings from a job's unique ID to the branch where it is saved. This allows us to look up
// jobs either by their unique ID, or by their owner/repo/branch/number combination.
//
// The logs for a job are saved to the (job number)_Logs bucket.
func (s *BoltJobStore) Save(j *Job) error {
	ref := boltJobRef{
		Owner:  j.Owner,
		Repo:   j.Repo,
		Branch: j.Branch,
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b, err := getBuckets(ref, tx)
		if err != nil {
			return err
		}

		if j.ID == 0 {
			// assign the job an incrementing ID and number
			id, err := b.Jobs.NextSequence()
			if err != nil {
				return err
			}

			n, err := b.Branch.NextSequence()
			if err != nil {
				return err
			}

			j.ID, j.Number = id, n

			// save a reference from the job's unique ID to its home bucket
			if err := saveJobRef(j.ID, j.Number, j.Owner, j.Repo, j.Branch, b.Jobs); err != nil {
				return err
			}
		}

		key := Uint64ToBytes(j.Number)
		val, err := json.Marshal(j)
		if err != nil {
			return err
		}

		return b.Branch.Put(key, val)
	})
}

func getBuckets(ref boltJobRef, tx *bolt.Tx) (*buckets, error) {
	var jb, ob, rb, bb, lb *bolt.Bucket
	var err error

	if tx.Writable() {
		jb, err = tx.CreateBucketIfNotExists(JobsBucket)
	} else {
		jb = tx.Bucket(JobsBucket)
	}
	if err != nil {
		return nil, err
	} else if jb == nil {
		return nil, errors.New("Jobs bucket doesn't exist")
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

	if tx.Writable() {
		bb, err = rb.CreateBucketIfNotExists([]byte(ref.Branch))
	} else {
		bb = rb.Bucket([]byte(ref.Branch))
	}
	if err != nil {
		return nil, err
	} else if bb == nil {
		return nil, errors.New("branch bucket doesn't exist for ref: " + fmt.Sprint(ref))
	}

	if ref.Number != 0 {
		lbn := string(ref.Number) + "_Logs"

		if tx.Writable() {
			lb, err = bb.CreateBucketIfNotExists([]byte(lbn))
		} else {
			lb = bb.Bucket([]byte(lbn))
		}
		if err != nil {
			return nil, err
		} else if lb == nil {
			return nil, errors.New("logs bucket doesn't exist for ref: " + fmt.Sprint(ref))
		}
	}

	return &buckets{
		Jobs:   jb,
		Owner:  ob,
		Repo:   rb,
		Branch: bb,
		Logs:   lb,
	}, nil
}

func saveJobRef(id, number uint64, owner, repo, branch string, b *bolt.Bucket) error {
	ref := boltJobRef{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
		Number: number,
	}

	key := Uint64ToBytes(id)
	val, err := json.Marshal(ref)
	if err != nil {
		return err
	}

	return b.Put(key, val)
}

func Uint64ToBytes(x uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, x)

	return b
}
