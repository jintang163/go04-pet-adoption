package model

import "errors"

// 领域错误。HTTP 层根据错误类型映射状态码。
var (
	ErrNotFound           = errors.New("resource not found")
	ErrAlreadyExists      = errors.New("resource already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrValidation         = errors.New("validation error")
	ErrConflict           = errors.New("conflict")
	ErrInternal           = errors.New("internal error")
	ErrAccountFrozen      = errors.New("account is frozen")
	ErrAccountBanned      = errors.New("account is banned")
	ErrCreditRestricted   = errors.New("credit level is restricted for this action")

	ErrPetNotPublished       = errors.New("pet is not published for applications")
	ErrPetNotReservable      = errors.New("pet cannot be reserved in current status")
	ErrPetAlreadyReserved    = errors.New("pet is already reserved")
	ErrPetNotAdopted         = errors.New("pet is not in adopted status")
	ErrCannotApplyOwnPet     = errors.New("cannot apply for a pet you manage")
	ErrAlreadyApplied        = errors.New("already have an active application for this pet")
	ErrTooManyApplications   = errors.New("too many active applications")
	ErrRequirementNotMet     = errors.New("adopter profile does not meet pet requirements")
	ErrHomeCheckRequired     = errors.New("home check must be completed before approval")
	ErrHomeCheckFailed       = errors.New("home check scores are below the approval threshold")
	ErrInvalidAppStatus      = errors.New("application status does not allow this action")
	ErrNotApplicant          = errors.New("only the applicant can perform this action")
	ErrNotPetStaff           = errors.New("only shelter staff or admin can perform this action")
	ErrInvalidVisitStatus    = errors.New("visit status does not allow this action")
	ErrVisitNotDue           = errors.New("visit is not yet due")
	ErrInvalidVisitType      = errors.New("invalid visit type")
	ErrReturnNotAllowed      = errors.New("return is not allowed in current status")
	ErrShelterInactive       = errors.New("shelter is inactive")
	ErrStaffShelterRequired  = errors.New("staff account must bind a shelter")
	ErrDuplicateFavorite     = errors.New("pet already favorited")
	ErrCannotDeleteShelter   = errors.New("cannot delete shelter with active pets")
	ErrHandoverNotApproved   = errors.New("only an approved application can be handed over")
	ErrInvalidHandleAction   = errors.New("invalid handle action")

	ErrInvalidUsername    = errors.New("invalid username: 3-32 letters, digits or underscore")
	ErrInvalidPassword    = errors.New("invalid password: 6-64 characters")
	ErrInvalidRole        = errors.New("invalid role: must be admin, staff or adopter")
	ErrInvalidDisplayName = errors.New("invalid display name: 1-32 characters")
	ErrInvalidPhone       = errors.New("invalid phone: 0-20 characters")
	ErrInvalidBio         = errors.New("invalid bio: max 200 characters")
	ErrInvalidUserStatus  = errors.New("invalid user status")
	ErrInvalidSpecies     = errors.New("invalid species")
	ErrInvalidSize        = errors.New("invalid size")
	ErrInvalidSex         = errors.New("invalid sex")
	ErrInvalidPetStatus   = errors.New("invalid pet status")
	ErrInvalidHousing     = errors.New("invalid housing type")
	ErrInvalidExperience  = errors.New("invalid experience level")
	ErrInvalidScore       = errors.New("invalid score: 1-5")
	ErrInvalidName        = errors.New("invalid name: 1-40 characters")
	ErrInvalidStory       = errors.New("invalid story/description: 1-4000 characters")
	ErrInvalidAge         = errors.New("invalid age months")
	ErrInvalidCreditDelta = errors.New("invalid credit delta: -20 to +20")
	ErrInvalidCity        = errors.New("invalid city or address")
	ErrInvalidComment     = errors.New("invalid comment: 1-500 characters")
	ErrInvalidIntro       = errors.New("invalid intro: 1-500 characters")
	ErrInvalidHoursAlone  = errors.New("invalid hours alone: 0-24")
	ErrInvalidMinAge      = errors.New("invalid minimum adopter age: 18-80")
)

func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

func IsAlreadyExists(err error) bool { return errors.Is(err, ErrAlreadyExists) }

func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrAccountFrozen) ||
		errors.Is(err, ErrAccountBanned) ||
		errors.Is(err, ErrCreditRestricted) ||
		errors.Is(err, ErrNotApplicant) ||
		errors.Is(err, ErrNotPetStaff) ||
		errors.Is(err, ErrStaffShelterRequired)
}

func IsInvalidCredentials(err error) bool { return errors.Is(err, ErrInvalidCredentials) }

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrPetNotPublished) ||
		errors.Is(err, ErrPetNotReservable) ||
		errors.Is(err, ErrPetAlreadyReserved) ||
		errors.Is(err, ErrPetNotAdopted) ||
		errors.Is(err, ErrCannotApplyOwnPet) ||
		errors.Is(err, ErrAlreadyApplied) ||
		errors.Is(err, ErrTooManyApplications) ||
		errors.Is(err, ErrHomeCheckRequired) ||
		errors.Is(err, ErrHomeCheckFailed) ||
		errors.Is(err, ErrInvalidAppStatus) ||
		errors.Is(err, ErrInvalidVisitStatus) ||
		errors.Is(err, ErrReturnNotAllowed) ||
		errors.Is(err, ErrShelterInactive) ||
		errors.Is(err, ErrDuplicateFavorite) ||
		errors.Is(err, ErrCannotDeleteShelter) ||
		errors.Is(err, ErrHandoverNotApproved)
}

func IsValidation(err error) bool {
	switch {
	case errors.Is(err, ErrValidation),
		errors.Is(err, ErrRequirementNotMet),
		errors.Is(err, ErrInvalidUsername),
		errors.Is(err, ErrInvalidPassword),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidDisplayName),
		errors.Is(err, ErrInvalidPhone),
		errors.Is(err, ErrInvalidBio),
		errors.Is(err, ErrInvalidUserStatus),
		errors.Is(err, ErrInvalidSpecies),
		errors.Is(err, ErrInvalidSize),
		errors.Is(err, ErrInvalidSex),
		errors.Is(err, ErrInvalidPetStatus),
		errors.Is(err, ErrInvalidHousing),
		errors.Is(err, ErrInvalidExperience),
		errors.Is(err, ErrInvalidScore),
		errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidStory),
		errors.Is(err, ErrInvalidAge),
		errors.Is(err, ErrInvalidCreditDelta),
		errors.Is(err, ErrInvalidCity),
		errors.Is(err, ErrInvalidComment),
		errors.Is(err, ErrInvalidIntro),
		errors.Is(err, ErrInvalidHoursAlone),
		errors.Is(err, ErrInvalidMinAge),
		errors.Is(err, ErrInvalidVisitType),
		errors.Is(err, ErrInvalidHandleAction),
		errors.Is(err, ErrVisitNotDue):
		return true
	}
	return false
}
