import assert from 'node:assert/strict'
import test from 'node:test'

import {
  creativeDecisionActionRequiresNote,
  isCreativeDecisionActionable,
} from '../src/features/admin/creativeDecisionGovernance.mjs'

test('only PROPOSED creative decisions expose governance actions', () => {
  assert.equal(isCreativeDecisionActionable('PROPOSED'), true)
  assert.equal(isCreativeDecisionActionable('SELECTED'), false)
  assert.equal(isCreativeDecisionActionable('REJECTED'), false)
  assert.equal(isCreativeDecisionActionable('POSTPONED'), false)
})

test('reject and postpone require operator context while select does not', () => {
  assert.equal(creativeDecisionActionRequiresNote('select'), false)
  assert.equal(creativeDecisionActionRequiresNote('reject'), true)
  assert.equal(creativeDecisionActionRequiresNote('postpone'), true)
})
