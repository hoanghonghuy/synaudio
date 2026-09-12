package openapi

func init() {
	OperationBindings["postApiV1AdminCreativeDecisionsDecisionidPostpone"] = OperationBinding{
		Request:     "RejectContentRequest",
		Response:    "CreativeDecisionPascal",
		Description: "Creative decision postponed",
	}
}
