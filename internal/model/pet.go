package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Species string

const (
	SpeciesDog    Species = "dog"
	SpeciesCat    Species = "cat"
	SpeciesRabbit Species = "rabbit"
	SpeciesBird   Species = "bird"
	SpeciesOther  Species = "other"
)

func (s Species) IsValid() bool {
	switch s {
	case SpeciesDog, SpeciesCat, SpeciesRabbit, SpeciesBird, SpeciesOther:
		return true
	}
	return false
}

func (s Species) Label() string {
	switch s {
	case SpeciesDog:
		return "狗"
	case SpeciesCat:
		return "猫"
	case SpeciesRabbit:
		return "兔"
	case SpeciesBird:
		return "鸟"
	default:
		return "其他"
	}
}

type Size string

const (
	SizeSmall  Size = "small"
	SizeMedium Size = "medium"
	SizeLarge  Size = "large"
)

func (s Size) IsValid() bool {
	return s == SizeSmall || s == SizeMedium || s == SizeLarge
}

type Sex string

const (
	SexMale    Sex = "male"
	SexFemale  Sex = "female"
	SexUnknown Sex = "unknown"
)

func (s Sex) IsValid() bool {
	return s == SexMale || s == SexFemale || s == SexUnknown || s == ""
}

func (s Sex) Normalize() Sex {
	if s == "" {
		return SexUnknown
	}
	return s
}

type PetStatus string

const (
	PetDraft       PetStatus = "draft"
	PetPublished   PetStatus = "published"
	PetReserved    PetStatus = "reserved"
	PetAdopted     PetStatus = "adopted"
	PetUnavailable PetStatus = "unavailable"
	PetDeceased    PetStatus = "deceased"
)

func (s PetStatus) IsValid() bool {
	switch s {
	case PetDraft, PetPublished, PetReserved, PetAdopted, PetUnavailable, PetDeceased:
		return true
	}
	return false
}

func (s PetStatus) IsTerminal() bool {
	return s == PetDeceased
}

func (s PetStatus) CanReceiveApplication() bool {
	return s == PetPublished
}

func (s PetStatus) CanPublish() bool {
	return s == PetDraft || s == PetUnavailable
}

