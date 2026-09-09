package httpapi

import "net/http"

type adminRoutePolicy struct {
	Permission string
	RecentAuth bool
}

const unmappedAdminPermission = "__UNMAPPED_ADMIN_ROUTE__"

// adminPolicyFor returns the minimum permission for each current privileged
// route family. Unknown privileged routes deliberately receive a permission
// that is never granted so newly-added admin endpoints fail closed until their
// operation-specific policy is defined.
func adminPolicyFor(method, route string) adminRoutePolicy {
	if route == "/admin/audit" || route == "/admin/audit/{eventID}" {
		return adminRoutePolicy{Permission: "AUDIT_VIEW"}
	}

	if route == "/admin/users" && method == http.MethodGet {
		return adminRoutePolicy{Permission: "USER_STATUS_MANAGE"}
	}
	if route == "/admin/users/{userID}" && method == http.MethodGet {
		return adminRoutePolicy{Permission: "USER_STATUS_MANAGE"}
	}
	if hasPrefix(route, "/admin/users/") {
		switch {
		case hasSuffix(route, "/roles/admin") && method == http.MethodPost:
			return adminRoutePolicy{Permission: "ADMIN_ROLE_GRANT", RecentAuth: true}
		case hasSuffix(route, "/roles/admin") && method == http.MethodDelete:
			return adminRoutePolicy{Permission: "ADMIN_ROLE_REVOKE", RecentAuth: true}
		case hasSuffix(route, "/status") && method == http.MethodPatch:
			return adminRoutePolicy{Permission: "ADMIN_STATUS_MANAGE", RecentAuth: true}
		}
	}

	if hasPrefix(route, "/admin/retcons") {
		switch {
		case method == http.MethodGet:
			return adminRoutePolicy{Permission: "RETCON_VIEW"}
		case hasSuffix(route, "/approve"):
			return adminRoutePolicy{Permission: "RETCON_APPROVE", RecentAuth: true}
		case hasSuffix(route, "/apply"):
			return adminRoutePolicy{Permission: "RETCON_APPLY", RecentAuth: true}
		case hasSuffix(route, "/cancel"):
			return adminRoutePolicy{Permission: "RETCON_CANCEL", RecentAuth: true}
		default:
			return adminRoutePolicy{Permission: "RETCON_REQUEST"}
		}
	}

	if hasPrefix(route, "/admin/stories") {
		switch {
		case hasSuffix(route, "/activate"):
			return adminRoutePolicy{Permission: "STORY_ACTIVATE"}
		case hasSuffix(route, "/archive"):
			return adminRoutePolicy{Permission: "STORY_ARCHIVE"}
		case hasSuffix(route, "/restore"):
			return adminRoutePolicy{Permission: "STORY_RESTORE"}
		case hasSuffix(route, "/make-public") || hasSuffix(route, "/make-private"):
			return adminRoutePolicy{Permission: "STORY_VISIBILITY_MANAGE"}
		case hasSuffix(route, "/cover"):
			return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"}
		case hasSegment(route, "/content-profile"):
			return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"}
		case hasSegment(route, "/metadata"):
			return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"}
		case hasSegment(route, "/activation-readiness") || hasSegment(route, "/generation-policy"):
			return adminRoutePolicy{Permission: "STORY_ACTIVATE"}
		case hasSegment(route, "/bible"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "STORY_BIBLE_VIEW"}
			}
			return adminRoutePolicy{Permission: "STORY_BIBLE_MANAGE"}
		case hasSegment(route, "/characters"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "CHARACTER_VIEW"}
			}
			return adminRoutePolicy{Permission: "CHARACTER_MANAGE"}
		case hasSegment(route, "/arcs"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "ARC_VIEW"}
			}
			return adminRoutePolicy{Permission: "ARC_MANAGE"}
		case hasSegment(route, "/ending"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "ENDING_PLAN_VIEW"}
			}
			return adminRoutePolicy{Permission: "ENDING_PLAN_MANAGE"}
		case hasSegment(route, "/facts"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "STORY_FACT_VIEW"}
			}
			return adminRoutePolicy{Permission: "STORY_BIBLE_MANAGE"}
		case hasSegment(route, "/plot-threads"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "PLOT_THREAD_VIEW"}
			}
			return adminRoutePolicy{Permission: "STORY_BIBLE_MANAGE"}
		case hasSegment(route, "/canon-repair"):
			return adminRoutePolicy{Permission: "CANON_DATA_REPAIR_REQUEST"}
		case hasSegment(route, "/context-snapshots"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "CANON_VIEW"}
			}
			return adminRoutePolicy{Permission: "CANON_DATA_REPAIR_REQUEST"}
		case hasSegment(route, "/creative-decisions"):
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "CREATIVE_DECISION_VIEW"}
			}
			return adminRoutePolicy{Permission: "CREATIVE_DECISION_RESOLVE"}
		case hasSegment(route, "/attention"):
			return adminRoutePolicy{Permission: "STORY_WORKFLOW_SETTINGS_MANAGE"}
		case hasSegment(route, "/workflow"):
			return adminRoutePolicy{Permission: "STORY_WORKFLOW_SETTINGS_MANAGE"}
		case hasSegment(route, "/foundation"):
			return adminRoutePolicy{Permission: "STORY_BIBLE_MANAGE"}
		case hasSegment(route, "/batch-generate"):
			return adminRoutePolicy{Permission: "GENERATION_START"}
		case hasSegment(route, "/usage"):
			return adminRoutePolicy{Permission: "GENERATION_VIEW"}
		case hasSuffix(route, "/chapters") && method == http.MethodPost:
			return adminRoutePolicy{Permission: "CHAPTER_CREATE"}
		case method == http.MethodPost && route == "/admin/stories":
			return adminRoutePolicy{Permission: "STORY_CREATE"}
		case method == http.MethodGet:
			return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"}
		}
	}

	if hasPrefix(route, "/admin/chapters") {
		if hasSegment(route, "/narration") {
			switch {
			case hasSuffix(route, "/activate") || hasSuffix(route, "/activate-version"):
				return adminRoutePolicy{Permission: "AUDIO_ACTIVATE_VERSION"}
			case hasSuffix(route, "/retry"):
				return adminRoutePolicy{Permission: "AUDIO_RETRY"}
			case method == http.MethodGet:
				return adminRoutePolicy{Permission: "NARRATION_VIEW"}
			default:
				return adminRoutePolicy{Permission: "NARRATION_GENERATE"}
			}
		}
		if hasSegment(route, "/audio") {
			switch {
			case hasSuffix(route, "/activate"):
				return adminRoutePolicy{Permission: "AUDIO_ACTIVATE_VERSION"}
			case method == http.MethodGet:
				return adminRoutePolicy{Permission: "AUDIO_REVIEW"}
			default:
				return adminRoutePolicy{Permission: "AUDIO_GENERATE"}
			}
		}
		if hasSegment(route, "/plan") || hasSegment(route, "/plans") {
			if method == http.MethodGet {
				return adminRoutePolicy{Permission: "CHAPTER_PLAN_VIEW"}
			}
			return adminRoutePolicy{Permission: "CHAPTER_PLAN_MANAGE"}
		}
		if hasSegment(route, "/reviews") {
			return adminRoutePolicy{Permission: "CHAPTER_REVIEW"}
		}
		if hasSuffix(route, "/approve") {
			return adminRoutePolicy{Permission: "CHAPTER_APPROVE_CONTENT"}
		}
		if hasSuffix(route, "/publish") {
			return adminRoutePolicy{Permission: "CHAPTER_PUBLISH"}
		}
		if hasSuffix(route, "/unpublish") {
			return adminRoutePolicy{Permission: "CHAPTER_UNPUBLISH"}
		}
		if hasSuffix(route, "/ready") {
			return adminRoutePolicy{Permission: "CHAPTER_PUBLISH"}
		}
		if hasSuffix(route, "/regenerate") {
			return adminRoutePolicy{Permission: "GENERATION_REGENERATE"}
		}
		if hasSuffix(route, "/rewrite") {
			return adminRoutePolicy{Permission: "GENERATION_REWRITE"}
		}
		if hasSuffix(route, "/edit") {
			return adminRoutePolicy{Permission: "CHAPTER_EDIT_DRAFT"}
		}
		if hasSuffix(route, "/content") && method == http.MethodGet {
			return adminRoutePolicy{Permission: "GENERATION_VIEW"}
		}
		if hasSuffix(route, "/content") {
			return adminRoutePolicy{Permission: "CHAPTER_GENERATE"}
		}
		if hasSegment(route, "/revision-impact") {
			return adminRoutePolicy{Permission: "CHAPTER_REVISE_PRE_PUBLISH"}
		}
		if hasSegment(route, "/generation-run") {
			return adminRoutePolicy{Permission: "GENERATION_VIEW"}
		}
		if hasSuffix(route, "/publish-readiness") {
			return adminRoutePolicy{Permission: "CHAPTER_PUBLISH"}
		}
	}

	if hasPrefix(route, "/admin/canon-") || hasPrefix(route, "/admin/context-snapshots") {
		if hasSuffix(route, "/commit") {
			return adminRoutePolicy{Permission: "CANON_DATA_REPAIR_APPLY"}
		}
		if hasSuffix(route, "/promote") {
			return adminRoutePolicy{Permission: "CANON_DATA_REPAIR_APPLY"}
		}
		if method == http.MethodGet {
			return adminRoutePolicy{Permission: "CANON_VIEW"}
		}
		return adminRoutePolicy{Permission: "CANON_DATA_REPAIR_REQUEST"}
	}

	if hasPrefix(route, "/admin/creative-decisions") {
		if hasSuffix(route, "/reject") {
			return adminRoutePolicy{Permission: "CREATIVE_DECISION_REJECT"}
		}
		if hasSuffix(route, "/postpone") {
			return adminRoutePolicy{Permission: "CREATIVE_DECISION_POSTPONE"}
		}
		return adminRoutePolicy{Permission: "CREATIVE_DECISION_RESOLVE"}
	}

	if hasPrefix(route, "/admin/plot-threads") || hasPrefix(route, "/admin/attention") {
		return adminRoutePolicy{Permission: "STORY_BIBLE_MANAGE"}
	}

	if hasPrefix(route, "/admin/revisions/") {
		return adminRoutePolicy{Permission: "CHAPTER_REVIEW"}
	}

	if hasPrefix(route, "/admin/generation") || hasPrefix(route, "/admin/runs") || hasPrefix(route, "/admin/attempts") {
		if method == http.MethodGet {
			return adminRoutePolicy{Permission: "GENERATION_VIEW"}
		}
		if hasSuffix(route, "/retry") {
			return adminRoutePolicy{Permission: "GENERATION_RETRY"}
		}
		if hasSuffix(route, "/cancel") {
			return adminRoutePolicy{Permission: "GENERATION_CANCEL"}
		}
		if hasSuffix(route, "/mark-stale") {
			return adminRoutePolicy{Permission: "GENERATION_CANCEL"}
		}
		return adminRoutePolicy{Permission: "GENERATION_START"}
	}

	if hasPrefix(route, "/admin/tts-segments") {
		return adminRoutePolicy{Permission: "AUDIO_GENERATE"}
	}

	return adminRoutePolicy{Permission: unmappedAdminPermission}
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

func hasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}

func hasSegment(value, segment string) bool {
	for i := 0; i+len(segment) <= len(value); i++ {
		if value[i:i+len(segment)] == segment {
			return true
		}
	}
	return false
}
