# V1 HTTP contract

`api.yaml` is the executable contract for the supported V1 HTTP surface.

Rules:

- A public HTTP route change must update `api.yaml` in the same pull request.
- Authentication in the contract follows the V1 access/refresh architecture: normal protected operations use Bearer access tokens; refresh credentials remain an auth endpoint concern and are not a general API authorization mechanism.
- Admin operations remain Bearer-authenticated here. Operation-specific privileged authorization may be extended by the granular authorization workstream without weakening the current boundary.
- Path parameters must be declared as required OpenAPI path parameters and every operation must have a unique `operationId`.
- Shared error/security schemas live under `components` rather than being redefined inconsistently per operation.
- `frontend/src/api/openapi.generated.ts` is derived from this contract surface. Do not edit it independently. The backend OpenAPI contract test compares the generated boundary byte-for-byte and fails on drift.
- `backend/openapi/contract_test.go` also compares the actual chi route registrations with the OpenAPI method/path set, so adding or deleting an HTTP route without updating the contract is a CI failure.

The contract test deliberately has no dependency on a live database, AI provider, TTS provider, or external network service.
