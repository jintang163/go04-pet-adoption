package handler

import (
	"net/http"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/respond"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in model.UserInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Auth.Register(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in model.LoginInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Auth.Login(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.services.Auth.Logout(extractBearer(r))
	respond.NoContent(w)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Auth.Me(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var in model.ProfileInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.UpdateProfile(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in model.PasswordChangeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := h.services.User.ChangePassword(r.Context(), userFrom(r), in); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	f := model.UserFilter{
		Role:      model.UserRole(queryStr(r, "role")),
		Status:    model.UserStatus(queryStr(r, "status")),
		ShelterID: queryStr(r, "shelter_id"),
		Query:     queryStr(r, "q"),
	}
	out, err := h.services.User.List(r.Context(), userFrom(r), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in model.UserInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.Create(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.services.User.GetByID(r.Context(), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, u.Public())
}

func (h *Handler) FreezeUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.Freeze(r.Context(), userFrom(r), pathID(r), true)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UnfreezeUser(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.Freeze(r.Context(), userFrom(r), pathID(r), false)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) AdjustCredit(w http.ResponseWriter, r *http.Request) {
	var in model.CreditAdjustInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.AdjustCredit(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}
