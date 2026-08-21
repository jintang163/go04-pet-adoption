package service

import (
	"context"
	"strings"
	"time"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/store"
)

type HealthService struct {
	store store.Store
	clock Clock
}

func NewHealthService(s store.Store, clock Clock) *HealthService {
	return &HealthService{store: s, clock: clock}
}

func (svc *HealthService) Add(ctx context.Context, actor model.User, petID string, in model.HealthInput) (model.HealthRecord, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.HealthRecord{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.HealthRecord{}, err
	}
	p, err := svc.store.GetPet(ctx, petID)
	if err != nil {
		return model.HealthRecord{}, err
	}
	if !canManagePet(actor, p) {
		return model.HealthRecord{}, model.ErrNotPetStaff
	}
	rec := model.HealthRecord{
		PetID:      p.ID,
		StaffID:    actor.ID,
		Kind:       in.Kind,
		Title:      in.Title,
		Detail:     in.Detail,
		OccurredAt: in.OccurredAt,
		NextDueAt:  in.NextDueAt,
		CreatedAt:  svc.clock.Now(),
	}
	created, err := svc.store.CreateHealth(ctx, rec)
	if err != nil {
		return model.HealthRecord{}, err
	}
	switch in.Kind {
	case model.HealthVaccine:
		p.Vaccinated = true
		_, _ = svc.store.UpdatePet(ctx, p)
	case model.HealthSterilize:
		p.Sterilized = true
		_, _ = svc.store.UpdatePet(ctx, p)
	}
	return created, nil
}

func (svc *HealthService) List(ctx context.Context, actor model.User, petID string) ([]model.HealthRecord, error) {
	p, err := svc.store.GetPet(ctx, petID)
	if err != nil {
		return nil, err
	}
	if p.Status == model.PetDraft && !canManagePet(actor, p) {
		return nil, model.ErrNotFound
	}
	return svc.store.ListHealthByPet(ctx, petID)
}

type SocialService struct {
	store  store.Store
	notify *NotifyService
	clock  Clock
}

func NewSocialService(s store.Store, notify *NotifyService, clock Clock) *SocialService {
	return &SocialService{store: s, notify: notify, clock: clock}
}

func (svc *SocialService) Favorite(ctx context.Context, actor model.User, petID string) (model.Favorite, error) {
	if err := actor.CanWrite(); err != nil {
		return model.Favorite{}, err
	}
	if _, err := svc.store.GetPet(ctx, petID); err != nil {
		return model.Favorite{}, err
	}
	return svc.store.CreateFavorite(ctx, model.Favorite{
		UserID:    actor.ID,
		PetID:     petID,
		CreatedAt: svc.clock.Now(),
	})
}

func (svc *SocialService) Unfavorite(ctx context.Context, actor model.User, petID string) error {
	return svc.store.DeleteFavorite(ctx, actor.ID, petID)
}

func (svc *SocialService) MyFavorites(ctx context.Context, actor model.User) ([]model.PetView, error) {
	favs, err := svc.store.ListFavoritesByUser(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]model.PetView, 0, len(favs))
	for _, f := range favs {
		p, err := svc.store.GetPet(ctx, f.PetID)
		if err != nil {
			continue
		}
		out = append(out, model.PetView{Pet: p, Favorited: true})
	}
	return out, nil
}

func (svc *SocialService) Ask(ctx context.Context, actor model.User, petID string, in model.InquiryInput) (model.Inquiry, error) {
	if err := actor.CanWrite(); err != nil {
		return model.Inquiry{}, err
	}
	if err := in.Validate(); err != nil {
		return model.Inquiry{}, err
	}
	p, err := svc.store.GetPet(ctx, petID)
	if err != nil {
		return model.Inquiry{}, err
	}
	q, err := svc.store.CreateInquiry(ctx, model.Inquiry{
		PetID:     p.ID,
		UserID:    actor.ID,
		Content:   strings.TrimSpace(in.Content),
		CreatedAt: svc.clock.Now(),
	})
	if err != nil {
		return model.Inquiry{}, err
	}
	if p.StaffID != "" {
		_ = svc.notify.Push(ctx, p.StaffID, model.NotifyGeneric, "新的宠物问询", p.Name, q.ID)
	}
	return q, nil
}

func (svc *SocialService) Reply(ctx context.Context, actor model.User, id string, in model.InquiryInput) (model.Inquiry, error) {
	if err := in.Validate(); err != nil {
		return model.Inquiry{}, err
	}
	q, err := svc.store.GetInquiry(ctx, id)
	if err != nil {
		return model.Inquiry{}, err
	}
	p, err := svc.store.GetPet(ctx, q.PetID)
	if err != nil {
		return model.Inquiry{}, err
	}
	if !canManagePet(actor, p) {
		return model.Inquiry{}, model.ErrNotPetStaff
	}
	now := svc.clock.Now()
	q.Reply = strings.TrimSpace(in.Content)
	q.ReplierID = actor.ID
	q.RepliedAt = &now
	updated, err := svc.store.UpdateInquiry(ctx, q)
	if err != nil {
		return model.Inquiry{}, err
	}
	_ = svc.notify.Push(ctx, q.UserID, model.NotifyGeneric, "问询已回复", p.Name, q.ID)
	return updated, nil
}

func (svc *SocialService) ListInquiries(ctx context.Context, actor model.User, petID string) ([]model.Inquiry, error) {
	p, err := svc.store.GetPet(ctx, petID)
	if err != nil {
		return nil, err
	}
	if p.Status == model.PetDraft && !canManagePet(actor, p) {
		return nil, model.ErrNotFound
	}
	return svc.store.ListInquiriesByPet(ctx, petID)
}

