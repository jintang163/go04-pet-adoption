package service

import (
	"context"
	"strings"
	"time"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/policy"
	"go04-pet-adoption/internal/store"
)

type VisitService struct {
	store  store.Store
	notify *NotifyService
	clock  Clock
}

func NewVisitService(s store.Store, notify *NotifyService, clock Clock) *VisitService {
	return &VisitService{store: s, notify: notify, clock: clock}
}

func (svc *VisitService) Schedule(ctx context.Context, actor model.User, in model.ScheduleVisitInput) (model.Visit, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Visit{}, err
	}
	if !actor.IsStaff() && !actor.IsAdmin() {
		return model.Visit{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Visit{}, err
	}
	app, err := svc.store.GetApplication(ctx, in.ApplicationID)
	if err != nil {
		return model.Visit{}, err
	}
	p, err := svc.store.GetPet(ctx, app.PetID)
	if err != nil {
		return model.Visit{}, err
	}
	if in.PetID != "" && in.PetID != p.ID {
		return model.Visit{}, model.ErrValidation
	}
	if !canManagePet(actor, p) {
		return model.Visit{}, model.ErrNotPetStaff
	}
	switch in.Type {
	case model.VisitHomeCheck:
		if app.Status != model.AppPending && app.Status != model.AppUnderReview {
			return model.Visit{}, model.ErrInvalidAppStatus
		}
	case model.VisitExtra:
		if app.Status != model.AppCompleted || p.Status != model.PetAdopted {
			return model.Visit{}, model.ErrPetNotAdopted
		}
	default:
		return model.Visit{}, model.ErrInvalidVisitType
	}
	if in.ScheduledAt.Before(svc.clock.Now().Add(-1 * time.Hour)) {
		return model.Visit{}, model.ErrValidation
	}
	staffID := in.StaffID
	if staffID == "" {
		staffID = actor.ID
	}
	v := model.Visit{
		PetID:         p.ID,
		ApplicationID: app.ID,
		AdopterID:     app.ApplicantID,
		StaffID:       staffID,
		ShelterID:     p.ShelterID,
		Type:          in.Type,
		Status:        model.VisitScheduled,
		ScheduledAt:   in.ScheduledAt,
		Location:      in.Location,
	}
	created, err := svc.store.CreateVisit(ctx, v)
	if err != nil {
		return model.Visit{}, err
	}
	if in.Type == model.VisitHomeCheck {
		app.HomeCheckID = created.ID
		if app.Status == model.AppPending {
			app.Status = model.AppUnderReview
			app.ReviewerID = actor.ID
		}
		_, _ = svc.store.UpdateApplication(ctx, app)
	}
	_ = svc.notify.Push(ctx, app.ApplicantID, model.NotifyVisitDue, "回访已安排", visitTitle(in.Type), created.ID)
	return created, nil
}

func (svc *VisitService) Complete(ctx context.Context, actor model.User, id string, in model.CompleteVisitInput) (model.Visit, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Visit{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Visit{}, err
	}
	v, err := svc.store.GetVisit(ctx, id)
	if err != nil {
		return model.Visit{}, err
	}
	p, err := svc.store.GetPet(ctx, v.PetID)
	if err != nil {
		return model.Visit{}, err
	}
	if !canManagePet(actor, p) {
		return model.Visit{}, model.ErrNotPetStaff
	}
	if v.Status != model.VisitScheduled {
		return model.Visit{}, model.ErrInvalidVisitStatus
	}
	now := svc.clock.Now()
	t := now
	v.Status = model.VisitCompleted
	v.CompletedAt = &t
	v.LivingScore = in.LivingScore
	v.HealthScore = in.HealthScore
	v.BehaviorScore = in.BehaviorScore
	v.RiskFlag = in.RiskFlag
	v.Notes = in.Notes
	v.Issues = in.Issues
	v.Suggestion = in.Suggestion
	v.FollowUpNeeded = in.FollowUpNeeded
	v.StaffID = actor.ID
	if in.Location != "" {
		v.Location = in.Location
	}
	updated, err := svc.store.UpdateVisit(ctx, v)
	if err != nil {
		return model.Visit{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditVisit, updated.ID, string(updated.Type), now)
	_ = svc.notify.Push(ctx, v.AdopterID, model.NotifyVisitDone, "回访已完成", in.Notes, updated.ID)
	if in.RiskFlag {
		admins, _ := svc.store.ListUsers(ctx, model.UserFilter{Role: model.RoleAdmin})
		for _, ad := range admins {
			_ = svc.notify.Push(ctx, ad.ID, model.NotifyRisk, "回访发现风险", p.Name+" / "+in.Issues, updated.ID)
		}
	}
	if in.FollowUpNeeded && updated.Type != model.VisitHomeCheck {
		_, _ = svc.store.CreateVisit(ctx, model.Visit{
			PetID:         v.PetID,
			ApplicationID: v.ApplicationID,
			AdopterID:     v.AdopterID,
			StaffID:       actor.ID,
			ShelterID:     v.ShelterID,
			Type:          model.VisitExtra,
			Status:        model.VisitScheduled,
			ScheduledAt:   now.Add(7 * 24 * time.Hour),
			Location:      v.Location,
		})
	}
	if updated.Type == model.VisitFollowup && !in.RiskFlag && updated.AverageScore() >= 4 {
		visits, _ := svc.store.ListVisits(ctx, model.VisitFilter{ApplicationID: v.ApplicationID})
		if policy.AllFollowupsGood(visits) {
			_, _, _ = svc.store.ApplyCredit(ctx, v.AdopterID, policy.DeltaGoodVisit, model.CreditGoodVisit, v.ApplicationID, "随访全部良好", now)
		}
	}
	return updated, nil
}

func (svc *VisitService) Miss(ctx context.Context, actor model.User, id, note string) (model.Visit, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Visit{}, err
	}
	v, err := svc.store.GetVisit(ctx, id)
	if err != nil {
		return model.Visit{}, err
	}
	p, err := svc.store.GetPet(ctx, v.PetID)
	if err != nil {
		return model.Visit{}, err
	}
	if !canManagePet(actor, p) {
		return model.Visit{}, model.ErrNotPetStaff
	}
	if v.Status != model.VisitScheduled {
		return model.Visit{}, model.ErrInvalidVisitStatus
	}
	now := svc.clock.Now()
	if !policy.VisitOverdue(v.ScheduledAt, now) && !actor.IsAdmin() {
		return model.Visit{}, model.ErrVisitNotDue
	}
	v.Status = model.VisitMissed
	v.Notes = strings.TrimSpace(note)
	updated, err := svc.store.UpdateVisit(ctx, v)
	if err != nil {
		return model.Visit{}, err
	}
	_, _, _ = svc.store.ApplyCredit(ctx, v.AdopterID, policy.DeltaMissedVisit, model.CreditMissedVisit, v.ID, "缺访", now)
	_ = svc.notify.Push(ctx, v.AdopterID, model.NotifyVisitMissed, "回访缺席", "请尽快联系救助站补访", updated.ID)
	visits, _ := svc.store.ListVisits(ctx, model.VisitFilter{ApplicationID: v.ApplicationID})
	if policy.ConsecutiveMissed(visits) >= policy.ConsecutiveMissedLimit {
		_, _ = svc.store.CreateVisit(ctx, model.Visit{
			PetID:         v.PetID,
			ApplicationID: v.ApplicationID,
			AdopterID:     v.AdopterID,
			StaffID:       actor.ID,
			ShelterID:     v.ShelterID,
			Type:          model.VisitExtra,
			Status:        model.VisitScheduled,
			ScheduledAt:   now.Add(3 * 24 * time.Hour),
		})
	}
	return updated, nil
}

