package storage

import (
	"dashboard/common"
	"dashboard/config"
	"fmt"
	"sync"
	"time"
)

type sessionKey struct {
	id string
	ip string
}

// SessionStore manages per-session in-memory SQLite databases for demo mode.
type SessionStore struct {
	mu         sync.Mutex
	sessions   map[sessionKey]Datasource
	defaultCfg config.Config
	wrap       func(Datasource) Datasource
}

// NewSessionStore creates a new session store seeded with the given default config.
// The wrap function is applied to each newly created datasource (e.g. to add error annotation).
func NewSessionStore(cfg config.Config, wrap func(Datasource) Datasource) *SessionStore {
	return &SessionStore{
		sessions:   make(map[sessionKey]Datasource),
		defaultCfg: cfg,
		wrap:       wrap,
	}
}

// GetOrCreate returns the datasource for the given session, creating one if needed.
// Sessions are keyed by (sessionID, clientIP) — a shared cookie from a different IP
// produces a different key. Returns an error if the session cap is reached.
func (s *SessionStore) GetOrCreate(sessionID, clientIP string) (Datasource, error) {
	key := sessionKey{id: sessionID, ip: clientIP}

	s.mu.Lock()
	defer s.mu.Unlock()

	if ds, ok := s.sessions[key]; ok {
		return ds, nil
	}

	maxSessions, err := common.DemoMaxSessions()
	if err != nil {
		return nil, err
	}
	if len(s.sessions) >= maxSessions {
		return nil, fmt.Errorf("demo session limit reached")
	}

	ttl, err := common.DemoSessionTTL()
	if err != nil {
		return nil, err
	}

	db, err := NewMemorySQLiteDB()
	if err != nil {
		return nil, fmt.Errorf("create session db: %w", err)
	}
	if err := db.ImportConfig(s.defaultCfg); err != nil {
		db.Close()
		return nil, fmt.Errorf("seed session db: %w", err)
	}

	var ds Datasource = db
	if s.wrap != nil {
		ds = s.wrap(db)
	}
	s.sessions[key] = ds

	go func() {
		time.Sleep(ttl)
		s.mu.Lock()
		delete(s.sessions, key)
		s.mu.Unlock()
		// Grace period: let in-flight requests drain before closing the DB.
		time.Sleep(60 * time.Second)
		db.Close()
	}()

	return ds, nil
}
