package models

type Role string

type Outcome string

const (
	RoleBuyer  Role = "buyer"
	RoleSeller Role = "seller"
)

const (
	OutcomeSafe    Outcome = "safe"
	OutcomePartial Outcome = "partial"
	OutcomeScammed Outcome = "scammed"
)
