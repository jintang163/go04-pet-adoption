package store

import (
	"context"
	"sort"

	"go04-pet-adoption/internal/model"
)

func (s *MemoryStore) CreateVisit(ctx context.Context, v model.Visit) (model.Visit, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.ID == "" {
		v.ID = s.genID(model.VisitIDPrefix)
	}
	now := s.now()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	v.UpdatedAt = now
	if v.Status == "" {
		v.Status = model.VisitScheduled
	}
	s.visits[v.ID] = v
	s.afterWrite()
	return v, nil
}

func (s *MemoryStore) GetVisit(ctx context.Context, id string) (model.Visit, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.visits[id]
	if !ok {
		return model.Visit{}, model.ErrNotFound
	}
	return v, nil
}

func (s *MemoryStore) ListVisits(ctx context.Context, f model.VisitFilter) ([]model.Visit, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Visit, 0)
	for _, v := range s.visits {
		if f.PetID != "" && v.PetID != f.PetID {
			continue
		}
		if f.ApplicationID != "" && v.ApplicationID != f.ApplicationID {
			continue
		}
		if f.AdopterID != "" && v.AdopterID != f.AdopterID {
			continue
		}
		if f.StaffID != "" && v.StaffID != f.StaffID {
			continue
		}
		if f.ShelterID != "" && v.ShelterID != f.ShelterID {
			continue
		}
		if f.Type != "" && v.Type != f.Type {
			continue
		}
		if f.Status != "" && v.Status != f.Status {
			continue
		}
		if f.DueBefore != nil && v.ScheduledAt.After(*f.DueBefore) {
			continue
		}
		if f.DueAfter != nil && v.ScheduledAt.Before(*f.DueAfter) {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScheduledAt.Equal(out[j].ScheduledAt) {
			return out[i].Sequence < out[j].Sequence
		}
		return out[i].ScheduledAt.Before(out[j].ScheduledAt)
	})
	return out, nil
}

func (s *MemoryStore) UpdateVisit(ctx context.Context, v model.Visit) (model.Visit, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.visits[v.ID]; !ok {
		return model.Visit{}, model.ErrNotFound
	}
	v.UpdatedAt = s.now()
	s.visits[v.ID] = v
	s.afterWrite()
	return v, nil
}

func (s *MemoryStore) CountVisits(ctx context.Context) (scheduled, completed, missed int, err error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.visits {
		switch v.Status {
		case model.VisitScheduled:
			scheduled++
		case model.VisitCompleted:
			completed++
		case model.VisitMissed:
			missed++
		}
	}
	return scheduled, completed, missed, nil
}
