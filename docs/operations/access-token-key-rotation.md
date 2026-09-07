# Access-token signing-key rotation

Synaudio production access tokens use a bounded HS256 keyring. The API must start with all of the following configured:

- `ACCESS_TOKEN_ACTIVE_KID`: the key id used to sign new access tokens.
- `ACCESS_TOKEN_KEYS`: 1-4 comma-separated `kid=secret` entries. Each secret must be at least 32 bytes. The active key must be present.
- `ACCESS_TOKEN_TTL`: lifetime issued to new access tokens.
- `ACCESS_TOKEN_MAX_TTL`: maximum signed lifetime the verifier accepts. It must be greater than or equal to `ACCESS_TOKEN_TTL`.

`ACCESS_TOKEN_SECRET` is only a development compatibility input after the keyring rollout and is not the steady-state production signing contract. Never commit or log production key material.

## First rollout from the legacy single-secret deployment

The pre-keyring production composition created access tokens through `NewAccessTokenManager`, which identifies its configured signing key as `kid=legacy`. The first rollout to keyring configuration therefore has a required compatibility bootstrap; arbitrarily renaming that current key to `old` during the mixed-version rollout is unsafe.

1. Take the exact current production `ACCESS_TOKEN_SECRET` value and configure the new release with `ACCESS_TOKEN_ACTIVE_KID=legacy` and `ACCESS_TOKEN_KEYS=legacy=<current-secret>`.
2. Roll that configuration/release to every serving API replica. During this mixed-version phase, old replicas and new replicas both issue `kid=legacy` with the same key material, and new replicas can verify tokens issued by the old constructor.
3. Verify fleet convergence before introducing a new key id. Do not remove or rename `legacy` while any old-composition replica may still issue tokens.
4. Once every serving replica is on keyring-aware composition, proceed with a normal planned rotation below.

This bootstrap is the only intended production use of the former `ACCESS_TOKEN_SECRET` value: its bytes are migrated into the explicit `legacy` keyring entry for a controlled rollout, not retained as an independent fallback path.

## Planned rotation after keyring convergence

Assume `old` is the currently active key id and `new` is a freshly generated random key. For the first rotation after bootstrap, `old` is `legacy`.

1. **Prepare overlap:** deploy `ACCESS_TOKEN_KEYS=old=<old-secret>,new=<new-secret>` to every serving replica while keeping `ACCESS_TOKEN_ACTIVE_KID=old`.
2. **Verify fleet convergence:** confirm all serving replicas can authenticate tokens carrying either configured key id. Do not switch the signer until every serving replica has the overlap keyring.
3. **Switch signer:** set `ACCESS_TOKEN_ACTIVE_KID=new` while retaining both keys. New tokens now carry `kid=new`; old tokens remain verifiable.
4. **Hold bounded overlap:** retain `old` for at least the maximum access-token lifetime accepted by the verifier (`ACCESS_TOKEN_MAX_TTL`) plus operational clock/rollout margin, measured from the last point any replica could have issued an `old` token.
5. **Remove old key:** after the overlap window and fleet convergence are verified, deploy `ACCESS_TOKEN_KEYS=new=<new-secret>` only.

Mixed replicas during the overlap must not create intermittent 401 responses. Unknown or removed `kid` values fail closed.

## TTL policy changes

Do not calculate rotation timing only from the current `ACCESS_TOKEN_TTL`. Removal safety is based on the maximum lifetime the verifier accepts. When reducing `ACCESS_TOKEN_MAX_TTL`, first ensure no still-valid token issued under the previous accepted lifetime can remain in circulation, or intentionally invalidate those tokens as an emergency/security action.

## Emergency compromise

If a signing key is believed compromised, remove it from `ACCESS_TOKEN_KEYS` as soon as the incident decision requires. Tokens signed by that key will immediately fail verification. This intentionally trades session continuity for containment. Rotate to a fresh active key, verify all replicas have the new keyring, and use normal incident/release evidence to record the invalidation boundary.

## Startup failure semantics

Production startup fails closed when the active key id is missing, the key list is missing/malformed/duplicate/too large, a secret is shorter than 32 bytes, the active id is not present, or the TTL/max-TTL relationship is invalid. Production must not fall back to a development key or to `ACCESS_TOKEN_SECRET`.