func (svc *VisitService) Cancel(ctx context.Context, actor model.User, id string) (model.Visit, error) {
	v, err := svc.store.GetVisit(ctx, id)
	if err != nil {
		return model.Visit{}, err
	}
	p, err := svc.store.GetPet(ctx, v.PetID)
	if err != nil {
		return model.Visit{}, err
	}
	if !canManagePet(actor, p) {
		return model.Visit{}, model.ErrNotPetStaff
	}
	if v.Status != model.VisitScheduled {
		return model.Visit{}, model.ErrInvalidVisitStatus
	}
	v.Status = model.VisitCancelled
	return svc.store.UpdateVisit(ctx, v)
}

func (svc *VisitService) Comment(ctx context.Context, actor model.User, id string, in model.VisitCommentInput) (model.Visit, error) {
	if err := in.Validate(); err != nil {
		return model.Visit{}, err
	}
	v, err := svc.store.GetVisit(ctx, id)
	if err != nil {
		return model.Visit{}, err
	}
	if v.AdopterID != actor.ID && !actor.IsAdmin() {
		return model.Visit{}, model.ErrForbidden
	}
	v.AdopterComment = strings.TrimSpace(in.Comment)
	return svc.store.UpdateVisit(ctx, v)
}

func (svc *VisitService) Get(ctx context.Context, actor model.User, id string) (model.VisitView, error) {
	v, err := svc.store.GetVisit(ctx, id)
	if err != nil {
		return model.VisitView{}, err
	}
	if !svc.canSee(actor, v) {
		return model.VisitView{}, model.ErrForbidden
	}
	return svc.toView(ctx, v), nil
}

func (svc *VisitService) List(ctx context.Context, actor model.User, f model.VisitFilter) ([]model.VisitView, error) {
	if actor.IsAdopter() && !actor.IsAdmin() && !actor.IsStaff() {
		f.AdopterID = actor.ID
	}
	if actor.IsStaff() && !actor.IsAdmin() && f.ShelterID == "" {
		f.ShelterID = actor.ShelterID
	}
	visits, err := svc.store.ListVisits(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]model.VisitView, 0, len(visits))
	for _, v := range visits {
		if !svc.canSee(actor, v) {
			continue
		}
		out = append(out, svc.toView(ctx, v))
	}
	return out, nil
}

func (svc *VisitService) canSee(actor model.User, v model.Visit) bool {
	if actor.IsAdmin() {
		return true
	}
	if v.AdopterID == actor.ID {
		return true
	}
	if actor.IsStaff() && actor.ShelterID == v.ShelterID {
		return true
	}
	return false
}

func (svc *VisitService) toView(ctx context.Context, v model.Visit) model.VisitView {
	vw := model.VisitView{Visit: v}
	if p, err := svc.store.GetPet(ctx, v.PetID); err == nil {
		vw.PetName = p.Name
	}
	if u, err := svc.store.GetUserByID(ctx, v.AdopterID); err == nil {
		vw.AdopterName = u.DisplayName
	}
	if v.StaffID != "" {
		if u, err := svc.store.GetUserByID(ctx, v.StaffID); err == nil {
			vw.StaffName = u.DisplayName
		}
	}
	return vw
}

func visitTitle(t model.VisitType) string {
	switch t {
	case model.VisitHomeCheck:
		return "家访"
	case model.VisitExtra:
		return "加访"
	default:
		return "随访"
	}
}
