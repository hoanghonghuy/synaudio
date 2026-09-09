package identity

// IsMfaCapableActiveAdmin reports whether a user is an ACTIVE administrator with
// confirmed, non-disabled MFA. This is the eligibility contract shared by
// privileged-session assurance and the Last Active Admin invariant.
func IsMfaCapableActiveAdmin(status string, roles []string, mfa *MFAMethod) bool {
	if status != StatusActive {
		return false
	}
	admin := false
	for _, role := range roles {
		if role == RoleAdmin {
			admin = true
			break
		}
	}
	if !admin {
		return false
	}
	return mfa != nil && mfa.Confirmed && !mfa.Disabled
}
