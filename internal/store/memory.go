package store

import (
	"sync"
	"time"

	"go04-pet-adoption/internal/model"
)

type IDGenerator func(prefix string) string

// MemoryStore 基于 map + RWMutex 的内存实现。写成功后在释放锁后调用 persistHook。
type MemoryStore struct {
	mu sync.RWMutex

	users     map[string]model.User
	username  map[string]string
	shelters  map[string]model.Shelter
	pets      map[string]model.Pet
	apps      map[string]model.Application
	appIdx    map[string]string
	visits    map[string]model.Visit
	health    map[string]model.HealthRecord
	favorites map[string]model.Favorite
	favIdx    map[model.FavoriteKey]string
	inquiries map[string]model.Inquiry
	notifs    map[string]model.Notification
	audits    map[string]model.AuditLog
	credits   map[string]model.CreditLog

	now         func() time.Time
	genID       IDGenerator
	persistHook func()
	persistWG   sync.WaitGroup
}

func NewMemoryStore(now func() time.Time, genID IDGenerator) *MemoryStore {
	if now == nil {
		now = time.Now
	}
	if genID == nil {
		genID = defaultIDGenerator
	}
	return &MemoryStore{
		users:     make(map[string]model.User),
		username:  make(map[string]string),
		shelters:  make(map[string]model.Shelter),
		pets:      make(map[string]model.Pet),
		apps:      make(map[string]model.Application),
		appIdx:    make(map[string]string),
		visits:    make(map[string]model.Visit),
		health:    make(map[string]model.HealthRecord),
		favorites: make(map[string]model.Favorite),
		favIdx:    make(map[model.FavoriteKey]string),
		inquiries: make(map[string]model.Inquiry),
		notifs:    make(map[string]model.Notification),
		audits:    make(map[string]model.AuditLog),
		credits:   make(map[string]model.CreditLog),
		now:       now,
		genID:     genID,
	}
}

func (s *MemoryStore) SetPersistHook(hook func()) {
	s.mu.Lock()
	s.persistHook = hook
	s.mu.Unlock()
}

func (s *MemoryStore) afterWrite() {
	if s.persistHook == nil {
		return
	}
	hook := s.persistHook
	// persistHook 会调用 Snapshot()（再取读锁）。写路径持有写锁，必须异步落盘，
	// 否则 RWMutex 重入会死锁。关停时仍由 FileStore.Flush 同步刷盘。
	s.persistWG.Add(1)
	go func() {
		defer s.persistWG.Done()
		hook()
	}()
}

func (s *MemoryStore) waitForPersist() {
	s.persistWG.Wait()
}

func appKey(petID, applicantID string) string { return petID + "|" + applicantID }

func favKey(userID, petID string) model.FavoriteKey {
	return model.FavoriteKey{UserID: userID, PetID: petID}
}

const snapshotVersion = 1

type Snapshot struct {
	Version       int                  `json:"version"`
	Users         []model.User         `json:"users"`
	Shelters      []model.Shelter      `json:"shelters"`
	Pets          []model.Pet          `json:"pets"`
	Applications  []model.Application  `json:"applications"`
	Visits        []model.Visit        `json:"visits"`
	Health        []model.HealthRecord `json:"health"`
	Favorites     []model.Favorite     `json:"favorites"`
	Inquiries     []model.Inquiry      `json:"inquiries"`
	Notifications []model.Notification `json:"notifications"`
	AuditLogs     []model.AuditLog     `json:"audit_logs"`
	CreditLogs    []model.CreditLog    `json:"credit_logs"`
}

func mapValues[K comparable, V any](m map[K]V) []V {
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func (s *MemoryStore) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Snapshot{
		Version:       snapshotVersion,
		Users:         mapValues(s.users),
		Shelters:      mapValues(s.shelters),
		Pets:          mapValues(s.pets),
		Applications:  mapValues(s.apps),
		Visits:        mapValues(s.visits),
		Health:        mapValues(s.health),
		Favorites:     mapValues(s.favorites),
		Inquiries:     mapValues(s.inquiries),
		Notifications: mapValues(s.notifs),
		AuditLogs:     mapValues(s.audits),
		CreditLogs:    mapValues(s.credits),
	}
}

func (s *MemoryStore) ReplaceAll(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[string]model.User, len(snap.Users))
	s.username = make(map[string]string, len(snap.Users))
	for _, u := range snap.Users {
		s.users[u.ID] = u
		s.username[u.Username] = u.ID
	}
	s.shelters = make(map[string]model.Shelter, len(snap.Shelters))
	for _, sh := range snap.Shelters {
		s.shelters[sh.ID] = sh
	}
	s.pets = make(map[string]model.Pet, len(snap.Pets))
	for _, p := range snap.Pets {
		p.Personality = cloneStrings(p.Personality)
		p.Photos = cloneStrings(p.Photos)
		s.pets[p.ID] = p
	}
	s.apps = make(map[string]model.Application, len(snap.Applications))
	s.appIdx = make(map[string]string, len(snap.Applications))
	for _, a := range snap.Applications {
		s.apps[a.ID] = a
		s.appIdx[appKey(a.PetID, a.ApplicantID)] = a.ID
	}
	s.visits = make(map[string]model.Visit, len(snap.Visits))
	for _, v := range snap.Visits {
		s.visits[v.ID] = v
	}
	s.health = make(map[string]model.HealthRecord, len(snap.Health))
	for _, h := range snap.Health {
		s.health[h.ID] = h
	}
	s.favorites = make(map[string]model.Favorite, len(snap.Favorites))
	s.favIdx = make(map[model.FavoriteKey]string, len(snap.Favorites))
	for _, f := range snap.Favorites {
		s.favorites[f.ID] = f
		s.favIdx[favKey(f.UserID, f.PetID)] = f.ID
	}
	s.inquiries = make(map[string]model.Inquiry, len(snap.Inquiries))
	for _, q := range snap.Inquiries {
		s.inquiries[q.ID] = q
	}
	s.notifs = make(map[string]model.Notification, len(snap.Notifications))
	for _, n := range snap.Notifications {
		s.notifs[n.ID] = n
	}
	s.audits = make(map[string]model.AuditLog, len(snap.AuditLogs))
	for _, l := range snap.AuditLogs {
		s.audits[l.ID] = l
	}
	s.credits = make(map[string]model.CreditLog, len(snap.CreditLogs))
	for _, c := range snap.CreditLogs {
		s.credits[c.ID] = c
	}
}