func (svc *SocialService) DeleteInquiry(ctx context.Context, actor model.User, id string) error {
	q, err := svc.store.GetInquiry(ctx, id)
	if err != nil {
		return err
	}
	if !actor.IsAdmin() && q.UserID != actor.ID {
		p, err := svc.store.GetPet(ctx, q.PetID)
		if err != nil {
			return err
		}
		if !canManagePet(actor, p) {
			return model.ErrForbidden
		}
	}
	return svc.store.DeleteInquiry(ctx, id)
}

type NotifyService struct {
	store store.Store
	clock Clock
}

func NewNotifyService(s store.Store, clock Clock) *NotifyService {
	return &NotifyService{store: s, clock: clock}
}

func (svc *NotifyService) Push(ctx context.Context, userID string, kind model.NotificationKind, title, body, related string) error {
	if userID == "" {
		return nil
	}
	_, err := svc.store.CreateNotification(ctx, model.Notification{
		UserID:    userID,
		Kind:      kind,
		Title:     title,
		Body:      body,
		RelatedID: related,
		CreatedAt: svc.clock.Now(),
	})
	return err
}

func (svc *NotifyService) List(ctx context.Context, actor model.User, unreadOnly bool) ([]model.Notification, error) {
	return svc.store.ListNotifications(ctx, actor.ID, unreadOnly)
}

func (svc *NotifyService) Read(ctx context.Context, actor model.User, id string) (model.Notification, error) {
	n, err := svc.store.GetNotification(ctx, id)
	if err != nil {
		return model.Notification{}, err
	}
	if n.UserID != actor.ID {
		return model.Notification{}, model.ErrForbidden
	}
	now := svc.clock.Now()
	n.ReadAt = &now
	return svc.store.UpdateNotification(ctx, n)
}

func (svc *NotifyService) ReadAll(ctx context.Context, actor model.User) (int, error) {
	return svc.store.MarkAllNotificationsRead(ctx, actor.ID, svc.clock.Now())
}

type StatsService struct {
	store store.Store
	clock Clock
}

func NewStatsService(s store.Store, clock Clock) *StatsService {
	return &StatsService{store: s, clock: clock}
}

func (svc *StatsService) Global(ctx context.Context, actor model.User) (model.StatsSnapshot, error) {
	if !actor.IsAdmin() {
		return model.StatsSnapshot{}, model.ErrForbidden
	}
	total, active, frozen, err := svc.store.CountUsers(ctx)
	if err != nil {
		return model.StatsSnapshot{}, err
	}
	byStatus, petsTotal, err := svc.store.CountPetsByStatus(ctx, "")
	if err != nil {
		return model.StatsSnapshot{}, err
	}
	appsTotal, appsApproved, appsCompleted, returned, err := svc.store.CountApplications(ctx)
	if err != nil {
		return model.StatsSnapshot{}, err
	}
	vs, vc, vm, err := svc.store.CountVisits(ctx)
	if err != nil {
		return model.StatsSnapshot{}, err
	}
	now := svc.clock.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	handoverMonth := 0
	apps, _ := svc.store.ListApplications(ctx, model.ApplicationFilter{Status: model.AppCompleted})
	for _, a := range apps {
		if a.HandoverAt != nil && !a.HandoverAt.Before(monthStart) {
			handoverMonth++
		}
	}
	statusMap := map[string]int{}
	for k, v := range byStatus {
		statusMap[string(k)] = v
	}
	conv := 0.0
	if appsTotal > 0 {
		conv = float64(appsCompleted) / float64(appsTotal)
	}
	visitRate := 0.0
	den := vc + vm
	if den > 0 {
		visitRate = float64(vc) / float64(den)
	}
	return model.StatsSnapshot{
		UsersTotal:        total,
		UsersActive:       active,
		UsersFrozen:       frozen,
		PetsByStatus:      statusMap,
		PetsTotal:         petsTotal,
		AppsTotal:         appsTotal,
		AppsApproved:      appsApproved,
		AppsCompleted:     appsCompleted,
		HandoverThisMonth: handoverMonth,
		VisitsScheduled:   vs,
		VisitsCompleted:   vc,
		VisitsMissed:      vm,
		ReturnCount:       returned,
		ConversionRate:    conv,
		VisitCompleteRate: visitRate,
	}, nil
}

func (svc *StatsService) StaffBoard(ctx context.Context, actor model.User) (model.StaffBoard, error) {
	if !actor.IsStaff() && !actor.IsAdmin() {
		return model.StaffBoard{}, model.ErrForbidden
	}
	shelterID := ""
	if actor.IsStaff() && !actor.IsAdmin() {
		shelterID = actor.ShelterID
	}
	pending, _ := svc.store.ListApplications(ctx, model.ApplicationFilter{ShelterID: shelterID, Status: model.AppPending})
	review, _ := svc.store.ListApplications(ctx, model.ApplicationFilter{ShelterID: shelterID, Status: model.AppUnderReview})
	by, _, _ := svc.store.CountPetsByStatus(ctx, shelterID)
	now := svc.clock.Now()
	start := startOfDay(now)
	end := start.Add(24 * time.Hour)
	due, _ := svc.store.ListVisits(ctx, model.VisitFilter{
		ShelterID: shelterID,
		Status:    model.VisitScheduled,
		DueAfter:  &start,
		DueBefore: &end,
	})
	return model.StaffBoard{
		PendingApps:     len(pending),
		UnderReviewApps: len(review),
		DraftPets:       by[model.PetDraft],
		PublishedPets:   by[model.PetPublished],
		DueVisitsToday:  len(due),
	}, nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
