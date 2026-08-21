package service

import (
	"context"
	"sort"
	"strings"

	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/store"
	"go04-pet-adoption/internal/validate"
)

type PetService struct {
	store  store.Store
	notify *NotifyService
	clock  Clock
}

func NewPetService(s store.Store, notify *NotifyService, clock Clock) *PetService {
	return &PetService{store: s, notify: notify, clock: clock}
}

func (svc *PetService) Create(ctx context.Context, actor model.User, in model.PetInput) (model.Pet, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Pet{}, err
	}
	if !actor.IsStaff() && !actor.IsAdmin() {
		return model.Pet{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Pet{}, err
	}
	shelterID := actor.ShelterID
	if actor.IsAdmin() && shelterID == "" {
		list, err := svc.store.ListShelters(ctx, true)
		if err != nil {
			return model.Pet{}, err
		}
		if len(list) == 0 {
			return model.Pet{}, model.ErrStaffShelterRequired
		}
		sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt.Before(list[j].CreatedAt) })
		shelterID = list[0].ID
	}
	if shelterID == "" {
		return model.Pet{}, model.ErrStaffShelterRequired
	}
	sh, err := svc.store.GetShelter(ctx, shelterID)
	if err != nil {
		return model.Pet{}, err
	}
	if !sh.IsActive() {
		return model.Pet{}, model.ErrShelterInactive
	}
	now := svc.clock.Now()
	need := true
	if in.NeedHomeCheck != nil {
		need = *in.NeedHomeCheck
	}
	p := model.Pet{
		ShelterID:      shelterID,
		StaffID:        actor.ID,
		Name:           in.Name,
		Species:        in.Species,
		Breed:          in.Breed,
		Size:           in.Size,
		Sex:            in.Sex,
		AgeMonths:      in.AgeMonths,
		Color:          in.Color,
		Sterilized:     in.Sterilized,
		Vaccinated:     in.Vaccinated,
		SpecialNeeds:   in.SpecialNeeds,
		Personality:    in.Personality,
		Story:          in.Story,
		CoverURL:       in.CoverURL,
		Photos:         in.Photos,
		AllowApartment: in.AllowApartment,
		AllowChildren:  in.AllowChildren,
		AllowOtherPets: in.AllowOtherPets,
		MinAdopterAge:  in.MinAdopterAge,
		NeedHomeCheck:  need,
		Status:         model.PetDraft,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	created, err := svc.store.CreatePet(ctx, p)
	if err != nil {
		return model.Pet{}, err
	}
	if in.Publish {
		return svc.Publish(ctx, actor, created.ID)
	}
	return created, nil
}

func (svc *PetService) Update(ctx context.Context, actor model.User, id string, in model.PetInput) (model.Pet, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Pet{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Pet{}, err
	}
	p, err := svc.store.GetPet(ctx, id)
	if err != nil {
		return model.Pet{}, err
	}
	if !canManagePet(actor, p) {
		return model.Pet{}, model.ErrNotPetStaff
	}
	if p.Status != model.PetDraft && p.Status != model.PetPublished && p.Status != model.PetUnavailable {
		return model.Pet{}, model.ErrConflict
	}
	p.Name = in.Name
	p.Species = in.Species
	p.Breed = in.Breed
	p.Size = in.Size
	p.Sex = in.Sex
	p.AgeMonths = in.AgeMonths
	p.Color = in.Color
	p.Sterilized = in.Sterilized
	p.Vaccinated = in.Vaccinated
	p.SpecialNeeds = in.SpecialNeeds
	p.Personality = in.Personality
	p.Story = in.Story
	p.CoverURL = in.CoverURL
	p.Photos = in.Photos
	p.AllowApartment = in.AllowApartment
	p.AllowChildren = in.AllowChildren
	p.AllowOtherPets = in.AllowOtherPets
	p.MinAdopterAge = in.MinAdopterAge
	if in.NeedHomeCheck != nil {
		p.NeedHomeCheck = *in.NeedHomeCheck
	}
	return svc.store.UpdatePet(ctx, p)
}

func (svc *PetService) Publish(ctx context.Context, actor model.User, id string) (model.Pet, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Pet{}, err
	}
	p, err := svc.store.GetPet(ctx, id)
	if err != nil {
		return model.Pet{}, err
	}
	if !canManagePet(actor, p) {
		return model.Pet{}, model.ErrNotPetStaff
	}
	if !p.Status.CanPublish() {
		return model.Pet{}, model.ErrConflict
	}
	now := svc.clock.Now()
	p.Status = model.PetPublished
	p.PublishedAt = &now
	p.UnavailableNote = ""
	updated, err := svc.store.UpdatePet(ctx, p)
	if err != nil {
		return model.Pet{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditPetPublish, updated.ID, updated.Name, now)
	return updated, nil
}

