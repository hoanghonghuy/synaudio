import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const auth = await readFile(new URL('../src/features/auth/AuthPage.vue', import.meta.url), 'utf8')
const styles = await readFile(new URL('../src/styles.css', import.meta.url), 'utf8')

test('password toggle stays inside a dedicated input wrapper', () => {
  assert.match(auth, /class="password-wrap"[\s\S]*class="password-toggle-btn"/)
  assert.match(auth, /class="password-toggle-btn"[\s\S]*:aria-label="showPassword \? 'Ẩn mật khẩu' : 'Hiện mật khẩu'"/)
})

test('auth submit sizing never catches the password toggle', () => {
  assert.doesNotMatch(styles, /\.auth-form button:not\(\.text-button\)/)
  assert.match(styles, /\.auth-form button\[type="submit"\], \.auth-submit-btn\s*\{[^}]*width:\s*100%/)
  assert.match(styles, /\.password-toggle-btn\s*\{[^}]*position:\s*absolute;[^}]*width:\s*auto !important;[^}]*min-height:\s*32px !important;/)
})
