# V1 HTTP contract

`api.yaml` is the executable contract for the supported V1 HTTP surface.

Rules:

- A public HTTP route change must update `api.yaml` in the same pull request.
- Authentication in the contract follows the V1 access/refresh architecture: normal protected operations use Bearer access tokens; refresh credentials remain an auth endpoint concern and are not a general API authorization mechanism.
- Admin operations remain Bearer-authenticated here. Operation-specific privileged authorization may be extended by the granular authorization workstream without weakening the current boundary.
- Path parameters must be declared as required OpenAPI path parameters and every operation must have a unique `operationId`.
- Shared error/security schemas live under `components` rather than being redefined inconsistently per operation.
- Reusable request/response schemas live in `schemas.yaml` and are bound to operations via `bindings.go` (`OperationBindings`). Every operation must declare concrete JSON `requestBody` and success-response content schemas (multipart where applicable).
- Regenerate enriched `api.yaml` with `REGEN_OPENAPI=1 go test ./openapi -run TestRegenerateAPIContract` after changing bindings or schemas.
- `frontend/src/api/openapi.generated.ts` is derived from this contract surface (operations + schema types). Do not edit it independently. Regenerate with `REGEN_OPENAPI_FRONTEND=1 go test ./openapi -run TestRegenerateFrontendContract`.
- `backend/openapi/contract_test.go` compares chi route registrations with OpenAPI method/path operations and validates schema coverage via `ValidateOperationSchemas`.

The contract test deliberately has no dependency on a live database, AI provider, TTS provider, or external network service.
