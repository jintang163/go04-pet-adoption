package store

import (
	"context"
	"sort"
	"strings"
	"time"

	"go04-pet-adoption/internal/model"
)

// CreateApplication 在单次写锁内原子地完成校验与插入。
//
// 校验顺序与返回错误与上层 Apply 保持一致：
//  1. 同一宠物+申请人的重复活跃申请 → ErrAlreadyApplied
//  2. 申请人活跃申请数已达上限 → ErrTooManyApplications（maxActive<=0 时不校验）
//
// 将数量上限校验放在写锁内，可避免上层先读后写时两个并发请求同时通过上限
// 检查、随后各自插入，绕过"待处理申请上限"的竞态。
func (s *MemoryStore) CreateApplication(ctx context.Context, a model.Application, maxActive int) (model.Application, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	key := appKey(a.PetID, a.ApplicantID)
	if id, ok := s.appIdx[key]; ok {
		old := s.apps[id]
		if old.Status.IsActive() {
			return model.Application{}, model.ErrAlreadyApplied
		}
	}
	if maxActive > 0 {
		n := 0
		for _, ex := range s.apps {
			if ex.ApplicantID == a.ApplicantID && ex.Status.CountsTowardLimit() {
				n++
			}
		}
		if n >= maxActive {
			return model.Application{}, model.ErrTooManyApplications
		}
	}
	if a.ID == "" {
		a.ID = s.genID(model.ApplicationIDPrefix)
	}
	now := s.now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	s.apps[a.ID] = a
	s.appIdx[key] = a.ID
	s.afterWrite()
	return a, nil
}

