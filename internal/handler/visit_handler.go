package handler

import (
	"net/http"
	"time"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/respond"
)

func (h *Handler) ListVisits(w http.ResponseWriter, r *http.Request) {
	f := model.VisitFilter{
		PetID:         queryStr(r, "pet_id"),
		ApplicationID: queryStr(r, "application_id"),
		AdopterID:     queryStr(r, "adopter_id"),
		Type:          model.VisitType(queryStr(r, "type")),
		Status:        model.VisitStatus(queryStr(r, "status")),
		ShelterID:     queryStr(r, "shelter_id"),
	}
	if ds := queryStr(r, "due_before"); ds != "" {
		if t, err := time.Parse(time.RFC3339, ds); err == nil {
			f.DueBefore = &t
		}
	}
	out, err := h.services.Visit.List(r.Context(), userFrom(r), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) ScheduleVisit(w http.ResponseWriter, r *http.Request) {
	var in model.ScheduleVisitInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Visit.Schedule(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetVisit(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Visit.Get(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CompleteVisit(w http.ResponseWriter, r *http.Request) {
	var in model.CompleteVisitInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Visit.Complete(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MissVisit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Note string `json:"note"`
	}
	if !decodeJSONOptional(w, r, &in) {
		return
	}
	out, err := h.services.Visit.Miss(r.Context(), userFrom(r), pathID(r), in.Note)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CancelVisit(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Visit.Cancel(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) CommentVisit(w http.ResponseWriter, r *http.Request) {
	var in model.VisitCommentInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Visit.Comment(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MyVisits(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Visit.List(r.Context(), userFrom(r), model.VisitFilter{AdopterID: userFrom(r).ID})
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}
