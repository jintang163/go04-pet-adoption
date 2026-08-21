package handler

import (
	"net/http"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/respond"
)

func (h *Handler) ListPets(w http.ResponseWriter, r *http.Request) {
	f := model.PetFilter{
		Status:         model.PetStatus(queryStr(r, "status")),
		Species:        model.Species(queryStr(r, "species")),
		Size:           model.Size(queryStr(r, "size")),
		ShelterID:      queryStr(r, "shelter_id"),
		Query:          queryStr(r, "q"),
		Sterilized:     optionalBoolQuery(r, "sterilized"),
		Vaccinated:     optionalBoolQuery(r, "vaccinated"),
		SpecialNeeds:   optionalBoolQuery(r, "special_needs"),
		AllowApartment: optionalBoolQuery(r, "allow_apartment"),
	}
	out, err := h.services.Pet.List(r.Context(), userFrom(r), f)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) CreatePet(w http.ResponseWriter, r *http.Request) {
	var in model.PetInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Pet.Create(r.Context(), userFrom(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}

func (h *Handler) GetPet(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Pet.Get(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UpdatePet(w http.ResponseWriter, r *http.Request) {
	var in model.PetInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Pet.Update(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) PublishPet(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Pet.Publish(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) UnpublishPet(w http.ResponseWriter, r *http.Request) {
	var in model.UnpublishInput
	if !decodeJSONOptional(w, r, &in) {
		return
	}
	out, err := h.services.Pet.Unpublish(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) MarkDeceased(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Note string `json:"note"`
	}
	if !decodeJSONOptional(w, r, &in) {
		return
	}
	out, err := h.services.Pet.MarkDeceased(r.Context(), userFrom(r), pathID(r), in.Note)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, out)
}

func (h *Handler) ListHealth(w http.ResponseWriter, r *http.Request) {
	out, err := h.services.Health.List(r.Context(), userFrom(r), pathID(r))
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.OK(w, map[string]any{"items": out, "total": len(out)})
}

func (h *Handler) AddHealth(w http.ResponseWriter, r *http.Request) {
	var in model.HealthInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.services.Health.Add(r.Context(), userFrom(r), pathID(r), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond.Created(w, out)
}
