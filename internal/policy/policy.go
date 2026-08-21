package policy

import (
	"time"

	"go04-pet-adoption/internal/model"
)

const (
	DefaultMaxPendingApplications = 3
	HomeCheckMinScore             = 3
	VisitGraceDays                = 3
	EarlyReturnDays               = 30
	ConsecutiveMissedLimit        = 2
)

var FollowupScheduleDays = []int{7, 30, 90}

const (
	DeltaApplyDefault = -8
	DeltaMissedVisit  = -6
	DeltaGoodVisit    = 3
	DeltaEarlyReturn  = -12
	DeltaReturnOK     = -3
	DeltaAdoptionDone = 5
)

func FollowupDueTimes(handoverAt time.Time) []time.Time {
	out := make([]time.Time, 0, len(FollowupScheduleDays))
	for _, d := range FollowupScheduleDays {
		out = append(out, handoverAt.Add(time.Duration(d)*24*time.Hour))
	}
	return out
}

func IsEarlyReturn(handoverAt, returnedAt time.Time) bool {
	if handoverAt.IsZero() || returnedAt.IsZero() {
		return false
	}
	return returnedAt.Sub(handoverAt) < time.Duration(EarlyReturnDays)*24*time.Hour
}

func VisitOverdue(scheduledAt, now time.Time) bool {
	return now.After(scheduledAt.Add(time.Duration(VisitGraceDays) * 24 * time.Hour))
}

func HousingOK(pet model.Pet, housing model.HousingType) bool {
	if housing == model.HousingApartment && !pet.AllowApartment {
		return false
	}
	return true
}

func ApplyMatches(pet model.Pet, in model.ApplyInput, adopterAge int) error {
	return pet.MatchesAdopter(in.Housing, in.HasChildren, in.HasOtherPets, adopterAge)
}

func LargeDogNeedsSpace(pet model.Pet, housing model.HousingType, areaSqm int) bool {
	if pet.Species != model.SpeciesDog || pet.Size != model.SizeLarge {
		return true
	}
	if housing == model.HousingApartment && areaSqm > 0 && areaSqm < 70 {
		return false
	}
	return true
}

func ExperienceOK(pet model.Pet, exp model.ExperienceLevel) bool {
	if pet.SpecialNeeds && (exp == model.ExperienceNone || exp == model.ExperienceFirstTime) {
		return false
	}
	if pet.Size == model.SizeLarge && pet.Species == model.SpeciesDog && exp == model.ExperienceNone {
		return false
	}
	return true
}

func CreditDeltaForReturn(handoverAt, returnedAt time.Time, medical bool) (int, model.CreditReason) {
	if medical {
		return 0, model.CreditReturnOK
	}
	if IsEarlyReturn(handoverAt, returnedAt) {
		return DeltaEarlyReturn, model.CreditEarlyReturn
	}
	return DeltaReturnOK, model.CreditReturnOK
}

func AllFollowupsGood(visits []model.Visit) bool {
	var completed, expected int
	for _, v := range visits {
		if v.Type != model.VisitFollowup {
			continue
		}
		expected++
		if v.Status == model.VisitCompleted && !v.RiskFlag {
			completed++
		}
	}
	return expected >= len(FollowupScheduleDays) && completed == expected
}

func ConsecutiveMissed(visits []model.Visit) int {
	maxStreak, streak := 0, 0
	for _, v := range visits {
		if v.Type == model.VisitHomeCheck {
			continue
		}
		if v.Status == model.VisitMissed {
			streak++
			if streak > maxStreak {
				maxStreak = streak
			}
			continue
		}
		if v.Status == model.VisitCompleted {
			streak = 0
		}
	}
	return maxStreak
}

func SpeciesOptions() []model.EnumOption {
	return []model.EnumOption{
		{ID: string(model.SpeciesDog), Label: "狗"},
		{ID: string(model.SpeciesCat), Label: "猫"},
		{ID: string(model.SpeciesRabbit), Label: "兔"},
		{ID: string(model.SpeciesBird), Label: "鸟"},
		{ID: string(model.SpeciesOther), Label: "其他"},
	}
}

func SizeOptions() []model.EnumOption {
	return []model.EnumOption{
		{ID: string(model.SizeSmall), Label: "小型"},
		{ID: string(model.SizeMedium), Label: "中型"},
		{ID: string(model.SizeLarge), Label: "大型"},
	}
}

func HousingOptions() []model.EnumOption {
	return []model.EnumOption{
		{ID: string(model.HousingApartment), Label: "公寓"},
		{ID: string(model.HousingHouse), Label: "住宅"},
		{ID: string(model.HousingDetached), Label: "自建/别墅"},
	}
}

func ExperienceOptions() []model.EnumOption {
	return []model.EnumOption{
		{ID: string(model.ExperienceNone), Label: "无经验"},
		{ID: string(model.ExperienceFirstTime), Label: "初次养宠"},
		{ID: string(model.ExperienceSome), Label: "有一些经验"},
		{ID: string(model.ExperienceExpert), Label: "资深"},
	}
}

func PetStatusOptions() []model.EnumOption {
	return []model.EnumOption{
		{ID: string(model.PetDraft), Label: "草稿"},
		{ID: string(model.PetPublished), Label: "待领养"},
		{ID: string(model.PetReserved), Label: "已预留"},
		{ID: string(model.PetAdopted), Label: "已领养"},
		{ID: string(model.PetUnavailable), Label: "暂不可领养"},
		{ID: string(model.PetDeceased), Label: "已死亡"},
	}
}

func Catalog() map[string][]model.EnumOption {
	return map[string][]model.EnumOption{
		"species":    SpeciesOptions(),
		"size":       SizeOptions(),
		"housing":    HousingOptions(),
		"experience": ExperienceOptions(),
		"pet_status": PetStatusOptions(),
		"sex": {
			{ID: string(model.SexMale), Label: "公"},
			{ID: string(model.SexFemale), Label: "母"},
			{ID: string(model.SexUnknown), Label: "未知"},
		},
		"health_kind": {
			{ID: string(model.HealthVaccine), Label: "疫苗"},
			{ID: string(model.HealthDeworm), Label: "驱虫"},
			{ID: string(model.HealthSterilize), Label: "绝育"},
			{ID: string(model.HealthCheckup), Label: "体检"},
			{ID: string(model.HealthTreatment), Label: "治疗"},
			{ID: string(model.HealthOther), Label: "其他"},
		},
		"visit_type": {
			{ID: string(model.VisitHomeCheck), Label: "家访"},
			{ID: string(model.VisitFollowup), Label: "随访"},
			{ID: string(model.VisitExtra), Label: "加访"},
		},
	}
}
