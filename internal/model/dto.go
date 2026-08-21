package model

import "time"

type PetView struct {
	Pet
	ShelterName    string     `json:"shelter_name,omitempty"`
	StaffName      string     `json:"staff_name,omitempty"`
	Favorited      bool       `json:"favorited,omitempty"`
	MyApplication  *Application `json:"my_application,omitempty"`
	HealthCount    int        `json:"health_count,omitempty"`
	InquiryCount   int        `json:"inquiry_count,omitempty"`
}

type ApplicationView struct {
	Application
	PetName       string     `json:"pet_name,omitempty"`
	PetSpecies    Species    `json:"pet_species,omitempty"`
	PetStatus     PetStatus  `json:"pet_status,omitempty"`
	ApplicantName string     `json:"applicant_name,omitempty"`
	Applicant     PublicUser `json:"applicant,omitempty"`
}

type VisitView struct {
	Visit
	PetName       string `json:"pet_name,omitempty"`
	AdopterName   string `json:"adopter_name,omitempty"`
	StaffName     string `json:"staff_name,omitempty"`
}

type PageResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

type ClockNow func() time.Time
