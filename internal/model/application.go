package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

type ApplicationStatus string

const (
	AppPending      ApplicationStatus = "pending"
	AppUnderReview  ApplicationStatus = "under_review"
	AppWaitlisted   ApplicationStatus = "waitlisted"
	AppApproved     ApplicationStatus = "approved"
	AppRejected     ApplicationStatus = "rejected"
	AppWithdrawn    ApplicationStatus = "withdrawn"
	AppCompleted    ApplicationStatus = "completed"
	AppRevoked      ApplicationStatus = "revoked"
)

func (s ApplicationStatus) IsValid() bool {
	switch s {
	case AppPending, AppUnderReview, AppWaitlisted, AppApproved, AppRejected, AppWithdrawn, AppCompleted, AppRevoked:
		return true
	}
	return false
}

func (s ApplicationStatus) IsActive() bool {
	switch s {
	case AppPending, AppUnderReview, AppWaitlisted, AppApproved:
		return true
	}
	return false
}

func (s ApplicationStatus) CountsTowardLimit() bool {
	switch s {
	case AppPending, AppUnderReview, AppApproved:
		return true
	}
	return false
}

func (s ApplicationStatus) CanWithdraw() bool {
	return s == AppPending || s == AppUnderReview || s == AppWaitlisted || s == AppApproved
}

func (s ApplicationStatus) CanReview() bool {
	return s == AppPending || s == AppUnderReview
}

func (s ApplicationStatus) CanApprove() bool {
	return s == AppPending || s == AppUnderReview || s == AppWaitlisted
}

type Application struct {
	ID            string            `json:"id"`
	PetID         string            `json:"pet_id"`
	ApplicantID   string            `json:"applicant_id"`
	ShelterID     string            `json:"shelter_id"`
	Status        ApplicationStatus `json:"status"`
	Housing       HousingType       `json:"housing"`
	AreaSqm       int               `json:"area_sqm,omitempty"`
	HasChildren   bool              `json:"has_children"`
	HasOtherPets  bool              `json:"has_other_pets"`
	HoursAlone    int               `json:"hours_alone"`
	Experience    ExperienceLevel   `json:"experience"`
	Phone         string            `json:"phone"`
	Intro         string            `json:"intro"`
	WaitlistRank  int               `json:"waitlist_rank,omitempty"`
	ReviewerID    string            `json:"reviewer_id,omitempty"`
	ReviewNote    string            `json:"review_note,omitempty"`
	RejectReason  string            `json:"reject_reason,omitempty"`
	HomeCheckID   string            `json:"home_check_id,omitempty"`
	HandoverNote  string            `json:"handover_note,omitempty"`
	HandoverAt    *time.Time        `json:"handover_at,omitempty"`
	ReturnedAt    *time.Time        `json:"returned_at,omitempty"`
	ReturnReason  string            `json:"return_reason,omitempty"`
	ReturnMedical bool              `json:"return_medical,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	ReviewedAt    *time.Time        `json:"reviewed_at,omitempty"`
}

type ApplyInput struct {
	Housing      HousingType     `json:"housing"`
	AreaSqm      int             `json:"area_sqm"`
	HasChildren  bool            `json:"has_children"`
	HasOtherPets bool            `json:"has_other_pets"`
	HoursAlone   int             `json:"hours_alone"`
	Experience   ExperienceLevel `json:"experience"`
	Phone        string          `json:"phone"`
	Intro        string          `json:"intro"`
}

func (in *ApplyInput) Normalize() {
	in.Phone = strings.TrimSpace(in.Phone)
	in.Intro = strings.TrimSpace(in.Intro)
	if in.Experience == "" {
		in.Experience = ExperienceFirstTime
	}
}

func (in ApplyInput) Validate() error {
	in.Normalize()
	if !in.Housing.IsValid() || in.Housing == "" {
		return ErrInvalidHousing
	}
	if !in.Experience.IsValid() {
		return ErrInvalidExperience
	}
	if in.HoursAlone < 0 || in.HoursAlone > 24 {
		return ErrInvalidHoursAlone
	}
	if in.AreaSqm < 0 || in.AreaSqm > 2000 {
		return ErrValidation
	}
	if in.Phone == "" || utf8.RuneCountInString(in.Phone) > 20 {
		return ErrInvalidPhone
	}
	n := utf8.RuneCountInString(in.Intro)
	if n < 1 || n > 500 {
		return ErrInvalidIntro
	}
	return nil
}

type ReviewInput struct {
	Note string `json:"note"`
}

type RejectInput struct {
	Reason string `json:"reason"`
}

func (in RejectInput) Validate() error {
	n := utf8.RuneCountInString(strings.TrimSpace(in.Reason))
	if n < 1 || n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type HandoverInput struct {
	Note     string `json:"note"`
	Checklist string `json:"checklist"`
}

func (in HandoverInput) Validate() error {
	if utf8.RuneCountInString(strings.TrimSpace(in.Note)) > 500 {
		return ErrInvalidComment
	}
	return nil
}

type ReturnInput struct {
	Reason  string `json:"reason"`
	Medical bool   `json:"medical"`
	Approve bool   `json:"approve"`
}

func (in ReturnInput) Validate() error {
	n := utf8.RuneCountInString(strings.TrimSpace(in.Reason))
	if n < 1 || n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type ApplicationFilter struct {
	PetID       string
	ApplicantID string
	ShelterID   string
	Status      ApplicationStatus
	StaffID     string
}
