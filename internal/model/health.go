package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

type HealthKind string

const (
	HealthVaccine   HealthKind = "vaccine"
	HealthDeworm    HealthKind = "deworm"
	HealthSterilize HealthKind = "sterilize"
	HealthCheckup   HealthKind = "checkup"
	HealthTreatment HealthKind = "treatment"
	HealthOther     HealthKind = "other"
)

func (k HealthKind) IsValid() bool {
	switch k {
	case HealthVaccine, HealthDeworm, HealthSterilize, HealthCheckup, HealthTreatment, HealthOther:
		return true
	}
	return false
}

type HealthRecord struct {
	ID         string     `json:"id"`
	PetID      string     `json:"pet_id"`
	StaffID    string     `json:"staff_id"`
	Kind       HealthKind `json:"kind"`
	Title      string     `json:"title"`
	Detail     string     `json:"detail,omitempty"`
	OccurredAt time.Time  `json:"occurred_at"`
	NextDueAt  *time.Time `json:"next_due_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type HealthInput struct {
	Kind       HealthKind `json:"kind"`
	Title      string     `json:"title"`
	Detail     string     `json:"detail"`
	OccurredAt time.Time  `json:"occurred_at"`
	NextDueAt  *time.Time `json:"next_due_at"`
}

func (in *HealthInput) Normalize() {
	in.Title = strings.TrimSpace(in.Title)
	in.Detail = strings.TrimSpace(in.Detail)
}

func (in HealthInput) Validate() error {
	in.Normalize()
	if !in.Kind.IsValid() {
		return ErrValidation
	}
	n := utf8.RuneCountInString(in.Title)
	if n < 1 || n > 80 {
		return ErrInvalidName
	}
	if utf8.RuneCountInString(in.Detail) > 1000 {
		return ErrInvalidComment
	}
	if in.OccurredAt.IsZero() {
		return ErrValidation
	}
	if in.NextDueAt != nil && in.NextDueAt.Before(in.OccurredAt) {
		return ErrValidation
	}
	return nil
}

type ShelterStatus string

const (
	ShelterActive   ShelterStatus = "active"
	ShelterInactive ShelterStatus = "inactive"
)

func (s ShelterStatus) IsValid() bool {
	return s == ShelterActive || s == ShelterInactive || s == ""
}

func (s ShelterStatus) Normalize() ShelterStatus {
	if s == "" {
		return ShelterActive
	}
	return s
}

type Shelter struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	City      string        `json:"city,omitempty"`
	Address   string        `json:"address,omitempty"`
	Phone     string        `json:"phone,omitempty"`
	Status    ShelterStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (s Shelter) IsActive() bool { return s.Status.Normalize() == ShelterActive }

type ShelterInput struct {
	Name    string `json:"name"`
	City    string `json:"city"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

func (in *ShelterInput) Normalize() {
	in.Name = strings.TrimSpace(in.Name)
	in.City = strings.TrimSpace(in.City)
	in.Address = strings.TrimSpace(in.Address)
	in.Phone = strings.TrimSpace(in.Phone)
}

func (in ShelterInput) Validate() error {
	in.Normalize()
	n := utf8.RuneCountInString(in.Name)
	if n < 1 || n > 40 {
		return ErrInvalidName
	}
	if utf8.RuneCountInString(in.City) > 32 {
		return ErrInvalidCity
	}
	if utf8.RuneCountInString(in.Address) > 120 {
		return ErrInvalidCity
	}
	if utf8.RuneCountInString(in.Phone) > 20 {
		return ErrInvalidPhone
	}
	return nil
}
