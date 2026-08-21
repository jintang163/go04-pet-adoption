package service

import (
	"context"
	"time"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/store"
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

var DefaultClock Clock = systemClock{}

type Services struct {
	Auth    *AuthService
	User    *UserService
	Pet     *PetService
	App     *ApplicationService
	Visit   *VisitService
	Health  *HealthService
	Social  *SocialService
	Stats   *StatsService
	Notify  *NotifyService
	MaxPend int
}

func NewServices(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, maxPend int) *Services {
	if clock == nil {
		clock = DefaultClock
	}
	if maxPend <= 0 {
		maxPend = 3
	}
	notify := NewNotifyService(s, clock)
	svc := &Services{
		Notify:  notify,
		MaxPend: maxPend,
	}
	svc.Auth = NewAuthService(s, hasher, sessions, clock, notify)
	svc.User = NewUserService(s, hasher, sessions, clock, notify)
	svc.Pet = NewPetService(s, notify, clock)
	svc.App = NewApplicationService(s, notify, clock, maxPend)
	svc.Visit = NewVisitService(s, notify, clock)
	svc.Health = NewHealthService(s, clock)
	svc.Social = NewSocialService(s, notify, clock)
	svc.Stats = NewStatsService(s, clock)
	return svc
}

type ctxKey string

const ctxUserKey ctxKey = "user"

func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, ctxUserKey, u)
}

func UserFromContext(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(model.User)
	return u, ok
}

func MustUserFromContext(ctx context.Context) model.User {
	u, ok := UserFromContext(ctx)
	if !ok {
		panic("service: user not found in context")
	}
	return u
}

func requireActiveWriter(u model.User) error {
	if u.IsAdmin() {
		return nil
	}
	return u.CanWrite()
}

func publicOf(ctx context.Context, s store.Store, id string) (model.PublicUser, error) {
	u, err := s.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func canManagePet(u model.User, p model.Pet) bool {
	if u.IsAdmin() {
		return true
	}
	return u.IsStaff() && u.ShelterID == p.ShelterID
}

func audit(ctx context.Context, s store.Store, actorID string, action model.AuditAction, targetID, detail string, now time.Time) {
	_, _ = s.CreateAuditLog(ctx, model.AuditLog{
		ActorID:   actorID,
		Action:    action,
		TargetID:  targetID,
		Detail:    detail,
		CreatedAt: now,
	})
}