type Pet struct {
	ID              string    `json:"id"`
	ShelterID       string    `json:"shelter_id"`
	StaffID         string    `json:"staff_id"`
	Name            string    `json:"name"`
	Species         Species   `json:"species"`
	Breed           string    `json:"breed,omitempty"`
	Size            Size      `json:"size"`
	Sex             Sex       `json:"sex"`
	AgeMonths       int       `json:"age_months"`
	Color           string    `json:"color,omitempty"`
	Sterilized      bool      `json:"sterilized"`
	Vaccinated      bool      `json:"vaccinated"`
	SpecialNeeds    bool      `json:"special_needs"`
	Personality     []string  `json:"personality,omitempty"`
	Story           string    `json:"story"`
	CoverURL        string    `json:"cover_url,omitempty"`
	Photos          []string  `json:"photos,omitempty"`
	AllowApartment  bool      `json:"allow_apartment"`
	AllowChildren   bool      `json:"allow_children"`
	AllowOtherPets  bool      `json:"allow_other_pets"`
	MinAdopterAge   int       `json:"min_adopter_age"`
	NeedHomeCheck   bool      `json:"need_home_check"`
	Status          PetStatus `json:"status"`
	ReservedAppID   string    `json:"reserved_app_id,omitempty"`
	AdoptedBy       string    `json:"adopted_by,omitempty"`
	AdoptedAt       *time.Time `json:"adopted_at,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`
	UnavailableNote string    `json:"unavailable_note,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
}

func (p Pet) AgeYearsLabel() string {
	if p.AgeMonths < 12 {
		return "幼年"
	}
	if p.AgeMonths < 84 {
		return "成年"
	}
	return "老年"
}

func (p Pet) MatchesAdopter(housing HousingType, hasChildren, hasOtherPets bool, ageYears int) error {
	if ageYears > 0 && p.MinAdopterAge > 0 && ageYears < p.MinAdopterAge {
		return ErrRequirementNotMet
	}
	if housing == HousingApartment && !p.AllowApartment {
		return ErrRequirementNotMet
	}
	if hasChildren && !p.AllowChildren {
		return ErrRequirementNotMet
	}
	if hasOtherPets && !p.AllowOtherPets {
		return ErrRequirementNotMet
	}
	return nil
}

func (p Pet) Public() Pet {
	return p
}

type PetInput struct {
	Name           string   `json:"name"`
	Species        Species  `json:"species"`
	Breed          string   `json:"breed"`
	Size           Size     `json:"size"`
	Sex            Sex      `json:"sex"`
	AgeMonths      int      `json:"age_months"`
	Color          string   `json:"color"`
	Sterilized     bool     `json:"sterilized"`
	Vaccinated     bool     `json:"vaccinated"`
	SpecialNeeds   bool     `json:"special_needs"`
	Personality    []string `json:"personality"`
	Story          string   `json:"story"`
	CoverURL       string   `json:"cover_url"`
	Photos         []string `json:"photos"`
	AllowApartment bool     `json:"allow_apartment"`
	AllowChildren  bool     `json:"allow_children"`
	AllowOtherPets bool     `json:"allow_other_pets"`
	MinAdopterAge  int      `json:"min_adopter_age"`
	NeedHomeCheck  *bool    `json:"need_home_check"`
	Publish        bool     `json:"publish"`
}

func (in *PetInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.Breed = strings.TrimSpace(in.Breed)
	in.Color = strings.TrimSpace(in.Color)
	in.Story = strings.TrimSpace(in.Story)
	in.CoverURL = strings.TrimSpace(in.CoverURL)
	in.Sex = in.Sex.Normalize()
	if in.MinAdopterAge == 0 {
		in.MinAdopterAge = 18
	}
	cleaned := make([]string, 0, len(in.Personality))
	for _, t := range in.Personality {
		t = strings.TrimSpace(t)
		if t != "" {
			cleaned = append(cleaned, t)
		}
	}
	in.Personality = cleaned
	photos := make([]string, 0, len(in.Photos))
	for _, u := range in.Photos {
		u = strings.TrimSpace(u)
		if u != "" {
			photos = append(photos, u)
		}
	}
	in.Photos = photos
}

func (in PetInput) Validate() error {
	in.Normalize()
	n := utf8.RuneCountInString(in.Name)
	if n < 1 || n > 40 {
		return ErrInvalidName
	}
	if !in.Species.IsValid() {
		return ErrInvalidSpecies
	}
	if !in.Size.IsValid() {
		return ErrInvalidSize
	}
	if !in.Sex.IsValid() {
		return ErrInvalidSex
	}
	if in.AgeMonths < 0 || in.AgeMonths > 360 {
		return ErrInvalidAge
	}
	sn := utf8.RuneCountInString(in.Story)
	if sn < 1 || sn > 4000 {
		return ErrInvalidStory
	}
	if utf8.RuneCountInString(in.Breed) > 40 {
		return ErrInvalidName
	}
	if utf8.RuneCountInString(in.Color) > 32 {
		return ErrInvalidName
	}
	if in.MinAdopterAge < 18 || in.MinAdopterAge > 80 {
		return ErrInvalidMinAge
	}
	if len(in.Personality) > 12 {
		return ErrValidation
	}
	if len(in.Photos) > 8 {
		return ErrValidation
	}
	return nil
}

type PetFilter struct {
	Status         PetStatus
	Species        Species
	Size           Size
	ShelterID      string
	Sterilized     *bool
	Vaccinated     *bool
	SpecialNeeds   *bool
	AllowApartment *bool
	Query          string
	StaffID        string
	AdoptedBy      string
}

type UnpublishInput struct {
	Unavailable bool   `json:"unavailable"`
	Note        string `json:"note"`
}
