package handler

import (
	"io/fs"
	"net/http"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/middleware"
	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/respond"
	"go04-pet-adoption/internal/service"
	"go04-pet-adoption/internal/store"
)

type Handler struct {
	services *service.Services
	store    store.Store
	sessions *auth.SessionManager
	assets   fs.FS
}

func New(svc *service.Services, st store.Store, sessions *auth.SessionManager, assets fs.FS) *Handler {
	return &Handler{services: svc, store: st, sessions: sessions, assets: assets}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	authMw := middleware.RequireAuth(h.sessions, h.store)
	admin := middleware.Chain(authMw, middleware.RequireAdmin())
	staff := middleware.Chain(authMw, middleware.RequireStaff())
	adopter := middleware.Chain(authMw, middleware.RequireRole(model.RoleAdopter, model.RoleAdmin))

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.Handle("POST /api/auth/logout", authMw(http.HandlerFunc(h.Logout)))
	mux.Handle("GET /api/auth/me", authMw(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/me/profile", authMw(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("PUT /api/me/password", authMw(http.HandlerFunc(h.ChangePassword)))

	mux.Handle("GET /api/species", authMw(http.HandlerFunc(h.Catalog)))
	mux.Handle("GET /api/shelters", authMw(http.HandlerFunc(h.ListShelters)))
	mux.Handle("POST /api/shelters", admin(http.HandlerFunc(h.CreateShelter)))
	mux.Handle("PUT /api/shelters/{id}", admin(http.HandlerFunc(h.UpdateShelter)))
	mux.Handle("POST /api/shelters/{id}/active", admin(http.HandlerFunc(h.SetShelterActive)))

	mux.Handle("GET /api/users", admin(http.HandlerFunc(h.ListUsers)))
	mux.Handle("POST /api/users", admin(http.HandlerFunc(h.CreateUser)))
	mux.Handle("GET /api/users/{id}", authMw(http.HandlerFunc(h.GetUser)))
	mux.Handle("POST /api/users/{id}/freeze", admin(http.HandlerFunc(h.FreezeUser)))
	mux.Handle("POST /api/users/{id}/unfreeze", admin(http.HandlerFunc(h.UnfreezeUser)))
	mux.Handle("POST /api/users/{id}/credit", admin(http.HandlerFunc(h.AdjustCredit)))

	mux.Handle("GET /api/pets", authMw(http.HandlerFunc(h.ListPets)))
	mux.Handle("POST /api/pets", staff(http.HandlerFunc(h.CreatePet)))
	mux.Handle("GET /api/pets/{id}", authMw(http.HandlerFunc(h.GetPet)))
	mux.Handle("PUT /api/pets/{id}", staff(http.HandlerFunc(h.UpdatePet)))
	mux.Handle("POST /api/pets/{id}/publish", staff(http.HandlerFunc(h.PublishPet)))
	mux.Handle("POST /api/pets/{id}/unpublish", staff(http.HandlerFunc(h.UnpublishPet)))
	mux.Handle("POST /api/pets/{id}/deceased", staff(http.HandlerFunc(h.MarkDeceased)))
	mux.Handle("POST /api/pets/{id}/apply", adopter(http.HandlerFunc(h.ApplyPet)))
	mux.Handle("GET /api/pets/{id}/applications", staff(http.HandlerFunc(h.ListPetApplications)))
	mux.Handle("GET /api/pets/{id}/health", authMw(http.HandlerFunc(h.ListHealth)))
	mux.Handle("POST /api/pets/{id}/health", staff(http.HandlerFunc(h.AddHealth)))
	mux.Handle("POST /api/pets/{id}/favorite", authMw(http.HandlerFunc(h.Favorite)))
	mux.Handle("DELETE /api/pets/{id}/favorite", authMw(http.HandlerFunc(h.Unfavorite)))
	mux.Handle("GET /api/pets/{id}/inquiries", authMw(http.HandlerFunc(h.ListInquiries)))
	mux.Handle("POST /api/pets/{id}/inquiries", authMw(http.HandlerFunc(h.AskInquiry)))

	mux.Handle("GET /api/applications/{id}", authMw(http.HandlerFunc(h.GetApplication)))
	mux.Handle("POST /api/applications/{id}/review", staff(http.HandlerFunc(h.StartReview)))
	mux.Handle("POST /api/applications/{id}/approve", staff(http.HandlerFunc(h.ApproveApplication)))
	mux.Handle("POST /api/applications/{id}/reject", staff(http.HandlerFunc(h.RejectApplication)))
	mux.Handle("POST /api/applications/{id}/withdraw", authMw(http.HandlerFunc(h.WithdrawApplication)))
	mux.Handle("POST /api/applications/{id}/handover", staff(http.HandlerFunc(h.Handover)))
	mux.Handle("POST /api/applications/{id}/return", authMw(http.HandlerFunc(h.ReturnAdoption)))
	mux.Handle("GET /api/me/applications", authMw(http.HandlerFunc(h.MyApplications)))
	mux.Handle("GET /api/staff/applications", staff(http.HandlerFunc(h.StaffApplications)))

	mux.Handle("GET /api/visits", authMw(http.HandlerFunc(h.ListVisits)))
	mux.Handle("POST /api/visits", staff(http.HandlerFunc(h.ScheduleVisit)))
	mux.Handle("GET /api/visits/{id}", authMw(http.HandlerFunc(h.GetVisit)))
	mux.Handle("POST /api/visits/{id}/complete", staff(http.HandlerFunc(h.CompleteVisit)))
	mux.Handle("POST /api/visits/{id}/miss", staff(http.HandlerFunc(h.MissVisit)))
	mux.Handle("POST /api/visits/{id}/cancel", staff(http.HandlerFunc(h.CancelVisit)))
	mux.Handle("POST /api/visits/{id}/comment", authMw(http.HandlerFunc(h.CommentVisit)))
	mux.Handle("GET /api/me/visits", authMw(http.HandlerFunc(h.MyVisits)))

	mux.Handle("POST /api/inquiries/{id}/reply", staff(http.HandlerFunc(h.ReplyInquiry)))
	mux.Handle("DELETE /api/inquiries/{id}", authMw(http.HandlerFunc(h.DeleteInquiry)))
	mux.Handle("GET /api/me/favorites", authMw(http.HandlerFunc(h.MyFavorites)))
	mux.Handle("GET /api/me/notifications", authMw(http.HandlerFunc(h.MyNotifications)))
	mux.Handle("POST /api/me/notifications/{id}/read", authMw(http.HandlerFunc(h.ReadNotification)))
	mux.Handle("POST /api/me/notifications/read-all", authMw(http.HandlerFunc(h.ReadAllNotifications)))
	mux.Handle("GET /api/me/credit-logs", authMw(http.HandlerFunc(h.MyCreditLogs)))

	mux.Handle("GET /api/stats", admin(http.HandlerFunc(h.GlobalStats)))
	mux.Handle("GET /api/staff/board", staff(http.HandlerFunc(h.StaffBoard)))
	mux.Handle("GET /api/audit", admin(http.HandlerFunc(h.ListAudit)))

	h.registerPageRoutes(mux)
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, model.HealthResponse{Status: "ok"})
}
