package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleStaff   UserRole = "staff"
	RoleAdopter UserRole = "adopter"
)

func (r UserRole) IsValid() bool {
	return r == RoleAdmin || r == RoleStaff || r == RoleAdopter
}

type UserStatus string

const (
	UserActive UserStatus = "active"
	UserFrozen UserStatus = "frozen"
	UserBanned UserStatus = "banned"
)

func (s UserStatus) IsValid() bool {
	return s == UserActive || s == UserFrozen || s == UserBanned || s == ""
}

func (s UserStatus) Normalize() UserStatus {
	if s == "" {
		return UserActive
	}
	return s
}

type CreditLevel string

const (
	CreditRestricted CreditLevel = "restricted"
	CreditNew        CreditLevel = "new"
	CreditNormal     CreditLevel = "normal"
	CreditTrusted    CreditLevel = "trusted"
	CreditExcellent  CreditLevel = "excellent"
)

const (
	CreditMin           = 0
	CreditMax           = 100
	CreditInitial       = 60
	CreditRestrictedMax = 39
	CreditNewMax        = 59
	CreditNormalMax     = 74
	CreditTrustedMax    = 89
)

func CreditLevelOf(score int) CreditLevel {
	switch {
	case score <= CreditRestrictedMax:
		return CreditRestricted
	case score <= CreditNewMax:
		return CreditNew
	case score <= CreditNormalMax:
		return CreditNormal
	case score <= CreditTrustedMax:
		return CreditTrusted
	default:
		return CreditExcellent
	}
}

func ClampCredit(score int) int {
	if score < CreditMin {
		return CreditMin
	}
	if score > CreditMax {
		return CreditMax
	}
	return score
}

func (l CreditLevel) Rank() int {
	switch l {
	case CreditExcellent:
		return 4
	case CreditTrusted:
		return 3
	case CreditNormal:
		return 2
	case CreditNew:
		return 1
	default:
		return 0
	}
}

func (l CreditLevel) CanApply() bool {
	return l != CreditRestricted
}

type HousingType string

const (
	HousingApartment HousingType = "apartment"
	HousingHouse     HousingType = "house"
	HousingDetached  HousingType = "detached"
)

func (h HousingType) IsValid() bool {
	return h == HousingApartment || h == HousingHouse || h == HousingDetached || h == ""
}

type ExperienceLevel string

const (
	ExperienceNone      ExperienceLevel = "none"
	ExperienceFirstTime ExperienceLevel = "first_time"
	ExperienceSome      ExperienceLevel = "some"
	ExperienceExpert    ExperienceLevel = "expert"
)

func (e ExperienceLevel) IsValid() bool {
	return e == ExperienceNone || e == ExperienceFirstTime || e == ExperienceSome || e == ExperienceExpert || e == ""
}

