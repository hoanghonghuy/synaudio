# Access-token signing-key rotation

Synaudio production access tokens use a bounded HS256 keyring. The API must start with all of the following configured:

- `ACCESS_TOKEN_ACTIVE_KID`: the key id used to sign new access tokens.
- `ACCESS_TOKEN_KEYS`: 1-4 comma-separated `kid=secret` entries. Each secret must be at least 32 bytes. The active key must be present.
- `ACCESS_TOKEN_TTL`: lifetime issued to new access tokens.
- `ACCESS_TOKEN_MAX_TTL`: maximum signed lifetime the verifier accepts. It must be greater than or equal to `ACCESS_TOKEN_TTL`.

`ACCESS_TOKEN_SECRET` is only a development compatibility input and is not a production signing contract. Never commit or log production key material.

## Planned rotation

Assume `old` is the current signer and `new` is a freshly generated random key.

1. **Prepare overlap:** deploy `ACCESS_TOKEN_KEYS=old=<old-secret>,new=<new-secret>` to every serving replica while keeping `ACCESS_TOKEN_ACTIVE_KID=old`.
2. **Verify fleet convergence:** confirm all serving replicas can authenticate tokens carrying either `kid=old` or `kid=new`. Do not switch the signer while replicas with the old single-key view are still serving traffic.
3. **Switch signer:** set `ACCESS_TOKEN_ACTIVE_KID=new` while retaining both keys. New tokens now carry `kid=new`; old tokens remain verifiable.
4. **Hold bounded overlap:** retain `old` for at least the maximum access-token lifetime accepted by the verifier (`ACCESS_TOKEN_MAX_TTL`) plus operational clock/rollout margin, measured from the last point any replica could have issued an `old` token.
5. **Remove old key:** after the overlap window and fleet convergence are verified, deploy `ACCESS_TOKEN_KEYS=new=<new-secret>` only.

Mixed old/new replicas during steps 1-4 must not create intermittent 401 responses. Unknown or removed `kid` values fail closed.

## TTL policy changes

Do not calculate rotation timing only from the current `ACCESS_TOKEN_TTL`. Removal safety is based on the maximum lifetime the verifier accepts. When reducing `ACCESS_TOKEN_MAX_TTL`, first ensure no still-valid token issued under the previous accepted lifetime can remain in circulation, or intentionally invalidate those tokens as an emergency/security action.

## Emergency compromise

If a signing key is believed compromised, remove it from `ACCESS_TOKEN_KEYS` as soon as the incident decision requires. Tokens signed by that key will immediately fail verification. This intentionally trades session continuity for containment. Rotate to a fresh active key, verify all replicas have the new keyring, and use normal incident/release evidence to record the invalidation boundary.

## Startup failure semantics

Production startup fails closed when the active key id is missing, the key list is missing/malformed/duplicate/too large, a secret is shorter than 32 bytes, the active id is not present, or the TTL/max-TTL relationship is invalid. Production must not fall back to a development key or to `ACCESS_TOKEN_SECRET`.
