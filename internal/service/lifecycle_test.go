package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/model"
	"go04-pet-adoption/internal/service"
	"go04-pet-adoption/internal/store"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// applicationLimitBarrierStore 通过在 CreateApplication 真正落库前同步两个并发请求，
// 强制制造曾经导致竞态的"check-then-act"窗口：两个请求必须同时到达临界区，
// 才能验证数量上限校验是在同一把写锁内原子完成的。
type applicationLimitBarrierStore struct {
	store.Store
	entered chan struct{}
	release chan struct{}
}

type withdrawalCreditBarrierStore struct {
	store.Store
	entered chan struct{}
	release chan struct{}
	applied chan struct{}
	proceed chan struct{}
}

func (s *withdrawalCreditBarrierStore) ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string, now time.Time) (model.User, model.CreditLog, error) {
	s.entered <- struct{}{}
	<-s.release
	u, log, err := s.Store.ApplyCredit(ctx, userID, delta, reason, relatedID, note, now)
	s.applied <- struct{}{}
	<-s.proceed
	return u, log, err
}

func (s *applicationLimitBarrierStore) CreateApplication(ctx context.Context, a model.Application, maxActive int) (model.Application, error) {
	s.entered <- struct{}{}
	<-s.release
	return s.Store.CreateApplication(ctx, a, maxActive)
}

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

func TestConcurrentApplicationsRespectPendingLimit(t *testing.T) {
	ctx := context.Background()
	mem := store.NewMemoryStore(time.Now, nil)
	barrier := &applicationLimitBarrierStore{
		Store:   mem,
		entered: make(chan struct{}, 2),
		release: make(chan struct{}),
	}
	svc := service.NewServices(barrier, auth.NewPasswordHasher(), auth.NewSessionManager(time.Hour), nil, 1)

	adopter, err := mem.CreateUser(ctx, model.User{
		Username: "concurrent-adopter", DisplayName: "并发申请人", Role: model.RoleAdopter,
		Status: model.UserActive, CreditScore: 60, AgeYears: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	pet1, err := mem.CreatePet(ctx, model.Pet{
		Name: "小白", Species: model.SpeciesCat, Size: model.SizeSmall, Status: model.PetPublished,
		AllowApartment: true, AllowChildren: true, AllowOtherPets: true, MinAdopterAge: 18,
	})
	if err != nil {
		t.Fatal(err)
	}
	pet2, err := mem.CreatePet(ctx, model.Pet{
		Name: "小黑", Species: model.SpeciesCat, Size: model.SizeSmall, Status: model.PetPublished,
		AllowApartment: true, AllowChildren: true, AllowOtherPets: true, MinAdopterAge: 18,
	})
	if err != nil {
		t.Fatal(err)
	}

	in := model.ApplyInput{
		Housing: model.HousingApartment, Experience: model.ExperienceSome,
		Phone: "13800006666", Intro: "有稳定住所并能长期照顾宠物。",
	}
	petIDs := []string{pet1.ID, pet2.ID}
	errs := make(chan error, len(petIDs))
	var wg sync.WaitGroup
	for _, petID := range petIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, applyErr := svc.App.Apply(ctx, adopter, petID, in)
			errs <- applyErr
		}()
	}

	<-barrier.entered
	<-barrier.entered
	close(barrier.release)
	wg.Wait()
	close(errs)

	successes := 0
	limitErrors := 0
	for applyErr := range errs {
		switch {
		case applyErr == nil:
			successes++
		case errors.Is(applyErr, model.ErrTooManyApplications):
			limitErrors++
		default:
			t.Fatalf("unexpected apply error: %v", applyErr)
		}
	}
	if successes != 1 || limitErrors != 1 {
		t.Fatalf("pending limit=1: want one success and one limit error, got successes=%d limit_errors=%d", successes, limitErrors)
	}
	active, err := mem.CountActiveApplicationsByApplicant(ctx, adopter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("pending limit=1: active applications=%d", active)
	}
}

func TestConcurrentApprovedWithdrawalsApplyOnePenalty(t *testing.T) {
	ctx := context.Background()
	mem := store.NewMemoryStore(time.Now, nil)
	barrier := &withdrawalCreditBarrierStore{
		Store:   mem,
		entered: make(chan struct{}, 2),
		release: make(chan struct{}),
		applied: make(chan struct{}, 2),
		proceed: make(chan struct{}),
	}
	svc := service.NewServices(barrier, auth.NewPasswordHasher(), auth.NewSessionManager(time.Hour), nil, 3)

	adopter, err := mem.CreateUser(ctx, model.User{
		ID: "user-withdraw", Username: "withdraw-adopter", DisplayName: "撤回申请人",
		Role: model.RoleAdopter, Status: model.UserActive, CreditScore: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	const appID = "app-approved"
	pet, err := mem.CreatePet(ctx, model.Pet{
		ID: "pet-reserved", Name: "团团", Species: model.SpeciesCat, Size: model.SizeSmall,
		Status: model.PetReserved, ReservedAppID: appID,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = mem.CreateApplication(ctx, model.Application{
		ID: appID, PetID: pet.ID, ApplicantID: adopter.ID, Status: model.AppApproved,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, withdrawErr := svc.App.Withdraw(ctx, adopter, appID)
			errs <- withdrawErr
		}()
	}

	<-barrier.entered
	<-barrier.entered
	close(barrier.release)
	<-barrier.applied
	<-barrier.applied
	close(barrier.proceed)
	wg.Wait()
	close(errs)

	successes := 0
	conflicts := 0
	for withdrawErr := range errs {
		switch {
		case withdrawErr == nil:
			successes++
		case errors.Is(withdrawErr, model.ErrInvalidAppStatus):
			conflicts++
		default:
			t.Fatalf("unexpected withdraw error: %v", withdrawErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("want one successful withdrawal and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}
	fresh, err := mem.GetUserByID(ctx, adopter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.CreditScore != 52 {
		t.Fatalf("one approved withdrawal should apply one -8 penalty: credit=%d", fresh.CreditScore)
	}
	logs, err := mem.ListCreditLogs(ctx, adopter.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("one approved withdrawal should create one credit log: logs=%d", len(logs))
	}
}
