package handler

import (
	"net/http"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/respond"
)

func (h *Handler) ApplyPet(w http.ResponseWriter, r *http.Request) {
	var in model.ApplyInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.App.Apply(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) ListPetApplications(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.App.ListByPet(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) GetApplication(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.App.Get(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) StartReview(w http.ResponseWriter, r *http.Request) {
	var in model.ReviewInput
	if !decodeJSONOptional(w, r, &in) {
		return
	}
	out, err := h.services.App.StartReview(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ApproveApplication(w http.ResponseWriter, r *http.Request) {
	var in struct {
		model.ReviewInput
		Force bool `json:"force"`
	}
	if !decodeJSONOptional(w, r, &in) {
		return
	}
	out, err := h.services.App.Approve(r.Context(), userFrom(r), pathID(r), in.ReviewInput, in.Force)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) RejectApplication(w http.ResponseWriter, r *http.Request) {
	var in model.RejectInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.App.Reject(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) WithdrawApplication(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.App.Withdraw(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) Handover(w http.ResponseWriter, r *http.Request) {
	var in model.HandoverInput
	if !decodeJSONOptional(w, r, &in) {
		return
	}
	app, visits, err := h.services.App.Handover(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"application": app, "visits": visits})
}

func (h *Handler) ReturnAdoption(w http.ResponseWriter, r *http.Request) {
	var in model.ReturnInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.App.Return(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyApplications(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.App.ListMine(r.Context(), userFrom(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) StaffApplications(w http.ResponseWriter, r *http.Request) {
	status := model.ApplicationStatus(queryStr(r, "status"))
	out, err := h.services.App.ListForStaff(r.Context(), userFrom(r), status)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}
