package identity

import "testing"

func TestIsMfaCapableActiveAdminRequiresConfirmedEnabledMFA(t *testing.T) {
	if IsMfaCapableActiveAdmin(StatusActive, []string{RoleAdmin}, &MFAMethod{Confirmed: true}) {
		// ok
	} else {
		t.Fatal("confirmed enabled MFA admin must be eligible")
	}
	if IsMfaCapableActiveAdmin(StatusActive, []string{RoleAdmin}, nil) {
		t.Fatal("admin without MFA must not count as MFA-capable")
	}
	if IsMfaCapableActiveAdmin(StatusActive, []string{RoleAdmin}, &MFAMethod{Confirmed: true, Disabled: true}) {
		t.Fatal("disabled MFA must not count as MFA-capable")
	}
	if IsMfaCapableActiveAdmin(StatusActive, []string{RoleUser}, &MFAMethod{Confirmed: true}) {
		t.Fatal("non-admin must not count as MFA-capable admin")
	}
}
