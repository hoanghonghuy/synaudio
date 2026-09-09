package identity_test

import "github.com/synaudio/synaudio/backend/internal/identity"

func countMfaCapableActiveAdmins(store *fakeStore) int {
	count := 0
	for _, user := range store.users {
		if identity.IsMfaCapableActiveAdmin(user.Status, store.userRoles[user.ID], store.mfaMethods[user.ID]) {
			count++
		}
	}
	return count
}

func isMfaCapableActiveAdmin(store *fakeStore, userID string) bool {
	for _, user := range store.users {
		if user.ID != userID {
			continue
		}
		return identity.IsMfaCapableActiveAdmin(user.Status, store.userRoles[user.ID], store.mfaMethods[user.ID])
	}
	return false
}
