package service

import (
	"context"
	"strings"
	"time"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/store"
)

type AuthService struct {
	store    store.Store
	hasher   *auth.PasswordHasher
	sessions *auth.SessionManager
	clock    Clock
	notify   *NotifyService
}

func NewAuthService(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, notify *NotifyService) *AuthService {
	return &AuthService{store: s, hasher: hasher, sessions: sessions, clock: clock, notify: notify}
}

func (svc *AuthService) Register(ctx context.Context, in model.UserInput) (model.AuthResult, error) {
	in.Normalize()
	if in.Role == "" {
		in.Role = model.RoleAdopter
	}
	if in.Role != model.RoleAdopter {
		return model.AuthResult{}, model.ErrForbidden
	}
	if err := in.Validate(); err != nil {
		return model.AuthResult{}, err
	}
	if _, err := svc.store.GetUserByUsername(ctx, in.Username); err == nil {
		return model.AuthResult{}, model.ErrAlreadyExists
	}
	salt, hash, it, err := svc.hasher.Hash(in.Password)
	if err != nil {
		return model.AuthResult{}, err
	}
	now := svc.clock.Now()
	u, err := svc.store.CreateUser(ctx, model.User{
		Username:     in.Username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   it,
		Role:         model.RoleAdopter,
		Status:       model.UserActive,
		DisplayName:  in.DisplayName,
		Phone:        in.Phone,
		Bio:          in.Bio,
		City:         in.City,
		Housing:      in.Housing,
		HasChildren:  in.HasChildren,
		HasOtherPets: in.HasOtherPets,
		AgeYears:     in.AgeYears,
		Experience:   in.Experience,
		CreditScore:  model.CreditInitial,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return model.AuthResult{}, err
	}
	token, err := svc.sessions.Create(u)
	if err != nil {
		return model.AuthResult{}, err
	}
	return model.AuthResult{Token: token, User: u.Public()}, nil
}

func (svc *AuthService) Login(ctx context.Context, in model.LoginInput) (model.AuthResult, error) {
	in.Normalize()
	if in.Username == "" || in.Password == "" {
		return model.AuthResult{}, model.ErrInvalidCredentials
	}
	u, err := svc.store.GetUserByUsername(ctx, in.Username)
	if err != nil {
		return model.AuthResult{}, model.ErrInvalidCredentials
	}
	if u.IsBanned() {
		return model.AuthResult{}, model.ErrAccountBanned
	}
	if !svc.hasher.Verify(in.Password, u.PasswordSalt, u.PasswordHash, u.Iterations) {
		return model.AuthResult{}, model.ErrInvalidCredentials
	}
	now := svc.clock.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
	if _, err := svc.store.UpdateUser(ctx, u); err != nil {
		return model.AuthResult{}, err
	}
	token, err := svc.sessions.Create(u)
	if err != nil {
		return model.AuthResult{}, err
	}
	return model.AuthResult{Token: token, User: u.Public()}, nil
}

func (svc *AuthService) Logout(token string) {
	svc.sessions.Invalidate(token)
}

func (svc *AuthService) Me(ctx context.Context, u model.User) (model.MeResponse, error) {
	fresh, err := svc.store.GetUserByID(ctx, u.ID)
	if err != nil {
		return model.MeResponse{}, err
	}
	unread, _ := svc.store.CountUnreadNotifications(ctx, u.ID)
	active, _ := svc.store.CountActiveApplicationsByApplicant(ctx, u.ID)
	visits, _ := svc.store.ListVisits(ctx, model.VisitFilter{AdopterID: u.ID, Status: model.VisitScheduled})
	upcoming := 0
	now := svc.clock.Now()
	horizon := now.Add(14 * 24 * time.Hour)
	for _, v := range visits {
		if v.ScheduledAt.Before(horizon) {
			upcoming++
		}
	}
	return model.MeResponse{
		User:           fresh.Public(),
		UnreadNotify:   unread,
		ActiveApps:     active,
		UpcomingVisits: upcoming,
	}, nil
}

type UserService struct {
	store    store.Store
	hasher   *auth.PasswordHasher
	sessions *auth.SessionManager
	clock    Clock
	notify   *NotifyService
}

func NewUserService(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, notify *NotifyService) *UserService {
	return &UserService{store: s, hasher: hasher, sessions: sessions, clock: clock, notify: notify}
}

func (svc *UserService) GetByID(ctx context.Context, id string) (model.User, error) {
	return svc.store.GetUserByID(ctx, id)
}

func (svc *UserService) List(ctx context.Context, actor model.User, f model.UserFilter) ([]model.PublicUser, error) {
	if !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	users, err := svc.store.ListUsers(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicUser, 0, len(users))
	for _, u := range users {
		out = append(out, u.Public())
	}
	return out, nil
}

func (svc *UserService) Create(ctx context.Context, actor model.User, in model.UserInput) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.PublicUser{}, err
	}
	if in.Role == model.RoleStaff {
		if in.ShelterID == "" {
			return model.PublicUser{}, model.ErrStaffShelterRequired
		}
		sh, err := svc.store.GetShelter(ctx, in.ShelterID)
		if err != nil {
			return model.PublicUser{}, err
		}
		if !sh.IsActive() {
			return model.PublicUser{}, model.ErrShelterInactive
		}
	}
	if _, err := svc.store.GetUserByUsername(ctx, in.Username); err == nil {
		return model.PublicUser{}, model.ErrAlreadyExists
	}
	salt, hash, it, err := svc.hasher.Hash(in.Password)
	if err != nil {
		return model.PublicUser{}, err
	}
	now := svc.clock.Now()
	u, err := svc.store.CreateUser(ctx, model.User{
		Username:     in.Username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   it,
		Role:         in.Role,
		Status:       model.UserActive,
		DisplayName:  in.DisplayName,
		Phone:        in.Phone,
		Bio:          in.Bio,
		City:         in.City,
		Housing:      in.Housing,
		HasChildren:  in.HasChildren,
		HasOtherPets: in.HasOtherPets,
		AgeYears:     in.AgeYears,
		Experience:   in.Experience,
		ShelterID:    in.ShelterID,
		CreditScore:  model.CreditInitial,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func (svc *UserService) UpdateProfile(ctx context.Context, actor model.User, in model.ProfileInput) (model.PublicUser, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.PublicUser{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.PublicUser{}, err
	}
	u, err := svc.store.GetUserByID(ctx, actor.ID)
	if err != nil {
		return model.PublicUser{}, err
	}
	u.DisplayName = in.DisplayName
	u.Phone = in.Phone
	u.Bio = in.Bio
	u.City = in.City
	u.Housing = in.Housing
	u.HasChildren = in.HasChildren
	u.HasOtherPets = in.HasOtherPets
	u.AgeYears = in.AgeYears
	u.Experience = in.Experience
	u, err = svc.store.UpdateUser(ctx, u)
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}

func (svc *UserService) ChangePassword(ctx context.Context, actor model.User, in model.PasswordChangeInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	u, err := svc.store.GetUserByID(ctx, actor.ID)
	if err != nil {
		return err
	}
	if !svc.hasher.Verify(in.OldPassword, u.PasswordSalt, u.PasswordHash, u.Iterations) {
		return model.ErrInvalidCredentials
	}
	salt, hash, it, err := svc.hasher.Hash(in.NewPassword)
	if err != nil {
		return err
	}
	u.PasswordSalt = salt
	u.PasswordHash = hash
	u.Iterations = it
	if _, err := svc.store.UpdateUser(ctx, u); err != nil {
		return err
	}
	svc.sessions.InvalidateByUser(u.ID)
	return nil
}

func (svc *UserService) Freeze(ctx context.Context, actor model.User, id string, freeze bool) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	u, err := svc.store.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	if u.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	if freeze {
		u.Status = model.UserFrozen
		svc.sessions.InvalidateByUser(u.ID)
	} else {
		u.Status = model.UserActive
	}
	u, err = svc.store.UpdateUser(ctx, u)
	if err != nil {
		return model.PublicUser{}, err
	}
	action := model.AuditUserFreeze
	detail := "unfreeze"
	if freeze {
		detail = "freeze"
	}
	audit(ctx, svc.store, actor.ID, action, u.ID, detail, svc.clock.Now())
	return u.Public(), nil
}

func (svc *UserService) AdjustCredit(ctx context.Context, actor model.User, id string, in model.CreditAdjustInput) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	if err := in.Validate(); err != nil {
		return model.PublicUser{}, err
	}
	u, _, err := svc.store.ApplyCredit(ctx, id, in.Delta, model.CreditAdminAdjust, actor.ID, strings.TrimSpace(in.Reason), svc.clock.Now())
	if err != nil {
		return model.PublicUser{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditCredit, id, in.Reason, svc.clock.Now())
	_ = svc.notify.Push(ctx, id, model.NotifyCredit, "信用分调整", in.Reason, id)
	return u.Public(), nil
}

func (svc *UserService) CreateShelter(ctx context.Context, actor model.User, in model.ShelterInput) (model.Shelter, error) {
	if !actor.IsAdmin() {
		return model.Shelter{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Shelter{}, err
	}
	return svc.store.CreateShelter(ctx, model.Shelter{
		Name:    in.Name,
		City:    in.City,
		Address: in.Address,
		Phone:   in.Phone,
		Status:  model.ShelterActive,
	})
}

func (svc *UserService) ListShelters(ctx context.Context, activeOnly bool) ([]model.Shelter, error) {
	return svc.store.ListShelters(ctx, activeOnly)
}

func (svc *UserService) UpdateShelter(ctx context.Context, actor model.User, id string, in model.ShelterInput) (model.Shelter, error) {
	if !actor.IsAdmin() {
		return model.Shelter{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Shelter{}, err
	}
	sh, err := svc.store.GetShelter(ctx, id)
	if err != nil {
		return model.Shelter{}, err
	}
	sh.Name = in.Name
	sh.City = in.City
	sh.Address = in.Address
	sh.Phone = in.Phone
	return svc.store.UpdateShelter(ctx, sh)
}

func (svc *UserService) SetShelterActive(ctx context.Context, actor model.User, id string, active bool) (model.Shelter, error) {
	if !actor.IsAdmin() {
		return model.Shelter{}, model.ErrForbidden
	}
	sh, err := svc.store.GetShelter(ctx, id)
	if err != nil {
		return model.Shelter{}, err
	}
	if active {
		sh.Status = model.ShelterActive
	} else {
		sh.Status = model.ShelterInactive
	}
	return svc.store.UpdateShelter(ctx, sh)
}

func (svc *UserService) CreditLogs(ctx context.Context, actor model.User, userID string) ([]model.CreditLog, error) {
	if userID == "" {
		userID = actor.ID
	}
	if userID != actor.ID && !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	return svc.store.ListCreditLogs(ctx, userID)
}
