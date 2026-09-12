export type CreativeDecisionAction = 'select' | 'reject' | 'postpone'

export function isCreativeDecisionActionable(status: string): boolean
export function creativeDecisionActionRequiresNote(action: CreativeDecisionAction): boolean
