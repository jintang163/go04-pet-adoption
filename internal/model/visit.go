package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

type VisitType string

const (
	VisitHomeCheck VisitType = "home_check"
	VisitFollowup  VisitType = "followup"
	VisitExtra     VisitType = "extra"
)

func (t VisitType) IsValid() bool {
	return t == VisitHomeCheck || t == VisitFollowup || t == VisitExtra
}

type VisitStatus string

const (
	VisitScheduled VisitStatus = "scheduled"
	VisitCompleted VisitStatus = "completed"
	VisitMissed    VisitStatus = "missed"
	VisitCancelled VisitStatus = "cancelled"
)

func (s VisitStatus) IsValid() bool {
	return s == VisitScheduled || s == VisitCompleted || s == VisitMissed || s == VisitCancelled
}

func (s VisitStatus) IsOpen() bool {
	return s == VisitScheduled
}

type Visit struct {
	ID             string      `json:"id"`
	PetID          string      `json:"pet_id"`
	ApplicationID  string      `json:"application_id"`
	AdopterID      string      `json:"adopter_id"`
	StaffID        string      `json:"staff_id,omitempty"`
	ShelterID      string      `json:"shelter_id"`
	Type           VisitType   `json:"type"`
	Status         VisitStatus `json:"status"`
	Sequence       int         `json:"sequence,omitempty"`
	ScheduledAt    time.Time   `json:"scheduled_at"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty"`
	Location       string      `json:"location,omitempty"`
	LivingScore    int         `json:"living_score,omitempty"`
	HealthScore    int         `json:"health_score,omitempty"`
	BehaviorScore  int         `json:"behavior_score,omitempty"`
	RiskFlag       bool        `json:"risk_flag"`
	Notes          string      `json:"notes,omitempty"`
	Issues         string      `json:"issues,omitempty"`
	Suggestion     string      `json:"suggestion,omitempty"`
	AdopterComment string      `json:"adopter_comment,omitempty"`
	FollowUpNeeded bool        `json:"follow_up_needed"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

func (v Visit) AverageScore() float64 {
	n := 0
	sum := 0
	for _, s := range []int{v.LivingScore, v.HealthScore, v.BehaviorScore} {
		if s > 0 {
			sum += s
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(sum) / float64(n)
}

func (v Visit) PassedHomeCheck(minScore int) bool {
	if v.Type != VisitHomeCheck || v.Status != VisitCompleted {
		return false
	}
	if v.RiskFlag {
		return false
	}
	return v.LivingScore >= minScore && v.HealthScore >= minScore && v.BehaviorScore >= minScore
}

type ScheduleVisitInput struct {
	PetID         string    `json:"pet_id"`
	ApplicationID string    `json:"application_id"`
	Type          VisitType `json:"type"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	Location      string    `json:"location"`
	StaffID       string    `json:"staff_id"`
}

func (in *ScheduleVisitInput) Normalize() {
	in.PetID = strings.TrimSpace(in.PetID)
	in.ApplicationID = strings.TrimSpace(in.ApplicationID)
	in.Location = strings.TrimSpace(in.Location)
	in.StaffID = strings.TrimSpace(in.StaffID)
}

func (in ScheduleVisitInput) Validate() error {
	in.Normalize()
	if in.PetID == "" || in.ApplicationID == "" {
		return ErrValidation
	}
	if !in.Type.IsValid() {
		return ErrInvalidVisitType
	}
	if in.Type == VisitFollowup {
		return ErrInvalidVisitType
	}
	if in.ScheduledAt.IsZero() {
		return ErrValidation
	}
	if utf8.RuneCountInString(in.Location) > 80 {
		return ErrInvalidCity
	}
	return nil
}

type CompleteVisitInput struct {
	LivingScore    int    `json:"living_score"`
	HealthScore    int    `json:"health_score"`
	BehaviorScore  int    `json:"behavior_score"`
	RiskFlag       bool   `json:"risk_flag"`
	Notes          string `json:"notes"`
	Issues         string `json:"issues"`
	Suggestion     string `json:"suggestion"`
	FollowUpNeeded bool   `json:"follow_up_needed"`
	Location       string `json:"location"`
}

func (in *CompleteVisitInput) Normalize() {
	in.Notes = strings.TrimSpace(in.Notes)
	in.Issues = strings.TrimSpace(in.Issues)
	in.Suggestion = strings.TrimSpace(in.Suggestion)
	in.Location = strings.TrimSpace(in.Location)
}

func validScore(n int) bool { return n >= 1 && n <= 5 }

func (in CompleteVisitInput) Validate() error {
	in.Normalize()
	if !validScore(in.LivingScore) || !validScore(in.HealthScore) || !validScore(in.BehaviorScore) {
		return ErrInvalidScore
	}
	if utf8.RuneCountInString(in.Notes) > 1000 {
		return ErrInvalidComment
	}
	if utf8.RuneCountInString(in.Issues) > 500 {
		return ErrInvalidComment
	}
	if utf8.RuneCountInString(in.Suggestion) > 500 {
		return ErrInvalidComment
	}
	return nil
}

type VisitCommentInput struct {
	Comment string `json:"comment"`
}

func (in VisitCommentInput) Validate() error {
	n := utf8.RuneCountInString(strings.TrimSpace(in.Comment))
	if n < 1 || n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type VisitFilter struct {
	PetID         string
	ApplicationID string
	AdopterID     string
	StaffID       string
	ShelterID     string
	Type          VisitType
	Status        VisitStatus
	DueBefore     *time.Time
	DueAfter      *time.Time
}
