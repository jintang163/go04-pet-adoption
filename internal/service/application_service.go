package service

import (
	"context"
	"strings"
	"time"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/policy"
	"go04-pet-adoption/internal/store"
)

type ApplicationService struct {
	store   store.Store
	notify  *NotifyService
	clock   Clock
	maxPend int
}

func NewApplicationService(s store.Store, notify *NotifyService, clock Clock, maxPend int) *ApplicationService {
	if maxPend <= 0 {
		maxPend = policy.DefaultMaxPendingApplications
	}
	return &ApplicationService{store: s, notify: notify, clock: clock, maxPend: maxPend}
}

func (svc *ApplicationService) Apply(ctx context.Context, actor model.User, petID string, in model.ApplyInput) (model.Application, error) {
	if err := actor.CanApply(); err != nil {
		return model.Application{}, err
	}
	if !actor.IsAdopter() && !actor.IsAdmin() {
		return model.Application{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Application{}, err
	}
	p, err := svc.store.GetPet(ctx, petID)
	if err != nil {
		return model.Application{}, err
	}
	if !p.Status.CanReceiveApplication() {
		return model.Application{}, model.ErrPetNotPublished
	}
	if p.StaffID == actor.ID {
		return model.Application{}, model.ErrCannotApplyOwnPet
	}
	if err := policy.ApplyMatches(p, in, actor.AgeYears); err != nil {
		return model.Application{}, err
	}
	if !policy.ExperienceOK(p, in.Experience) {
		return model.Application{}, model.ErrRequirementNotMet
	}
	if !policy.LargeDogNeedsSpace(p, in.Housing, in.AreaSqm) {
		return model.Application{}, model.ErrRequirementNotMet
	}
	// 活跃申请数量上限与重复申请校验在 CreateApplication 内部完成（同一把写锁），
	// 避免在此处先读后写造成并发请求同时通过上限检查的竞态。
	now := svc.clock.Now()
	app := model.Application{
		PetID:        p.ID,
		ApplicantID:  actor.ID,
		ShelterID:    p.ShelterID,
		Status:       model.AppPending,
		Housing:      in.Housing,
		AreaSqm:      in.AreaSqm,
		HasChildren:  in.HasChildren,
		HasOtherPets: in.HasOtherPets,
		HoursAlone:   in.HoursAlone,
		Experience:   in.Experience,
		Phone:        in.Phone,
		Intro:        in.Intro,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	created, err := svc.store.CreateApplication(ctx, app, svc.maxPend)
	if err != nil {
		return model.Application{}, err
	}
	if staff, err := svc.store.GetUserByID(ctx, p.StaffID); err == nil {
		_ = svc.notify.Push(ctx, staff.ID, model.NotifyGeneric, "新的领养申请", actor.DisplayName+" 申请领养 "+p.Name, created.ID)
	}
	return created, nil
}

func (svc *ApplicationService) Get(ctx context.Context, actor model.User, id string) (model.ApplicationView, error) {
	a, err := svc.store.GetApplication(ctx, id)
	if err != nil {
		return model.ApplicationView{}, err
	}
	if !svc.canSee(actor, a) {
		return model.ApplicationView{}, model.ErrForbidden
	}
	return svc.toView(ctx, a), nil
}

func (svc *ApplicationService) ListMine(ctx context.Context, actor model.User) ([]model.ApplicationView, error) {
	apps, err := svc.store.ListApplications(ctx, model.ApplicationFilter{ApplicantID: actor.ID})
	if err != nil {
		return nil, err
	}
	return svc.toViews(ctx, apps), nil
}

func (svc *ApplicationService) ListByPet(ctx context.Context, actor model.User, petID string) ([]model.ApplicationView, error) {
	p, err := svc.store.GetPet(ctx, petID)
	if err != nil {
		return nil, err
	}
	if !canManagePet(actor, p) {
		return nil, model.ErrNotPetStaff
	}
	apps, err := svc.store.ListApplications(ctx, model.ApplicationFilter{PetID: petID})
	if err != nil {
		return nil, err
	}
	return svc.toViews(ctx, apps), nil
}

func (svc *ApplicationService) ListForStaff(ctx context.Context, actor model.User, status model.ApplicationStatus) ([]model.ApplicationView, error) {
	if !actor.IsStaff() && !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	f := model.ApplicationFilter{Status: status}
	if actor.IsStaff() && !actor.IsAdmin() {
		f.ShelterID = actor.ShelterID
	}
	apps, err := svc.store.ListApplications(ctx, f)
	if err != nil {
		return nil, err
	}
	return svc.toViews(ctx, apps), nil
}

func (svc *ApplicationService) StartReview(ctx context.Context, actor model.User, id string, in model.ReviewInput) (model.Application, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Application{}, err
	}
	a, p, err := svc.loadManaged(ctx, actor, id)
	if err != nil {
		return model.Application{}, err
	}
	_ = p
	if !a.Status.CanReview() {
		return model.Application{}, model.ErrInvalidAppStatus
	}
	a.Status = model.AppUnderReview
	a.ReviewerID = actor.ID
	a.ReviewNote = strings.TrimSpace(in.Note)
	now := svc.clock.Now()
	a.ReviewedAt = &now
	updated, err := svc.store.UpdateApplication(ctx, a)
	if err != nil {
		return model.Application{}, err
	}
	_ = svc.notify.Push(ctx, a.ApplicantID, model.NotifyGeneric, "申请进入审核", "救助站已开始审核你的领养申请", updated.ID)
	return updated, nil
}

func (svc *ApplicationService) Approve(ctx context.Context, actor model.User, id string, in model.ReviewInput, force bool) (model.Application, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Application{}, err
	}
	a, p, err := svc.loadManaged(ctx, actor, id)
	if err != nil {
		return model.Application{}, err
	}
	if p.NeedHomeCheck && !force {
		ok, err := svc.homeCheckPassed(ctx, a)
		if err != nil {
			return model.Application{}, err
		}
		if !ok {
			return model.Application{}, model.ErrHomeCheckRequired
		}
	}
	updated, pet, waitlisted, err := svc.store.ApproveApplication(ctx, id, actor.ID, strings.TrimSpace(in.Note), svc.clock.Now())
	if err != nil {
		return model.Application{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditAppApprove, updated.ID, pet.Name, svc.clock.Now())
	_ = svc.notify.Push(ctx, updated.ApplicantID, model.NotifyAppApproved, "申请已录取", "你已获得 "+pet.Name+" 的预留资格，请等待交接通知", updated.ID)
	for _, w := range waitlisted {
		_ = svc.notify.Push(ctx, w.ApplicantID, model.NotifyAppWaitlisted, "进入候补", pet.Name+" 已被预留，你的申请进入候补", w.ID)
	}
	return updated, nil
}

func (svc *ApplicationService) Reject(ctx context.Context, actor model.User, id string, in model.RejectInput) (model.Application, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Application{}, err
	}
	if err := in.Validate(); err != nil {
		return model.Application{}, err
	}
	if _, _, err := svc.loadManaged(ctx, actor, id); err != nil {
		return model.Application{}, err
	}
	updated, pet, promoted, err := svc.store.RejectApplication(ctx, id, actor.ID, strings.TrimSpace(in.Reason), svc.clock.Now())
	if err != nil {
		return model.Application{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditAppReject, updated.ID, in.Reason, svc.clock.Now())
	_ = svc.notify.Push(ctx, updated.ApplicantID, model.NotifyAppRejected, "申请未通过", in.Reason, updated.ID)
	if pet != nil {
		for _, pmt := range promoted {
			_ = svc.notify.Push(ctx, pmt.ApplicantID, model.NotifyAppPromoted, "候补递补", "前面的申请已释放，请等待审核", pmt.ID)
		}
	}
	return updated, nil
}

func (svc *ApplicationService) Withdraw(ctx context.Context, actor model.User, id string) (model.Application, error) {
	a, err := svc.store.GetApplication(ctx, id)
	if err != nil {
		return model.Application{}, err
	}
	if a.ApplicantID != actor.ID && !actor.IsAdmin() {
		return model.Application{}, model.ErrNotApplicant
	}
	if !a.Status.CanWithdraw() {
		return model.Application{}, model.ErrInvalidAppStatus
	}
	now := svc.clock.Now()
	if a.Status == model.AppApproved {
		_, _, err := svc.store.ApplyCredit(ctx, a.ApplicantID, policy.DeltaApplyDefault, model.CreditApplyDefault, a.ID, "录取后撤回", now)
		if err != nil {
			return model.Application{}, err
		}
		updated, pet, promoted, err := svc.store.RejectApplication(ctx, id, actor.ID, "申请人撤回", now)
		if err != nil {
			return model.Application{}, err
		}
		updated.Status = model.AppWithdrawn
		updated, err = svc.store.UpdateApplication(ctx, updated)
		if err != nil {
			return model.Application{}, err
		}
		if pet != nil {
			for _, pmt := range promoted {
				_ = svc.notify.Push(ctx, pmt.ApplicantID, model.NotifyAppPromoted, "候补递补", "前序申请已撤回", pmt.ID)
			}
		}
		return updated, nil
	}
	a.Status = model.AppWithdrawn
	return svc.store.UpdateApplication(ctx, a)
}

func (svc *ApplicationService) Handover(ctx context.Context, actor model.User, id string, in model.HandoverInput) (model.Application, []model.Visit, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Application{}, nil, err
	}
	if err := in.Validate(); err != nil {
		return model.Application{}, nil, err
	}
	if _, _, err := svc.loadManaged(ctx, actor, id); err != nil {
		return model.Application{}, nil, err
	}
	now := svc.clock.Now()
	dues := policy.FollowupDueTimes(now)
	planned := make([]model.Visit, 0, len(dues))
	for i, due := range dues {
		planned = append(planned, model.Visit{
			Type:        model.VisitFollowup,
			Status:      model.VisitScheduled,
			ScheduledAt: due,
			StaffID:     actor.ID,
			Sequence:    i + 1,
		})
	}
	note := strings.TrimSpace(in.Note)
	if in.Checklist != "" {
		note = strings.TrimSpace(note + "\n清单: " + in.Checklist)
	}
	app, pet, visits, err := svc.store.HandoverAdoption(ctx, id, actor.ID, note, planned, now)
	if err != nil {
		return model.Application{}, nil, err
	}
	_, _, _ = svc.store.ApplyCredit(ctx, app.ApplicantID, policy.DeltaAdoptionDone, model.CreditAdoptionDone, app.ID, "完成交接", now)
	audit(ctx, svc.store, actor.ID, model.AuditHandover, app.ID, pet.Name, now)
	_ = svc.notify.Push(ctx, app.ApplicantID, model.NotifyHandover, "领养交接完成", "请按期配合回访："+pet.Name, app.ID)
	return app, visits, nil
}