func (s *MemoryStore) GetApplication(ctx context.Context, id string) (model.Application, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.apps[id]
	if !ok {
		return model.Application{}, model.ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) GetApplicationByPetApplicant(ctx context.Context, petID, applicantID string) (model.Application, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.appIdx[appKey(petID, applicantID)]
	if !ok {
		return model.Application{}, model.ErrNotFound
	}
	return s.apps[id], nil
}

func (s *MemoryStore) ListApplications(ctx context.Context, f model.ApplicationFilter) ([]model.Application, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Application, 0)
	for _, a := range s.apps {
		if f.PetID != "" && a.PetID != f.PetID {
			continue
		}
		if f.ApplicantID != "" && a.ApplicantID != f.ApplicantID {
			continue
		}
		if f.ShelterID != "" && a.ShelterID != f.ShelterID {
			continue
		}
		if f.Status != "" && a.Status != f.Status {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryStore) UpdateApplication(ctx context.Context, a model.Application) (model.Application, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[a.ID]; !ok {
		return model.Application{}, model.ErrNotFound
	}
	a.UpdatedAt = s.now()
	s.apps[a.ID] = a
	s.appIdx[appKey(a.PetID, a.ApplicantID)] = a.ID
	s.afterWrite()
	return a, nil
}

func (s *MemoryStore) CountActiveApplicationsByApplicant(ctx context.Context, applicantID string) (int, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, a := range s.apps {
		if a.ApplicantID == applicantID && a.Status.CountsTowardLimit() {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) CountApplications(ctx context.Context) (total, approved, completed, returned int, err error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.apps {
		total++
		if a.Status == model.AppApproved || a.Status == model.AppCompleted {
			approved++
		}
		if a.Status == model.AppCompleted {
			completed++
		}
		if a.ReturnedAt != nil {
			returned++
		}
	}
	return total, approved, completed, returned, nil
}

func (s *MemoryStore) waitlistOthersLocked(petID, exceptID string, now time.Time) []model.Application {
	type item struct {
		id string
		at time.Time
	}
	cands := make([]item, 0)
	for id, a := range s.apps {
		if a.PetID != petID || a.ID == exceptID {
			continue
		}
		if a.Status == model.AppPending || a.Status == model.AppUnderReview {
			cands = append(cands, item{id: id, at: a.CreatedAt})
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].at.Before(cands[j].at) })
	out := make([]model.Application, 0, len(cands))
	for i, c := range cands {
		a := s.apps[c.id]
		a.Status = model.AppWaitlisted
		a.WaitlistRank = i + 1
		a.UpdatedAt = now
		s.apps[c.id] = a
		out = append(out, a)
	}
	return out
}

func (s *MemoryStore) unlockPetAndPromoteLocked(pet model.Pet, now time.Time) (model.Pet, []model.Application) {
	pet.Status = model.PetPublished
	pet.ReservedAppID = ""
	pet.UpdatedAt = now
	s.pets[pet.ID] = pet

	type item struct {
		id string
		rk int
		at time.Time
	}
	cands := make([]item, 0)
	for id, a := range s.apps {
		if a.PetID != pet.ID || a.Status != model.AppWaitlisted {
			continue
		}
		cands = append(cands, item{id: id, rk: a.WaitlistRank, at: a.CreatedAt})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].rk == cands[j].rk {
			return cands[i].at.Before(cands[j].at)
		}
		if cands[i].rk == 0 {
			return false
		}
		if cands[j].rk == 0 {
			return true
		}
		return cands[i].rk < cands[j].rk
	})
	promoted := make([]model.Application, 0)
	for i, c := range cands {
		a := s.apps[c.id]
		if i == 0 {
			a.Status = model.AppPending
			a.WaitlistRank = 0
		} else {
			a.WaitlistRank = i
		}
		a.UpdatedAt = now
		s.apps[c.id] = a
		if i == 0 {
			promoted = append(promoted, a)
		}
	}
	return pet, promoted
}

func (s *MemoryStore) ApproveApplication(ctx context.Context, appID, reviewerID, note string, now time.Time) (model.Application, model.Pet, []model.Application, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[appID]
	if !ok {
		return model.Application{}, model.Pet{}, nil, model.ErrNotFound
	}
	if !a.Status.CanApprove() {
		return model.Application{}, model.Pet{}, nil, model.ErrInvalidAppStatus
	}
	p, ok := s.pets[a.PetID]
	if !ok {
		return model.Application{}, model.Pet{}, nil, model.ErrNotFound
	}
	if p.Status != model.PetPublished && !(p.Status == model.PetReserved && p.ReservedAppID == a.ID) {
		if p.Status == model.PetReserved {
			return model.Application{}, model.Pet{}, nil, model.ErrPetAlreadyReserved
		}
		return model.Application{}, model.Pet{}, nil, model.ErrPetNotReservable
	}
	if now.IsZero() {
		now = s.now()
	}
	a.Status = model.AppApproved
	a.ReviewerID = reviewerID
	a.ReviewNote = note
	a.WaitlistRank = 0
	a.UpdatedAt = now
	t := now
	a.ReviewedAt = &t
	p.Status = model.PetReserved
	p.ReservedAppID = a.ID
	p.UpdatedAt = now
	s.apps[a.ID] = a
	s.pets[p.ID] = p
	waitlisted := s.waitlistOthersLocked(p.ID, a.ID, now)
	s.afterWrite()
	return a, p, waitlisted, nil
}

func (s *MemoryStore) WithdrawApprovedApplication(ctx context.Context, appID, applicantID, actorID string, creditDelta int, creditReason model.CreditReason, creditNote string, now time.Time) (model.Application, *model.Pet, []model.Application, model.CreditLog, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[appID]
	if !ok {
		return model.Application{}, nil, nil, model.CreditLog{}, model.ErrNotFound
	}
	if a.Status != model.AppApproved {
		return model.Application{}, nil, nil, model.CreditLog{}, model.ErrInvalidAppStatus
	}
	p, ok := s.pets[a.PetID]
	if !ok {
		return model.Application{}, nil, nil, model.CreditLog{}, model.ErrNotFound
	}
	if now.IsZero() {
		now = s.now()
	}
	u, ok := s.users[applicantID]
	if !ok {
		return model.Application{}, nil, nil, model.CreditLog{}, model.ErrNotFound
	}
	u.CreditScore = model.ClampCredit(u.CreditScore + creditDelta)
	u.UpdatedAt = now
	log := model.CreditLog{
		ID: s.genID(model.CreditLogIDPrefix), UserID: applicantID, Delta: creditDelta,
		Score: u.CreditScore, Reason: creditReason, RelatedID: a.ID,
		Note: strings.TrimSpace(creditNote), CreatedAt: now,
	}
	s.users[u.ID] = u
	s.credits[log.ID] = log
	a.Status = model.AppWithdrawn
	a.ReviewerID = actorID
	a.RejectReason = "申请人撤回"
	a.UpdatedAt = now
	t := now
	a.ReviewedAt = &t
	s.apps[a.ID] = a
	var petPtr *model.Pet
	var promoted []model.Application
	if p.ReservedAppID == a.ID {
		np, pr := s.unlockPetAndPromoteLocked(p, now)
		petPtr = &np
		promoted = pr
	}
	s.afterWrite()
	return a, petPtr, promoted, log, nil
}

func (s *MemoryStore) RejectApplication(ctx context.Context, appID, reviewerID, reason string, now time.Time) (model.Application, *model.Pet, []model.Application, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[appID]
	if !ok {
		return model.Application{}, nil, nil, model.ErrNotFound
	}
	if a.Status == model.AppCompleted || a.Status == model.AppRejected || a.Status == model.AppWithdrawn {
		return model.Application{}, nil, nil, model.ErrInvalidAppStatus
	}
	p, ok := s.pets[a.PetID]
	if !ok {
		return model.Application{}, nil, nil, model.ErrNotFound
	}
	if now.IsZero() {
		now = s.now()
	}
	wasApproved := a.Status == model.AppApproved
	a.Status = model.AppRejected
	a.ReviewerID = reviewerID
	a.RejectReason = reason
	a.UpdatedAt = now
	t := now
	a.ReviewedAt = &t
	s.apps[a.ID] = a
	var petPtr *model.Pet
	var promoted []model.Application
	if wasApproved && p.ReservedAppID == a.ID {
		np, pr := s.unlockPetAndPromoteLocked(p, now)
		petPtr = &np
		promoted = pr
	}
	s.afterWrite()
	return a, petPtr, promoted, nil
}

func (s *MemoryStore) HandoverAdoption(ctx context.Context, appID, staffID, note string, visits []model.Visit, now time.Time) (model.Application, model.Pet, []model.Visit, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.apps[appID]
	if !ok {
		return model.Application{}, model.Pet{}, nil, model.ErrNotFound
	}
	if a.Status != model.AppApproved {
		return model.Application{}, model.Pet{}, nil, model.ErrHandoverNotApproved
	}
	p, ok := s.pets[a.PetID]
	if !ok {
		return model.Application{}, model.Pet{}, nil, model.ErrNotFound
	}
	if p.Status != model.PetReserved || p.ReservedAppID != a.ID {
		return model.Application{}, model.Pet{}, nil, model.ErrPetNotReservable
	}
	if now.IsZero() {
		now = s.now()
	}
	a.Status = model.AppCompleted
	a.HandoverNote = note
	a.UpdatedAt = now
	ht := now
	a.HandoverAt = &ht
	p.Status = model.PetAdopted
	p.AdoptedBy = a.ApplicantID
	p.AdoptedAt = &ht
	p.UpdatedAt = now
	s.apps[a.ID] = a
	s.pets[p.ID] = p
	created := make([]model.Visit, 0, len(visits))
	for i, v := range visits {
		if v.ID == "" {
			v.ID = s.genID(model.VisitIDPrefix)
		}
		v.PetID = p.ID
		v.ApplicationID = a.ID
		v.AdopterID = a.ApplicantID
		v.ShelterID = p.ShelterID
		if v.StaffID == "" {
			v.StaffID = staffID
		}
		if v.Status == "" {
			v.Status = model.VisitScheduled
		}
		if v.CreatedAt.IsZero() {
			v.CreatedAt = now
		}
		v.UpdatedAt = now
		v.Sequence = i + 1
		s.visits[v.ID] = v
		created = append(created, v)
	}
	if u, ok := s.users[a.ApplicantID]; ok {
		u.AdoptCount++
		u.UpdatedAt = now
		s.users[u.ID] = u
	}
	s.afterWrite()
	return a, p, created, nil
}

func (s *MemoryStore) ReturnAdoption(ctx context.Context, appID, actorID, reason string, medical bool, now time.Time) (model.Application, model.Pet, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = actorID
	a, ok := s.apps[appID]
	if !ok {
		return model.Application{}, model.Pet{}, model.ErrNotFound
	}
	if a.Status != model.AppCompleted {
		return model.Application{}, model.Pet{}, model.ErrReturnNotAllowed
	}
	if a.ReturnedAt != nil {
		return model.Application{}, model.Pet{}, model.ErrReturnNotAllowed
	}
	p, ok := s.pets[a.PetID]
	if !ok {
		return model.Application{}, model.Pet{}, model.ErrNotFound
	}
	if p.Status != model.PetAdopted {
		return model.Application{}, model.Pet{}, model.ErrPetNotAdopted
	}
	if now.IsZero() {
		now = s.now()
	}
	rt := now
	a.ReturnedAt = &rt
	a.ReturnReason = reason
	a.ReturnMedical = medical
	a.UpdatedAt = now
	if medical {
		p.Status = model.PetUnavailable
		p.UnavailableNote = "退养医疗留置: " + reason
	} else {
		p.Status = model.PetPublished
		p.UnavailableNote = ""
		pt := now
		p.PublishedAt = &pt
	}
	p.ReservedAppID = ""
	p.AdoptedBy = ""
	p.AdoptedAt = nil
	p.ReturnedAt = &rt
	p.UpdatedAt = now
	s.apps[a.ID] = a
	s.pets[p.ID] = p
	if u, ok := s.users[a.ApplicantID]; ok {
		u.ReturnCount++
		u.UpdatedAt = now
		s.users[u.ID] = u
	}
	s.afterWrite()
	return a, p, nil
}
