package store

import (
	"context"
	"time"

	"go04-pet-adoption/internal/model"
)

// Store 数据访问接口。service 层只依赖本接口。
type Store interface {
	CreateUser(ctx context.Context, u model.User) (model.User, error)
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	ListUsers(ctx context.Context, f model.UserFilter) ([]model.User, error)
	UpdateUser(ctx context.Context, u model.User) (model.User, error)
	CountUsers(ctx context.Context) (total, active, frozen int, err error)

	CreateShelter(ctx context.Context, s model.Shelter) (model.Shelter, error)
	GetShelter(ctx context.Context, id string) (model.Shelter, error)
	ListShelters(ctx context.Context, activeOnly bool) ([]model.Shelter, error)
	UpdateShelter(ctx context.Context, s model.Shelter) (model.Shelter, error)

	CreatePet(ctx context.Context, p model.Pet) (model.Pet, error)
	GetPet(ctx context.Context, id string) (model.Pet, error)
	ListPets(ctx context.Context, f model.PetFilter) ([]model.Pet, error)
	UpdatePet(ctx context.Context, p model.Pet) (model.Pet, error)
	CountPetsByStatus(ctx context.Context, shelterID string) (map[model.PetStatus]int, int, error)

	// CreateApplication 在同一把写锁内完成"重复活跃申请"与"待处理申请上限"两项校验再插入，
	// 避免上层先读后写造成 check-then-act 竞态。maxActive<=0 表示不校验上限。
	CreateApplication(ctx context.Context, a model.Application, maxActive int) (model.Application, error)
	GetApplication(ctx context.Context, id string) (model.Application, error)
	GetApplicationByPetApplicant(ctx context.Context, petID, applicantID string) (model.Application, error)
	ListApplications(ctx context.Context, f model.ApplicationFilter) ([]model.Application, error)
	UpdateApplication(ctx context.Context, a model.Application) (model.Application, error)
	CountActiveApplicationsByApplicant(ctx context.Context, applicantID string) (int, error)
	CountApplications(ctx context.Context) (total, approved, completed, returned int, err error)

	CreateVisit(ctx context.Context, v model.Visit) (model.Visit, error)
	GetVisit(ctx context.Context, id string) (model.Visit, error)
	ListVisits(ctx context.Context, f model.VisitFilter) ([]model.Visit, error)
	UpdateVisit(ctx context.Context, v model.Visit) (model.Visit, error)
	CountVisits(ctx context.Context) (scheduled, completed, missed int, err error)

	CreateHealth(ctx context.Context, h model.HealthRecord) (model.HealthRecord, error)
	ListHealthByPet(ctx context.Context, petID string) ([]model.HealthRecord, error)

	CreateFavorite(ctx context.Context, f model.Favorite) (model.Favorite, error)
	DeleteFavorite(ctx context.Context, userID, petID string) error
	GetFavorite(ctx context.Context, userID, petID string) (model.Favorite, error)
	ListFavoritesByUser(ctx context.Context, userID string) ([]model.Favorite, error)

	CreateInquiry(ctx context.Context, q model.Inquiry) (model.Inquiry, error)
	GetInquiry(ctx context.Context, id string) (model.Inquiry, error)
	ListInquiriesByPet(ctx context.Context, petID string) ([]model.Inquiry, error)
	UpdateInquiry(ctx context.Context, q model.Inquiry) (model.Inquiry, error)
	DeleteInquiry(ctx context.Context, id string) error

	CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	ListNotifications(ctx context.Context, userID string, unreadOnly bool) ([]model.Notification, error)
	GetNotification(ctx context.Context, id string) (model.Notification, error)
	UpdateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	MarkAllNotificationsRead(ctx context.Context, userID string, at time.Time) (int, error)
	CountUnreadNotifications(ctx context.Context, userID string) (int, error)

	CreateAuditLog(ctx context.Context, l model.AuditLog) (model.AuditLog, error)
	ListAuditLogs(ctx context.Context, targetID string) ([]model.AuditLog, error)

	CreateCreditLog(ctx context.Context, l model.CreditLog) (model.CreditLog, error)
	ListCreditLogs(ctx context.Context, userID string) ([]model.CreditLog, error)

	ApproveApplication(ctx context.Context, appID, reviewerID, note string, now time.Time) (model.Application, model.Pet, []model.Application, error)
	RejectApplication(ctx context.Context, appID, reviewerID, reason string, now time.Time) (model.Application, *model.Pet, []model.Application, error)
	WithdrawApprovedApplication(ctx context.Context, appID, applicantID, actorID string, creditDelta int, creditReason model.CreditReason, creditNote string, now time.Time) (model.Application, *model.Pet, []model.Application, model.CreditLog, error)
	HandoverAdoption(ctx context.Context, appID, staffID, note string, visits []model.Visit, now time.Time) (model.Application, model.Pet, []model.Visit, error)
	ReturnAdoption(ctx context.Context, appID, actorID, reason string, medical bool, now time.Time) (model.Application, model.Pet, error)
	ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string, now time.Time) (model.User, model.CreditLog, error)
}

var _ Store = (*MemoryStore)(nil)
