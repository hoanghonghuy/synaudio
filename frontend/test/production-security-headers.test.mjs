import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const nginx = await readFile(new URL('../nginx.conf', import.meta.url), 'utf8')

function header(name) {
  const match = nginx.match(new RegExp(`add_header\\s+${name}\\s+"([^"]+)"\\s+always;`))
  assert.ok(match, `${name} must be emitted with always`)
  return match[1]
}

test('production nginx owns a restrictive browser security-header baseline', () => {
  const csp = header('Content-Security-Policy')

  for (const directive of [
    "default-src 'self'",
    "base-uri 'self'",
    "object-src 'none'",
    "frame-ancestors 'none'",
    "form-action 'self'",
    "script-src 'self'",
    "connect-src 'self'",
    "media-src 'self' https: blob:",
  ]) {
    assert.match(csp, new RegExp(directive.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
  }

  assert.doesNotMatch(csp, /unsafe-eval/)
  assert.doesNotMatch(csp, /unsafe-inline/)
  assert.doesNotMatch(csp, /(?:^|\s)\*(?:\s|;|$)/)

  assert.equal(header('X-Frame-Options'), 'DENY')
  assert.equal(header('X-Content-Type-Options'), 'nosniff')
  assert.equal(header('Referrer-Policy'), 'strict-origin-when-cross-origin')

  const permissions = header('Permissions-Policy')
  for (const capability of ['camera=()', 'microphone=()', 'geolocation=()', 'payment=()', 'usb=()']) {
    assert.match(permissions, new RegExp(capability.replace(/[()]/g, '\\$&')))
  }
})

test('HTTP-only frontend container does not falsely claim HSTS ownership', () => {
  assert.match(nginx, /listen 8080;/)
  assert.doesNotMatch(nginx, /add_header\s+Strict-Transport-Security/i)
})
