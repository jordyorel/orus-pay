package enterprise

type CreateEnterpriseInput struct {
	Name              string                 `json:"name"`
	BusinessType      string                 `json:"business_type"`
	ContractStartDate string                 `json:"contract_start_date"`
	ContractEndDate   string                 `json:"contract_end_date"`
	CustomPricingPlan map[string]interface{} `json:"custom_pricing_plan"`
}

type APIKeyInput struct {
	KeyName     string `json:"key_name"`
	Environment string `json:"environment"`
}

type ComplianceInput struct {
	Officer string `json:"officer"`
	Email   string `json:"email"`
}
