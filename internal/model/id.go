package model

// 资源 ID 与会话 Token 前缀。store 层生成 ID 时拼接此前缀，便于排查与日志检索。

const (
	UserIDPrefix         = "u_"
	ShelterIDPrefix      = "s_"
	PetIDPrefix          = "p_"
	ApplicationIDPrefix  = "a_"
	VisitIDPrefix        = "v_"
	HealthIDPrefix       = "h_"
	FavoriteIDPrefix     = "f_"
	InquiryIDPrefix      = "q_"
	NotificationIDPrefix = "n_"
	AuditLogIDPrefix     = "l_"
	CreditLogIDPrefix    = "c_"
	TokenPrefix          = "t_"
)