func (svc *ApplicationService) Return(ctx context.Context, actor model.User, id string, in model.ReturnInput) (model.Application, error) {
	if err := in.Validate(); err != nil {
		return model.Application{}, err
	}
	a, err := svc.store.GetApplication(ctx, id)
	if err != nil {
		return model.Application{}, err
	}
	p, err := svc.store.GetPet(ctx, a.PetID)
	if err != nil {
		return model.Application{}, err
	}
	isStaff := canManagePet(actor, p)
	if a.ApplicantID != actor.ID && !isStaff {
		return model.Application{}, model.ErrForbidden
	}
	if a.ApplicantID == actor.ID && !in.Approve && !isStaff {
		// 领养人发起：仍走审核通过路径，由员工/管理员执行 Approve=true
		if !actor.IsAdmin() && !actor.IsStaff() {
			a.ReturnReason = strings.TrimSpace(in.Reason)
			a.ReturnMedical = in.Medical
			updated, err := svc.store.UpdateApplication(ctx, a)
			if err != nil {
				return model.Application{}, err
			}
			if p.StaffID != "" {
				_ = svc.notify.Push(ctx, p.StaffID, model.NotifyReturn, "收到退养申请", a.ReturnReason, a.ID)
			}
			return updated, nil
		}
	}
	if !isStaff && !actor.IsAdmin() {
		return model.Application{}, model.ErrForbidden
	}
	now := svc.clock.Now()
	updated, pet, err := svc.store.ReturnAdoption(ctx, id, actor.ID, strings.TrimSpace(in.Reason), in.Medical, now)
	if err != nil {
		return model.Application{}, err
	}
	delta, reason := policy.CreditDeltaForReturn(derefTime(updated.HandoverAt), now, in.Medical)
	if delta != 0 {
		_, _, _ = svc.store.ApplyCredit(ctx, updated.ApplicantID, delta, reason, updated.ID, in.Reason, now)
	}
	audit(ctx, svc.store, actor.ID, model.AuditReturn, updated.ID, pet.Name, now)
	_ = svc.notify.Push(ctx, updated.ApplicantID, model.NotifyReturn, "退养已办理", in.Reason, updated.ID)
	return updated, nil
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func (svc *ApplicationService) homeCheckPassed(ctx context.Context, a model.Application) (bool, error) {
	visits, err := svc.store.ListVisits(ctx, model.VisitFilter{
		ApplicationID: a.ID,
		Type:          model.VisitHomeCheck,
		Status:        model.VisitCompleted,
	})
	if err != nil {
		return false, err
	}
	for _, v := range visits {
		if v.PassedHomeCheck(policy.HomeCheckMinScore) {
			return true, nil
		}
		return false, model.ErrHomeCheckFailed
	}
	return false, nil
}

func (svc *ApplicationService) loadManaged(ctx context.Context, actor model.User, id string) (model.Application, model.Pet, error) {
	a, err := svc.store.GetApplication(ctx, id)
	if err != nil {
		return model.Application{}, model.Pet{}, err
	}
	p, err := svc.store.GetPet(ctx, a.PetID)
	if err != nil {
		return model.Application{}, model.Pet{}, err
	}
	if !canManagePet(actor, p) {
		return model.Application{}, model.Pet{}, model.ErrNotPetStaff
	}
	return a, p, nil
}

func (svc *ApplicationService) canSee(actor model.User, a model.Application) bool {
	if actor.IsAdmin() {
		return true
	}
	if a.ApplicantID == actor.ID {
		return true
	}
	if actor.IsStaff() && actor.ShelterID == a.ShelterID {
		return true
	}
	return false
}

func (svc *ApplicationService) toView(ctx context.Context, a model.Application) model.ApplicationView {
	v := model.ApplicationView{Application: a}
	if p, err := svc.store.GetPet(ctx, a.PetID); err == nil {
		v.PetName = p.Name
		v.PetSpecies = p.Species
		v.PetStatus = p.Status
	}
	if u, err := publicOf(ctx, svc.store, a.ApplicantID); err == nil {
		v.Applicant = u
		v.ApplicantName = u.DisplayName
	}
	return v
}

func (svc *ApplicationService) toViews(ctx context.Context, apps []model.Application) []model.ApplicationView {
	out := make([]model.ApplicationView, 0, len(apps))
	for _, a := range apps {
		out = append(out, svc.toView(ctx, a))
	}
	return out
}