// User 用户实体。口令仅存哈希与盐。
type User struct {
	ID           string          `json:"id"`
	Username     string          `json:"username"`
	PasswordHash string          `json:"password_hash"`
	PasswordSalt string          `json:"password_salt"`
	Iterations   int             `json:"iterations"`
	Role         UserRole        `json:"role"`
	Status       UserStatus      `json:"status"`
	DisplayName  string          `json:"display_name"`
	Phone        string          `json:"phone,omitempty"`
	Bio          string          `json:"bio,omitempty"`
	City         string          `json:"city,omitempty"`
	Housing      HousingType     `json:"housing,omitempty"`
	HasChildren  bool            `json:"has_children"`
	HasOtherPets bool            `json:"has_other_pets"`
	AgeYears     int             `json:"age_years,omitempty"`
	Experience   ExperienceLevel `json:"experience,omitempty"`
	ShelterID    string          `json:"shelter_id,omitempty"`
	CreditScore  int             `json:"credit_score"`
	AdoptCount   int             `json:"adopt_count"`
	ReturnCount  int             `json:"return_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	LastLoginAt  *time.Time      `json:"last_login_at,omitempty"`
}

func (u User) IsAdmin() bool   { return u.Role == RoleAdmin }
func (u User) IsStaff() bool   { return u.Role == RoleStaff }
func (u User) IsAdopter() bool { return u.Role == RoleAdopter }
func (u User) IsActive() bool  { return u.Status.Normalize() == UserActive }
func (u User) IsFrozen() bool  { return u.Status == UserFrozen }
func (u User) IsBanned() bool  { return u.Status == UserBanned }

func (u User) CreditLevel() CreditLevel { return CreditLevelOf(u.CreditScore) }

func (u User) CanWrite() error {
	switch u.Status.Normalize() {
	case UserBanned:
		return ErrAccountBanned
	case UserFrozen:
		return ErrAccountFrozen
	}
	return nil
}

func (u User) CanApply() error {
	if err := u.CanWrite(); err != nil {
		return err
	}
	if !u.CreditLevel().CanApply() {
		return ErrCreditRestricted
	}
	return nil
}

func (u User) ManagesShelter(shelterID string) bool {
	if u.IsAdmin() {
		return true
	}
	return u.IsStaff() && u.ShelterID != "" && u.ShelterID == shelterID
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Role:         u.Role,
		Status:       u.Status.Normalize(),
		City:         u.City,
		Housing:      u.Housing,
		HasChildren:  u.HasChildren,
		HasOtherPets: u.HasOtherPets,
		AgeYears:     u.AgeYears,
		Experience:   u.Experience,
		ShelterID:    u.ShelterID,
		CreditScore:  u.CreditScore,
		CreditLevel:  u.CreditLevel(),
		AdoptCount:   u.AdoptCount,
		ReturnCount:  u.ReturnCount,
		CreatedAt:    u.CreatedAt,
	}
}

type PublicUser struct {
	ID           string          `json:"id"`
	Username     string          `json:"username"`
	DisplayName  string          `json:"display_name"`
	Role         UserRole        `json:"role"`
	Status       UserStatus      `json:"status"`
	City         string          `json:"city,omitempty"`
	Housing      HousingType     `json:"housing,omitempty"`
	HasChildren  bool            `json:"has_children"`
	HasOtherPets bool            `json:"has_other_pets"`
	AgeYears     int             `json:"age_years,omitempty"`
	Experience   ExperienceLevel `json:"experience,omitempty"`
	ShelterID    string          `json:"shelter_id,omitempty"`
	CreditScore  int             `json:"credit_score"`
	CreditLevel  CreditLevel     `json:"credit_level"`
	AdoptCount   int             `json:"adopt_count"`
	ReturnCount  int             `json:"return_count"`
	CreatedAt    time.Time       `json:"created_at"`
}

type UserInput struct {
	Username     string          `json:"username"`
	Password     string          `json:"password"`
	Role         UserRole        `json:"role"`
	DisplayName  string          `json:"display_name"`
	Phone        string          `json:"phone"`
	Bio          string          `json:"bio"`
	City         string          `json:"city"`
	Housing      HousingType     `json:"housing"`
	HasChildren  bool            `json:"has_children"`
	HasOtherPets bool            `json:"has_other_pets"`
	AgeYears     int             `json:"age_years"`
	Experience   ExperienceLevel `json:"experience"`
	ShelterID    string          `json:"shelter_id"`
}

func (in *UserInput) Normalize() {
	in.Username = strings.TrimSpace(in.Username)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Bio = strings.TrimSpace(in.Bio)
	in.City = strings.TrimSpace(in.City)
	in.ShelterID = strings.TrimSpace(in.ShelterID)
	if in.Role == "" {
		in.Role = RoleAdopter
	}
	if in.Experience == "" {
		in.Experience = ExperienceFirstTime
	}
}

func (in UserInput) Validate() error {
	in.Normalize()
	if !IsValidUsername(in.Username) {
		return ErrInvalidUsername
	}
	if !IsValidPassword(in.Password) {
		return ErrInvalidPassword
	}
	if !in.Role.IsValid() {
		return ErrInvalidRole
	}
	if in.Role == RoleStaff && in.ShelterID == "" {
		return ErrStaffShelterRequired
	}
	if err := ValidateDisplayName(in.DisplayName); err != nil {
		return err
	}
	if utf8.RuneCountInString(in.Phone) > 20 {
		return ErrInvalidPhone
	}
	if utf8.RuneCountInString(in.Bio) > 200 {
		return ErrInvalidBio
	}
	if utf8.RuneCountInString(in.City) > 32 {
		return ErrInvalidCity
	}
	if !in.Housing.IsValid() {
		return ErrInvalidHousing
	}
	if !in.Experience.IsValid() {
		return ErrInvalidExperience
	}
	if in.AgeYears < 0 || in.AgeYears > 120 {
		return ErrInvalidAge
	}
	return nil
}

type ProfileInput struct {
	DisplayName  string          `json:"display_name"`
	Phone        string          `json:"phone"`
	Bio          string          `json:"bio"`
	City         string          `json:"city"`
	Housing      HousingType     `json:"housing"`
	HasChildren  bool            `json:"has_children"`
	HasOtherPets bool            `json:"has_other_pets"`
	AgeYears     int             `json:"age_years"`
	Experience   ExperienceLevel `json:"experience"`
}

func (in *ProfileInput) Normalize() {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Bio = strings.TrimSpace(in.Bio)
	in.City = strings.TrimSpace(in.City)
}

func (in ProfileInput) Validate() error {
	in.Normalize()
	if err := ValidateDisplayName(in.DisplayName); err != nil {
		return err
	}
	if utf8.RuneCountInString(in.Phone) > 20 {
		return ErrInvalidPhone
	}
	if utf8.RuneCountInString(in.Bio) > 200 {
		return ErrInvalidBio
	}
	if utf8.RuneCountInString(in.City) > 32 {
		return ErrInvalidCity
	}
	if !in.Housing.IsValid() {
		return ErrInvalidHousing
	}
	if !in.Experience.IsValid() {
		return ErrInvalidExperience
	}
	if in.AgeYears < 0 || in.AgeYears > 120 {
		return ErrInvalidAge
	}
	return nil
}

func ValidateDisplayName(s string) error {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	if n < 1 || n > 32 {
		return ErrInvalidDisplayName
	}
	return nil
}

func IsValidUsername(s string) bool {
	if len(s) < 3 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}

func IsValidPassword(s string) bool {
	return len(s) >= 6 && len(s) <= 64
}

type UserFilter struct {
	Role      UserRole
	Status    UserStatus
	ShelterID string
	Query     string
}

type PasswordChangeInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (in PasswordChangeInput) Validate() error {
	if !IsValidPassword(in.NewPassword) {
		return ErrInvalidPassword
	}
	if in.OldPassword == "" {
		return ErrInvalidPassword
	}
	return nil
}

type CreditAdjustInput struct {
	Delta  int    `json:"delta"`
	Reason string `json:"reason"`
}

func (in CreditAdjustInput) Validate() error {
	if in.Delta < -20 || in.Delta > 20 || in.Delta == 0 {
		return ErrInvalidCreditDelta
	}
	if strings.TrimSpace(in.Reason) == "" {
		return ErrValidation
	}
	return nil
}
