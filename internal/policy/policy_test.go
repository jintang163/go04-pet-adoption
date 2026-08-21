package policy

import (
	"testing"
	"time"

	"go04-pet-adoption/internal/model"
)

func TestFollowupDueTimes(t *testing.T) {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dues := FollowupDueTimes(base)
	if len(dues) != 3 {
		t.Fatalf("len=%d", len(dues))
	}
	if dues[0] != base.Add(7*24*time.Hour) {
		t.Fatal("day 7")
	}
}

func TestCreditDeltaForReturn(t *testing.T) {
	h := time.Now()
	d, r := CreditDeltaForReturn(h, h.Add(2*24*time.Hour), false)
	if d != DeltaEarlyReturn || r != model.CreditEarlyReturn {
		t.Fatalf("%d %s", d, r)
	}
	d, r = CreditDeltaForReturn(h, h.Add(2*24*time.Hour), true)
	if d != 0 {
		t.Fatalf("medical %d", d)
	}
}

func TestExperienceOK(t *testing.T) {
	p := model.Pet{SpecialNeeds: true, Species: model.SpeciesCat, Size: model.SizeSmall}
	if ExperienceOK(p, model.ExperienceFirstTime) {
		t.Fatal("special needs need experience")
	}
}

func TestConsecutiveMissed(t *testing.T) {
	visits := []model.Visit{
		{Type: model.VisitFollowup, Status: model.VisitMissed},
		{Type: model.VisitFollowup, Status: model.VisitMissed},
		{Type: model.VisitFollowup, Status: model.VisitCompleted},
	}
	if n := ConsecutiveMissed(visits); n != 2 {
		t.Fatalf("got %d", n)
	}
}
