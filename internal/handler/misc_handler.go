package handler

import (
	"net/http"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/policy"
	"go04-pet-adoption/internal/respond"
)

func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	respond.OK(w, policy.Catalog())
}

func (h *Handler) ListShelters(w http.ResponseWriter, r *http.Request) {
	activeOnly := parseBoolQuery(r, "active", true)
	out, err := h.services.User.ListShelters(r.Context(), activeOnly)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) CreateShelter(w http.ResponseWriter, r *http.Request) {
	var in model.ShelterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.CreateShelter(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) UpdateShelter(w http.ResponseWriter, r *http.Request) {
	var in model.ShelterInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.UpdateShelter(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) SetShelterActive(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Active bool `json:"active"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.User.SetShelterActive(r.Context(), userFrom(r), pathID(r), in.Active)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Favorite(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Social.Favorite(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) Unfavorite(w http.ResponseWriter, r *http.Request) {
	if err := h.services.Social.Unfavorite(r.Context(), userFrom(r), pathID(r)); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) MyFavorites(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Social.MyFavorites(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) ListInquiries(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Social.ListInquiries(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) AskInquiry(w http.ResponseWriter, r *http.Request) {
	var in model.InquiryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Social.Ask(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ReplyInquiry(w http.ResponseWriter, r *http.Request) {
	var in model.InquiryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Social.Reply(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) DeleteInquiry(w http.ResponseWriter, r *http.Request) {
	if err := h.services.Social.DeleteInquiry(r.Context(), userFrom(r), pathID(r)); err != nil {
		writeErr(w, err)
		return
	}
	respond.NoContent(w)
}

func (h *Handler) MyNotifications(w http.ResponseWriter, r *http.Request) {
	unread := parseBoolQuery(r, "unread", false)
	out, err := h.services.Notify.List(r.Context(), userFrom(r), unread)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Notify.Read(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ReadAllNotifications(w http.ResponseWriter, r *http.Request) {
	n, err := h.services.Notify.ReadAll(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]int{"read": n})
}

func (h *Handler) MyCreditLogs(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.User.CreditLogs(r.Context(), userFrom(r), queryStr(r, "user_id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) GlobalStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Stats.Global(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) StaffBoard(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Stats.StaffBoard(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ListAudit(w http.ResponseWriter, r *http.Request) {
	out, err := h.store.ListAuditLogs(r.Context(), queryStr(r, "target_id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}
