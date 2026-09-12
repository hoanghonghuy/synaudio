package openapi

func init() {
	OperationBindings["postApiV1AdminCreativeDecisionsDecisionidPostpone"] = OperationBinding{
		Request:     "PostponeCreativeDecisionRequest",
		Response:    "CreativeDecisionPascal",
		Description: "Creative decision postponed",
	}
}