func (svc *PetService) Unpublish(ctx context.Context, actor model.User, id string, in model.UnpublishInput) (model.Pet, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Pet{}, err
	}
	p, err := svc.store.GetPet(ctx, id)
	if err != nil {
		return model.Pet{}, err
	}
	if !canManagePet(actor, p) {
		return model.Pet{}, model.ErrNotPetStaff
	}
	if p.Status != model.PetPublished && p.Status != model.PetDraft {
		if !(actor.IsAdmin() && (p.Status == model.PetReserved || p.Status == model.PetUnavailable)) {
			return model.Pet{}, model.ErrConflict
		}
	}
	if in.Unavailable {
		p.Status = model.PetUnavailable
		p.UnavailableNote = strings.TrimSpace(in.Note)
	} else {
		p.Status = model.PetDraft
	}
	updated, err := svc.store.UpdatePet(ctx, p)
	if err != nil {
		return model.Pet{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditPetUnpublish, updated.ID, in.Note, svc.clock.Now())
	return updated, nil
}

func (svc *PetService) MarkDeceased(ctx context.Context, actor model.User, id, note string) (model.Pet, error) {
	if !actor.IsAdmin() && !actor.IsStaff() {
		return model.Pet{}, model.ErrForbidden
	}
	p, err := svc.store.GetPet(ctx, id)
	if err != nil {
		return model.Pet{}, err
	}
	if !canManagePet(actor, p) {
		return model.Pet{}, model.ErrNotPetStaff
	}
	p.Status = model.PetDeceased
	p.UnavailableNote = strings.TrimSpace(note)
	updated, err := svc.store.UpdatePet(ctx, p)
	if err != nil {
		return model.Pet{}, err
	}
	audit(ctx, svc.store, actor.ID, model.AuditForce, updated.ID, "deceased", svc.clock.Now())
	return updated, nil
}

func (svc *PetService) Get(ctx context.Context, actor model.User, id string) (model.PetView, error) {
	p, err := svc.store.GetPet(ctx, id)
	if err != nil {
		return model.PetView{}, err
	}
	if p.Status == model.PetDraft && !canManagePet(actor, p) {
		return model.PetView{}, model.ErrNotFound
	}
	return svc.toView(ctx, actor, p), nil
}

func (svc *PetService) List(ctx context.Context, actor model.User, f model.PetFilter) ([]model.PetView, error) {
	if f.Status == "" && !actor.IsStaff() && !actor.IsAdmin() {
		f.Status = model.PetPublished
	}
	if actor.IsStaff() && !actor.IsAdmin() && f.ShelterID == "" {
		f.ShelterID = actor.ShelterID
	}
	pets, err := svc.store.ListPets(ctx, f)
	if err != nil {
		return nil, err
	}
	sort.Slice(pets, func(i, j int) bool {
		if pets[i].UpdatedAt.Equal(pets[j].UpdatedAt) {
			return pets[i].Name < pets[j].Name
		}
		return pets[i].UpdatedAt.After(pets[j].UpdatedAt)
	})
	out := make([]model.PetView, 0, len(pets))
	for _, p := range pets {
		if p.Status == model.PetDraft && !canManagePet(actor, p) {
			continue
		}
		out = append(out, svc.toView(ctx, actor, p))
	}
	return out, nil
}

func (svc *PetService) toView(ctx context.Context, actor model.User, p model.Pet) model.PetView {
	v := model.PetView{Pet: p}
	if sh, err := svc.store.GetShelter(ctx, p.ShelterID); err == nil {
		v.ShelterName = sh.Name
	}
	if st, err := svc.store.GetUserByID(ctx, p.StaffID); err == nil {
		v.StaffName = st.DisplayName
	}
	if hs, err := svc.store.ListHealthByPet(ctx, p.ID); err == nil {
		v.HealthCount = len(hs)
	}
	if qs, err := svc.store.ListInquiriesByPet(ctx, p.ID); err == nil {
		v.InquiryCount = len(qs)
	}
	if actor.ID != "" {
		if _, err := svc.store.GetFavorite(ctx, actor.ID, p.ID); err == nil {
			v.Favorited = true
		}
		if app, err := svc.store.GetApplicationByPetApplicant(ctx, p.ID, actor.ID); err == nil && app.Status.IsActive() {
			cp := app
			v.MyApplication = &cp
		}
	}
	return v
}

func (svc *PetService) SearchPublished(ctx context.Context, actor model.User, q string, species model.Species, size model.Size) ([]model.PetView, error) {
	f := model.PetFilter{
		Status:  model.PetPublished,
		Species: species,
		Size:    size,
		Query:   validate.Trim(q),
	}
	return svc.List(ctx, actor, f)
}
