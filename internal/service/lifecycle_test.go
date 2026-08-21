package service_test

import (
	"context"
	"testing"
	"time"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/service"
	"go04-pet-adoption/internal/store"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func setup(t *testing.T) (context.Context, *service.Services, *store.MemoryStore, *auth.PasswordHasher) {
	t.Helper()
	ctx := context.Background()
	mem := store.NewMemoryStore(time.Now, nil)
	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(time.Hour)
	svc := service.NewServices(mem, hasher, sessions, nil, 3)
	return ctx, svc, mem, hasher
}

func mustAdopter(t *testing.T, ctx context.Context, svc *service.Services, name string, housing model.HousingType) model.User {
	t.Helper()
	out, err := svc.Auth.Register(ctx, model.UserInput{
		Username: name, Password: name + "1234", DisplayName: name,
		Role: model.RoleAdopter, Housing: housing, AgeYears: 28, Experience: model.ExperienceSome,
		Phone: "13800000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := svc.User.GetByID(ctx, out.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func mustStaff(t *testing.T, ctx context.Context, mem *store.MemoryStore, hasher *auth.PasswordHasher, name, shelterID string) model.User {
	t.Helper()
	salt, hash, it, err := hasher.Hash(name + "1234")
	if err != nil {
		t.Fatal(err)
	}
	u, err := mem.CreateUser(ctx, model.User{
		Username: name, PasswordHash: hash, PasswordSalt: salt, Iterations: it,
		Role: model.RoleStaff, Status: model.UserActive, DisplayName: name,
		ShelterID: shelterID, CreditScore: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestApplyApproveHandoverVisit(t *testing.T) {
	ctx, svc, mem, hasher := setup(t)
	sh, err := mem.CreateShelter(ctx, model.Shelter{Name: "站", Status: model.ShelterActive})
	if err != nil {
		t.Fatal(err)
	}
	staff := mustStaff(t, ctx, mem, hasher, "staffy", sh.ID)
	alice := mustAdopter(t, ctx, svc, "alice", model.HousingHouse)
	bob := mustAdopter(t, ctx, svc, "bobby", model.HousingHouse)

	need := true
	pet, err := svc.Pet.Create(ctx, staff, model.PetInput{
		Name: "橘子", Species: model.SpeciesCat, Size: model.SizeSmall, Sex: model.SexFemale,
		AgeMonths: 12, Story: "亲人橘猫，已疫苗绝育，适合家庭领养。",
		AllowApartment: true, AllowChildren: true, AllowOtherPets: true,
		MinAdopterAge: 18, NeedHomeCheck: &need, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pet.Status != model.PetPublished {
		t.Fatalf("status %s", pet.Status)
	}

	if _, err := svc.App.Apply(ctx, staff, pet.ID, model.ApplyInput{
		Housing: model.HousingHouse, Experience: model.ExperienceSome, Phone: "1", Intro: "员工不可申请自己的宠物啊啊",
	}); err == nil {
		t.Fatal("staff should not apply own pet")
	}

	app, err := svc.App.Apply(ctx, alice, pet.ID, model.ApplyInput{
		Housing: model.HousingHouse, HoursAlone: 3, Experience: model.ExperienceSome,
		Phone: "13800001111", Intro: "有稳定住所，可按时喂养和陪玩。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != model.AppPending {
		t.Fatalf("want pending got %s", app.Status)
	}

	app2, err := svc.App.Apply(ctx, bob, pet.ID, model.ApplyInput{
		Housing: model.HousingHouse, HoursAlone: 2, Experience: model.ExperienceSome,
		Phone: "13800002222", Intro: "同样希望领养这只猫，家里有院子。",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.App.Approve(ctx, staff, app.ID, model.ReviewInput{Note: "ok"}, false); err == nil {
		t.Fatal("home check required")
	}

	hc, err := svc.Visit.Schedule(ctx, staff, model.ScheduleVisitInput{
		PetID: pet.ID, ApplicationID: app.ID, Type: model.VisitHomeCheck, ScheduledAt: time.Now().Add(time.Hour),
		Location: "申请人住所",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Visit.Complete(ctx, staff, hc.ID, model.CompleteVisitInput{
		LivingScore: 4, HealthScore: 4, BehaviorScore: 4, Notes: "环境整洁",
	}); err != nil {
		t.Fatal(err)
	}

	approved, err := svc.App.Approve(ctx, staff, app.ID, model.ReviewInput{Note: "家访通过"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != model.AppApproved {
		t.Fatalf("got %s", approved.Status)
	}
	fresh, _ := mem.GetPet(ctx, pet.ID)
	if fresh.Status != model.PetReserved {
		t.Fatalf("pet %s", fresh.Status)
	}
	bobApp, _ := mem.GetApplication(ctx, app2.ID)
	if bobApp.Status != model.AppWaitlisted {
		t.Fatalf("bob %s", bobApp.Status)
	}

	handed, visits, err := svc.App.Handover(ctx, staff, approved.ID, model.HandoverInput{Note: "已领取"})
	if err != nil {
		t.Fatal(err)
	}
	if handed.Status != model.AppCompleted || len(visits) != 3 {
		t.Fatalf("handover %+v n=%d", handed, len(visits))
	}
	adopted, _ := mem.GetPet(ctx, pet.ID)
	if adopted.Status != model.PetAdopted {
		t.Fatalf("adopted %s", adopted.Status)
	}

	if _, err := svc.Visit.Complete(ctx, staff, visits[0].ID, model.CompleteVisitInput{
		LivingScore: 5, HealthScore: 5, BehaviorScore: 5, Notes: "适应良好",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRequirementAndCreditGate(t *testing.T) {
	ctx, svc, mem, hasher := setup(t)
	sh, _ := mem.CreateShelter(ctx, model.Shelter{Name: "站", Status: model.ShelterActive})
	staff := mustStaff(t, ctx, mem, hasher, "st2", sh.ID)
	need := false
	pet, err := svc.Pet.Create(ctx, staff, model.PetInput{
		Name: "阿黄", Species: model.SpeciesDog, Size: model.SizeLarge, Sex: model.SexMale,
		AgeMonths: 24, Story: "大型犬需要院子和有经验的领养人长期陪伴。",
		AllowApartment: false, AllowChildren: true, AllowOtherPets: true,
		MinAdopterAge: 21, NeedHomeCheck: &need, Publish: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	alice := mustAdopter(t, ctx, svc, "aptuser", model.HousingApartment)
	if _, err := svc.App.Apply(ctx, alice, pet.ID, model.ApplyInput{
		Housing: model.HousingApartment, Experience: model.ExperienceNone, Phone: "1", Intro: "我想领养大狗但是住公寓。",
	}); err == nil {
		t.Fatal("requirement should fail")
	}

	alice.CreditScore = 10
	alice, _ = mem.UpdateUser(ctx, alice)
	houseUser := mustAdopter(t, ctx, svc, "lowcred", model.HousingHouse)
	houseUser.CreditScore = 10
	houseUser, _ = mem.UpdateUser(ctx, houseUser)
	if _, err := svc.App.Apply(ctx, houseUser, pet.ID, model.ApplyInput{
		Housing: model.HousingHouse, Experience: model.ExperienceSome, Phone: "13800003333", Intro: "信用不足时不应允许提交申请。",
	}); err == nil {
		t.Fatal("restricted credit should fail")
	}
}

func TestRejectUnlocksWaitlist(t *testing.T) {
	ctx, svc, mem, hasher := setup(t)
	sh, _ := mem.CreateShelter(ctx, model.Shelter{Name: "站", Status: model.ShelterActive})
	staff := mustStaff(t, ctx, mem, hasher, "st3", sh.ID)
	a1 := mustAdopter(t, ctx, svc, "aaone", model.HousingHouse)
	a2 := mustAdopter(t, ctx, svc, "aatwo", model.HousingHouse)
	need := false
	pet, _ := svc.Pet.Create(ctx, staff, model.PetInput{
		Name: "棉花", Species: model.SpeciesCat, Size: model.SizeSmall, Sex: model.SexFemale,
		AgeMonths: 8, Story: "幼猫待领养，性格安静适合家庭。",
		AllowApartment: true, AllowChildren: true, AllowOtherPets: true,
		MinAdopterAge: 18, NeedHomeCheck: &need, Publish: true,
	})
	app1, _ := svc.App.Apply(ctx, a1, pet.ID, model.ApplyInput{
		Housing: model.HousingHouse, Experience: model.ExperienceSome, Phone: "13800004444", Intro: "第一申请人介绍足够长。",
	})
	app2, _ := svc.App.Apply(ctx, a2, pet.ID, model.ApplyInput{
		Housing: model.HousingHouse, Experience: model.ExperienceSome, Phone: "13800005555", Intro: "第二申请人介绍足够长。",
	})
	if _, err := svc.App.Approve(ctx, staff, app1.ID, model.ReviewInput{}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.App.Reject(ctx, staff, app1.ID, model.RejectInput{Reason: "临时无法交接需要释放名额"}); err != nil {
		t.Fatal(err)
	}
	p, _ := mem.GetPet(ctx, pet.ID)
	if p.Status != model.PetPublished {
		t.Fatalf("pet %s", p.Status)
	}
	w, _ := mem.GetApplication(ctx, app2.ID)
	if w.Status != model.AppPending {
		t.Fatalf("waitlist promote got %s", w.Status)
	}
}
