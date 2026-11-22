package idempotency

import (
	"net/http"
	"sync"
	"time"
)

type State int

const (
	StateMissing State = iota
	StateInProgress
	StateFound
)

type StoredResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

type Store interface {
	Begin(key string) (response *StoredResponse, state State)
	Complete(key string, response *StoredResponse)
	Abort(key string)
}

type memoryEntry struct {
	inProgress bool
	expires    time.Time
	response   *StoredResponse
}

type MemoryStore struct {
	mu          sync.Mutex
	entries     map[string]*memoryEntry
	ttl         time.Duration
	lastCleanup time.Time
	now         func() time.Time
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore(ttl time.Duration) *MemoryStore {
	now := time.Now
	return &MemoryStore{
		entries:     make(map[string]*memoryEntry),
		ttl:         ttl,
		lastCleanup: now(),
		now:         now,
	}
}

func (s *MemoryStore) Begin(key string) (*StoredResponse, State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.cleanup(now)

	if existing, ok := s.entries[key]; ok {
		if existing.inProgress {
			return nil, StateInProgress
		}

		if now.After(existing.expires) {
			delete(s.entries, key)
		} else {
			return existing.response, StateFound
		}
	}

	s.entries[key] = &memoryEntry{inProgress: true}

	return nil, StateMissing
}

func (s *MemoryStore) Complete(key string, response *StoredResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.entries[key]
	if !ok {
		return
	}

	existing.inProgress = false
	existing.response = response
	existing.expires = s.now().Add(s.ttl)
}

func (s *MemoryStore) Abort(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.entries[key]; ok && existing.inProgress {
		delete(s.entries, key)
	}
}

func (s *MemoryStore) cleanup(now time.Time) {
	interval := time.Minute
	if s.ttl < interval {
		interval = s.ttl
	}

	if now.Sub(s.lastCleanup) < interval {
		return
	}
	s.lastCleanup = now

	for key, existing := range s.entries {
		if !existing.inProgress && now.After(existing.expires) {
			delete(s.entries, key)
		}
	}
}
