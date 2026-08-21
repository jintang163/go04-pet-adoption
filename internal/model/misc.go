package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Favorite struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PetID     string    `json:"pet_id"`
	CreatedAt time.Time `json:"created_at"`
}

type FavoriteKey struct {
	UserID string
	PetID  string
}

type Inquiry struct {
	ID        string    `json:"id"`
	PetID     string    `json:"pet_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Reply     string    `json:"reply,omitempty"`
	ReplierID string    `json:"replier_id,omitempty"`
	RepliedAt *time.Time `json:"replied_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type InquiryInput struct {
	Content string `json:"content"`
}

func (in InquiryInput) Validate() error {
	n := utf8.RuneCountInString(strings.TrimSpace(in.Content))
	if n < 1 || n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type NotificationKind string

const (
	NotifyAppApproved   NotificationKind = "app_approved"
	NotifyAppRejected   NotificationKind = "app_rejected"
	NotifyAppWaitlisted NotificationKind = "app_waitlisted"
	NotifyAppPromoted   NotificationKind = "app_promoted"
	NotifyVisitDue      NotificationKind = "visit_due"
	NotifyVisitDone     NotificationKind = "visit_done"
	NotifyVisitMissed   NotificationKind = "visit_missed"
	NotifyHandover      NotificationKind = "handover"
	NotifyReturn        NotificationKind = "return"
	NotifyRisk          NotificationKind = "risk"
	NotifyCredit        NotificationKind = "credit"
	NotifyGeneric       NotificationKind = "generic"
)

type Notification struct {
	ID        string           `json:"id"`
	UserID    string           `json:"user_id"`
	Kind      NotificationKind `json:"kind"`
	Title     string           `json:"title"`
	Body      string           `json:"body,omitempty"`
	RelatedID string           `json:"related_id,omitempty"`
	ReadAt    *time.Time       `json:"read_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

func (n Notification) Unread() bool { return n.ReadAt == nil }

type AuditAction string

const (
	AuditPetPublish   AuditAction = "pet_publish"
	AuditPetUnpublish AuditAction = "pet_unpublish"
	AuditAppApprove   AuditAction = "app_approve"
	AuditAppReject    AuditAction = "app_reject"
	AuditHandover     AuditAction = "handover"
	AuditReturn       AuditAction = "return"
	AuditVisit        AuditAction = "visit"
	AuditUserFreeze   AuditAction = "user_freeze"
	AuditCredit       AuditAction = "credit"
	AuditForce        AuditAction = "force"
)

type AuditLog struct {
	ID        string      `json:"id"`
	ActorID   string      `json:"actor_id"`
	Action    AuditAction `json:"action"`
	TargetID  string      `json:"target_id"`
	Detail    string      `json:"detail,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

type CreditReason string

const (
	CreditApplyDefault   CreditReason = "apply_default"
	CreditMissedVisit    CreditReason = "missed_visit"
	CreditGoodVisit      CreditReason = "good_visit"
	CreditEarlyReturn    CreditReason = "early_return"
	CreditReturnOK       CreditReason = "return_ok"
	CreditAdminAdjust    CreditReason = "admin_adjust"
	CreditAdoptionDone   CreditReason = "adoption_done"
)

type CreditLog struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	Delta     int          `json:"delta"`
	Score     int          `json:"score"`
	Reason    CreditReason `json:"reason"`
	RelatedID string       `json:"related_id,omitempty"`
	Note      string       `json:"note,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type EnumOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (in *LoginInput) Normalize() {
	in.Username = strings.TrimSpace(in.Username)
}

type AuthResult struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type MeResponse struct {
	User            PublicUser `json:"user"`
	UnreadNotify    int        `json:"unread_notify"`
	ActiveApps      int        `json:"active_apps"`
	UpcomingVisits  int        `json:"upcoming_visits"`
}

type StatsSnapshot struct {
	UsersTotal       int            `json:"users_total"`
	UsersActive      int            `json:"users_active"`
	UsersFrozen      int            `json:"users_frozen"`
	PetsByStatus     map[string]int `json:"pets_by_status"`
	PetsTotal        int            `json:"pets_total"`
	AppsTotal        int            `json:"apps_total"`
	AppsApproved     int            `json:"apps_approved"`
	AppsCompleted    int            `json:"apps_completed"`
	HandoverThisMonth int           `json:"handover_this_month"`
	VisitsScheduled  int            `json:"visits_scheduled"`
	VisitsCompleted  int            `json:"visits_completed"`
	VisitsMissed     int            `json:"visits_missed"`
	ReturnCount      int            `json:"return_count"`
	ConversionRate   float64        `json:"conversion_rate"`
	VisitCompleteRate float64       `json:"visit_complete_rate"`
}

type StaffBoard struct {
	PendingApps     int `json:"pending_apps"`
	UnderReviewApps int `json:"under_review_apps"`
	DraftPets       int `json:"draft_pets"`
	PublishedPets   int `json:"published_pets"`
	DueVisitsToday  int `json:"due_visits_today"`
}
