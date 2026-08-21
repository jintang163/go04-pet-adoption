package store

import (
	"context"
	"strings"
	"time"

	"go04-pet-adoption/internal/model"
)

func (s *MemoryStore) CreateUser(ctx context.Context, u model.User) (model.User, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.Username == "" {
		return model.User{}, model.ErrInvalidUsername
	}
	if _, ok := s.username[u.Username]; ok {
		return model.User{}, model.ErrAlreadyExists
	}
	if u.ID == "" {
		u.ID = s.genID(model.UserIDPrefix)
	}
	now := s.now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	u.UpdatedAt = now
	u.DisplayName = sanitizeDisplayName(u.DisplayName)
	if u.Status == "" {
		u.Status = model.UserActive
	}
	s.users[u.ID] = u
	s.username[u.Username] = u.ID
	s.afterWrite()
	return u, nil
}

func (s *MemoryStore) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.username[username]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	u, ok := s.users[id]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) GetUserByID(ctx context.Context, id string) (model.User, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) ListUsers(ctx context.Context, f model.UserFilter) ([]model.User, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		if f.Role != "" && u.Role != f.Role {
			continue
		}
		if f.Status != "" && u.Status.Normalize() != f.Status {
			continue
		}
		if f.ShelterID != "" && u.ShelterID != f.ShelterID {
			continue
		}
		if f.Query != "" {
			hay := u.Username + " " + u.DisplayName + " " + u.Phone + " " + u.City
			if !matchQuery(hay, f.Query) {
				continue
			}
		}
		out = append(out, u)
	}
	return out, nil
}

func (s *MemoryStore) UpdateUser(ctx context.Context, u model.User) (model.User, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.users[u.ID]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	if old.Username != u.Username {
		if _, exists := s.username[u.Username]; exists {
			return model.User{}, model.ErrAlreadyExists
		}
		delete(s.username, old.Username)
		s.username[u.Username] = u.ID
	}
	u.UpdatedAt = s.now()
	u.DisplayName = sanitizeDisplayName(u.DisplayName)
	s.users[u.ID] = u
	s.afterWrite()
	return u, nil
}

func (s *MemoryStore) CountUsers(ctx context.Context) (total, active, frozen int, err error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		total++
		switch u.Status.Normalize() {
		case model.UserActive:
			active++
		case model.UserFrozen:
			frozen++
		}
	}
	return total, active, frozen, nil
}

func (s *MemoryStore) CreateShelter(ctx context.Context, sh model.Shelter) (model.Shelter, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if sh.ID == "" {
		sh.ID = s.genID(model.ShelterIDPrefix)
	}
	now := s.now()
	if sh.CreatedAt.IsZero() {
		sh.CreatedAt = now
	}
	sh.UpdatedAt = now
	if sh.Status == "" {
		sh.Status = model.ShelterActive
	}
	s.shelters[sh.ID] = sh
	s.afterWrite()
	return sh, nil
}

func (s *MemoryStore) GetShelter(ctx context.Context, id string) (model.Shelter, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	sh, ok := s.shelters[id]
	if !ok {
		return model.Shelter{}, model.ErrNotFound
	}
	return sh, nil
}

func (s *MemoryStore) ListShelters(ctx context.Context, activeOnly bool) ([]model.Shelter, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Shelter, 0, len(s.shelters))
	for _, sh := range s.shelters {
		if activeOnly && !sh.IsActive() {
			continue
		}
		out = append(out, sh)
	}
	return out, nil
}

func (s *MemoryStore) UpdateShelter(ctx context.Context, sh model.Shelter) (model.Shelter, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.shelters[sh.ID]; !ok {
		return model.Shelter{}, model.ErrNotFound
	}
	sh.UpdatedAt = s.now()
	s.shelters[sh.ID] = sh
	s.afterWrite()
	return sh, nil
}

func (s *MemoryStore) ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string, now time.Time) (model.User, model.CreditLog, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok {
		return model.User{}, model.CreditLog{}, model.ErrNotFound
	}
	if now.IsZero() {
		now = s.now()
	}
	u.CreditScore = model.ClampCredit(u.CreditScore + delta)
	u.UpdatedAt = now
	log := model.CreditLog{
		ID:        s.genID(model.CreditLogIDPrefix),
		UserID:    userID,
		Delta:     delta,
		Score:     u.CreditScore,
		Reason:    reason,
		RelatedID: relatedID,
		Note:      strings.TrimSpace(note),
		CreatedAt: now,
	}
	s.users[u.ID] = u
	s.credits[log.ID] = log
	s.afterWrite()
	return u, log, nil
}

func (s *MemoryStore) CreateCreditLog(ctx context.Context, l model.CreditLog) (model.CreditLog, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if l.ID == "" {
		l.ID = s.genID(model.CreditLogIDPrefix)
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = s.now()
	}
	s.credits[l.ID] = l
	s.afterWrite()
	return l, nil
}

func (s *MemoryStore) ListCreditLogs(ctx context.Context, userID string) ([]model.CreditLog, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CreditLog, 0)
	for _, l := range s.credits {
		if userID != "" && l.UserID != userID {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

func (s *MemoryStore) CreateAuditLog(ctx context.Context, l model.AuditLog) (model.AuditLog, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if l.ID == "" {
		l.ID = s.genID(model.AuditLogIDPrefix)
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = s.now()
	}
	s.audits[l.ID] = l
	s.afterWrite()
	return l, nil
}

func (s *MemoryStore) ListAuditLogs(ctx context.Context, targetID string) ([]model.AuditLog, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.AuditLog, 0)
	for _, l := range s.audits {
		if targetID != "" && l.TargetID != targetID {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

func (s *MemoryStore) CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if n.ID == "" {
		n.ID = s.genID(model.NotificationIDPrefix)
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = s.now()
	}
	s.notifs[n.ID] = n
	s.afterWrite()
	return n, nil
}

func (s *MemoryStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool) ([]model.Notification, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Notification, 0)
	for _, n := range s.notifs {
		if n.UserID != userID {
			continue
		}
		if unreadOnly && !n.Unread() {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func (s *MemoryStore) GetNotification(ctx context.Context, id string) (model.Notification, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notifs[id]
	if !ok {
		return model.Notification{}, model.ErrNotFound
	}
	return n, nil
}

func (s *MemoryStore) UpdateNotification(ctx context.Context, n model.Notification) (model.Notification, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notifs[n.ID]; !ok {
		return model.Notification{}, model.ErrNotFound
	}
	s.notifs[n.ID] = n
	s.afterWrite()
	return n, nil
}

func (s *MemoryStore) MarkAllNotificationsRead(ctx context.Context, userID string, at time.Time) (int, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, item := range s.notifs {
		if item.UserID != userID || item.ReadAt != nil {
			continue
		}
		t := at
		item.ReadAt = &t
		s.notifs[id] = item
		n++
	}
	if n > 0 {
		s.afterWrite()
	}
	return n, nil
}

func (s *MemoryStore) CountUnreadNotifications(ctx context.Context, userID string) (int, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, item := range s.notifs {
		if item.UserID == userID && item.Unread() {
			n++
		}
	}
	return n, nil
}
