package model

import "testing"

func TestCreditLevelOf(t *testing.T) {
	if CreditLevelOf(20) != CreditRestricted {
		t.Fatal("restricted")
	}
	if CreditLevelOf(50) != CreditNew {
		t.Fatal("new")
	}
	if CreditLevelOf(70) != CreditNormal {
		t.Fatal("normal")
	}
	if CreditLevelOf(80) != CreditTrusted {
		t.Fatal("trusted")
	}
	if CreditLevelOf(95) != CreditExcellent {
		t.Fatal("excellent")
	}
}

func TestPetMatchesAdopter(t *testing.T) {
	p := Pet{AllowApartment: false, AllowChildren: true, AllowOtherPets: false, MinAdopterAge: 21}
	if err := p.MatchesAdopter(HousingApartment, false, false, 30); err == nil {
		t.Fatal("apartment should fail")
	}
	if err := p.MatchesAdopter(HousingHouse, false, false, 18); err == nil {
		t.Fatal("age should fail")
	}
	if err := p.MatchesAdopter(HousingHouse, false, false, 25); err != nil {
		t.Fatal(err)
	}
}

func TestApplicationStatus(t *testing.T) {
	if !AppPending.CanApprove() || AppCompleted.CanApprove() {
		t.Fatal("approve flags")
	}
	if !AppApproved.CountsTowardLimit() || AppWaitlisted.CountsTowardLimit() {
		t.Fatal("limit flags")
	}
}

func TestVisitPassedHomeCheck(t *testing.T) {
	v := Visit{Type: VisitHomeCheck, Status: VisitCompleted, LivingScore: 4, HealthScore: 4, BehaviorScore: 3}
	if !v.PassedHomeCheck(3) {
		t.Fatal("should pass")
	}
	v.RiskFlag = true
	if v.PassedHomeCheck(3) {
		t.Fatal("risk should fail")
	}
}

func TestUserCanApply(t *testing.T) {
	u := User{Status: UserActive, CreditScore: 60, Role: RoleAdopter}
	if err := u.CanApply(); err != nil {
		t.Fatal(err)
	}
	u.CreditScore = 10
	if err := u.CanApply(); err == nil {
		t.Fatal("restricted should fail")
	}
}
