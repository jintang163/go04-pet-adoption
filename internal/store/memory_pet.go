package store

import (
	"context"

	"go04-pet-adoption/internal/model"
)

func (s *MemoryStore) CreatePet(ctx context.Context, p model.Pet) (model.Pet, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = s.genID(model.PetIDPrefix)
	}
	now := s.now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = model.PetDraft
	}
	p.Personality = cloneStrings(p.Personality)
	p.Photos = cloneStrings(p.Photos)
	s.pets[p.ID] = p
	s.afterWrite()
	return p, nil
}

func (s *MemoryStore) GetPet(ctx context.Context, id string) (model.Pet, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.pets[id]
	if !ok {
		return model.Pet{}, model.ErrNotFound
	}
	p.Personality = cloneStrings(p.Personality)
	p.Photos = cloneStrings(p.Photos)
	return p, nil
}

func (s *MemoryStore) ListPets(ctx context.Context, f model.PetFilter) ([]model.Pet, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Pet, 0)
	for _, p := range s.pets {
		if f.Status != "" && p.Status != f.Status {
			continue
		}
		if f.Species != "" && p.Species != f.Species {
			continue
		}
		if f.Size != "" && p.Size != f.Size {
			continue
		}
		if f.ShelterID != "" && p.ShelterID != f.ShelterID {
			continue
		}
		if f.StaffID != "" && p.StaffID != f.StaffID {
			continue
		}
		if f.AdoptedBy != "" && p.AdoptedBy != f.AdoptedBy {
			continue
		}
		if f.Sterilized != nil && p.Sterilized != *f.Sterilized {
			continue
		}
		if f.Vaccinated != nil && p.Vaccinated != *f.Vaccinated {
			continue
		}
		if f.SpecialNeeds != nil && p.SpecialNeeds != *f.SpecialNeeds {
			continue
		}
		if f.AllowApartment != nil && p.AllowApartment != *f.AllowApartment {
			continue
		}
		if f.Query != "" {
			hay := p.Name + " " + p.Breed + " " + p.Color + " " + p.Story + " " + string(p.Species)
			if !matchQuery(hay, f.Query) {
				continue
			}
		}
		cp := p
		cp.Personality = cloneStrings(p.Personality)
		cp.Photos = cloneStrings(p.Photos)
		out = append(out, cp)
	}
	return out, nil
}

func (s *MemoryStore) UpdatePet(ctx context.Context, p model.Pet) (model.Pet, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pets[p.ID]; !ok {
		return model.Pet{}, model.ErrNotFound
	}
	p.UpdatedAt = s.now()
	p.Personality = cloneStrings(p.Personality)
	p.Photos = cloneStrings(p.Photos)
	s.pets[p.ID] = p
	s.afterWrite()
	return p, nil
}

func (s *MemoryStore) CountPetsByStatus(ctx context.Context, shelterID string) (map[model.PetStatus]int, int, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := map[model.PetStatus]int{}
	total := 0
	for _, p := range s.pets {
		if shelterID != "" && p.ShelterID != shelterID {
			continue
		}
		m[p.Status]++
		total++
	}
	return m, total, nil
}

func (s *MemoryStore) CreateHealth(ctx context.Context, h model.HealthRecord) (model.HealthRecord, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pets[h.PetID]; !ok {
		return model.HealthRecord{}, model.ErrNotFound
	}
	if h.ID == "" {
		h.ID = s.genID(model.HealthIDPrefix)
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = s.now()
	}
	s.health[h.ID] = h
	s.afterWrite()
	return h, nil
}

func (s *MemoryStore) ListHealthByPet(ctx context.Context, petID string) ([]model.HealthRecord, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.HealthRecord, 0)
	for _, h := range s.health {
		if h.PetID == petID {
			out = append(out, h)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateFavorite(ctx context.Context, f model.Favorite) (model.Favorite, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	key := favKey(f.UserID, f.PetID)
	if _, ok := s.favIdx[key]; ok {
		return model.Favorite{}, model.ErrDuplicateFavorite
	}
	if f.ID == "" {
		f.ID = s.genID(model.FavoriteIDPrefix)
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = s.now()
	}
	s.favorites[f.ID] = f
	s.favIdx[key] = f.ID
	s.afterWrite()
	return f, nil
}

func (s *MemoryStore) DeleteFavorite(ctx context.Context, userID, petID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	key := favKey(userID, petID)
	id, ok := s.favIdx[key]
	if !ok {
		return model.ErrNotFound
	}
	delete(s.favorites, id)
	delete(s.favIdx, key)
	s.afterWrite()
	return nil
}

func (s *MemoryStore) GetFavorite(ctx context.Context, userID, petID string) (model.Favorite, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.favIdx[favKey(userID, petID)]
	if !ok {
		return model.Favorite{}, model.ErrNotFound
	}
	return s.favorites[id], nil
}

func (s *MemoryStore) ListFavoritesByUser(ctx context.Context, userID string) ([]model.Favorite, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Favorite, 0)
	for _, f := range s.favorites {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (s *MemoryStore) CreateInquiry(ctx context.Context, q model.Inquiry) (model.Inquiry, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pets[q.PetID]; !ok {
		return model.Inquiry{}, model.ErrNotFound
	}
	if q.ID == "" {
		q.ID = s.genID(model.InquiryIDPrefix)
	}
	if q.CreatedAt.IsZero() {
		q.CreatedAt = s.now()
	}
	s.inquiries[q.ID] = q
	s.afterWrite()
	return q, nil
}

func (s *MemoryStore) GetInquiry(ctx context.Context, id string) (model.Inquiry, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	q, ok := s.inquiries[id]
	if !ok {
		return model.Inquiry{}, model.ErrNotFound
	}
	return q, nil
}

func (s *MemoryStore) ListInquiriesByPet(ctx context.Context, petID string) ([]model.Inquiry, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Inquiry, 0)
	for _, q := range s.inquiries {
		if q.PetID == petID {
			out = append(out, q)
		}
	}
	return out, nil
}

func (s *MemoryStore) UpdateInquiry(ctx context.Context, q model.Inquiry) (model.Inquiry, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.inquiries[q.ID]; !ok {
		return model.Inquiry{}, model.ErrNotFound
	}
	s.inquiries[q.ID] = q
	s.afterWrite()
	return q, nil
}

func (s *MemoryStore) DeleteInquiry(ctx context.Context, id string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.inquiries[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.inquiries, id)
	s.afterWrite()
	return nil
}
