import type { RetconStatus } from '../../api/retcon'

export type RetconAction = 'analyze' | 'approve' | 'ready' | 'apply' | 'cancel'
export function retconActions(status: RetconStatus): RetconAction[]
export function retconStatusLabel(status: RetconStatus): string
export function retconActionLabel(action: RetconAction): string
export function retconActionIsDestructive(action: RetconAction): boolean
