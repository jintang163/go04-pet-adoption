package store

import (
	"context"
	"time"

	"go04-pet-adoption/internal/auth"
	"go04-pet-adoption/internal/model"
)

func SeedAdmin(ctx context.Context, st Store, hasher *auth.PasswordHasher, username, password string) error {
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin123"
	}
	if _, err := st.GetUserByUsername(ctx, username); err == nil {
		return nil
	}
	salt, hash, it, err := hasher.Hash(password)
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = st.CreateUser(ctx, model.User{
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   it,
		Role:         model.RoleAdmin,
		Status:       model.UserActive,
		DisplayName:  "系统管理员",
		CreditScore:  model.CreditInitial,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	return err
}

func SeedDemo(ctx context.Context, st Store, hasher *auth.PasswordHasher) error {
	users, err := st.ListUsers(ctx, model.UserFilter{Role: model.RoleAdopter})
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return nil
	}
	shelters, err := st.ListShelters(ctx, false)
	if err != nil {
		return err
	}
	var shelter model.Shelter
	if len(shelters) == 0 {
		shelter, err = st.CreateShelter(ctx, model.Shelter{
			Name:    "阳光宠物救助站",
			City:    "杭州",
			Address: "西湖区救助路 18 号",
			Phone:   "0571-00000000",
			Status:  model.ShelterActive,
		})
		if err != nil {
			return err
		}
	} else {
		shelter = shelters[0]
	}

	mustUser := func(name, pass, display string, role model.UserRole, shelterID string, housing model.HousingType) (model.User, error) {
		if u, err := st.GetUserByUsername(ctx, name); err == nil {
			return u, nil
		}
		salt, hash, it, err := hasher.Hash(pass)
		if err != nil {
			return model.User{}, err
		}
		return st.CreateUser(ctx, model.User{
			Username:     name,
			PasswordHash: hash,
			PasswordSalt: salt,
			Iterations:   it,
			Role:         role,
			Status:       model.UserActive,
			DisplayName:  display,
			City:         "杭州",
			Housing:      housing,
			AgeYears:     28,
			Experience:   model.ExperienceSome,
			ShelterID:    shelterID,
			CreditScore:  model.CreditInitial,
			Phone:        "13800000000",
		})
	}

	staff, err := mustUser("staff", "staff123", "站长小林", model.RoleStaff, shelter.ID, model.HousingHouse)
	if err != nil {
		return err
	}
	if _, err := mustUser("alice", "alice123", "爱心领养人小艾", model.RoleAdopter, "", model.HousingApartment); err != nil {
		return err
	}
	if _, err := mustUser("bob", "bob123", "领养人阿波", model.RoleAdopter, "", model.HousingHouse); err != nil {
		return err
	}

	needCheck := true
	now := time.Now()
	pub := now
	pets := []model.Pet{
		{
			ShelterID:      shelter.ID,
			StaffID:        staff.ID,
			Name:           "橘子",
			Species:        model.SpeciesCat,
			Breed:          "中华田园猫",
			Size:           model.SizeSmall,
			Sex:            model.SexFemale,
			AgeMonths:      14,
			Color:          "橘白",
			Sterilized:     true,
			Vaccinated:     true,
			Story:          "在小区车库被救助，亲人爱玩，适合公寓静养。已三联疫苗与驱虫。",
			AllowApartment: true,
			AllowChildren:  true,
			AllowOtherPets: false,
			MinAdopterAge:  18,
			NeedHomeCheck:  needCheck,
			Status:         model.PetPublished,
			PublishedAt:    &pub,
			Personality:    []string{"亲人", "爱玩"},
		},
		{
			ShelterID:      shelter.ID,
			StaffID:        staff.ID,
			Name:           "阿黄",
			Species:        model.SpeciesDog,
			Breed:          "中华田园犬",
			Size:           model.SizeMedium,
			Sex:            model.SexMale,
			AgeMonths:      24,
			Color:          "黄色",
			Sterilized:     true,
			Vaccinated:     true,
			Story:          "看门狗退役后寻找新家，需要每天遛狗，不适合全天无人的小户型。",
			AllowApartment: false,
			AllowChildren:  true,
			AllowOtherPets: true,
			MinAdopterAge:  21,
			NeedHomeCheck:  true,
			Status:         model.PetPublished,
			PublishedAt:    &pub,
			Personality:    []string{"忠诚", "护家"},
		},
		{
			ShelterID:      shelter.ID,
			StaffID:        staff.ID,
			Name:           "棉花糖",
			Species:        model.SpeciesRabbit,
			Breed:          "垂耳兔",
			Size:           model.SizeSmall,
			Sex:            model.SexFemale,
			AgeMonths:      8,
			Color:          "白色",
			Sterilized:     false,
			Vaccinated:     true,
			Story:          "幼兔，需要安静环境和定时喂食干草。草稿待完善健康档案后发布。",
			AllowApartment: true,
			AllowChildren:  false,
			AllowOtherPets: false,
			MinAdopterAge:  18,
			NeedHomeCheck:  false,
			Status:         model.PetDraft,
			Personality:    []string{"安静"},
		},
	}
	for _, p := range pets {
		created, err := st.CreatePet(ctx, p)
		if err != nil {
			return err
		}
		if created.Status == model.PetPublished {
			_, _ = st.CreateHealth(ctx, model.HealthRecord{
				PetID:      created.ID,
				StaffID:    staff.ID,
				Kind:       model.HealthVaccine,
				Title:      "年度疫苗",
				Detail:     "演示数据：已完成基础免疫",
				OccurredAt: now.Add(-40 * 24 * time.Hour),
			})
		}
	}
	return nil
}
