export function isCreativeDecisionActionable(status) {
  return status === 'PROPOSED'
}

export function creativeDecisionActionRequiresNote(action) {
  return action === 'reject' || action === 'postpone'
}
