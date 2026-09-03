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
			if method == http.MethodGet { return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"} }
			return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"}
		case hasSegment(route, "/bible"):
			if method == http.MethodGet { return adminRoutePolicy{Permission: "STORY_BIBLE_VIEW"} }
			return adminRoutePolicy{Permission: "STORY_BIBLE_MANAGE"}
		case hasSegment(route, "/characters"):
			if method == http.MethodGet { return adminRoutePolicy{Permission: "CHARACTER_VIEW"} }
			return adminRoutePolicy{Permission: "CHARACTER_MANAGE"}
		case hasSegment(route, "/arcs"):
			if method == http.MethodGet { return adminRoutePolicy{Permission: "ARC_VIEW"} }
			return adminRoutePolicy{Permission: "ARC_MANAGE"}
		case hasSegment(route, "/ending"):
			if method == http.MethodGet { return adminRoutePolicy{Permission: "ENDING_PLAN_VIEW"} }
			return adminRoutePolicy{Permission: "ENDING_PLAN_MANAGE"}
		case hasSegment(route, "/decisions"):
			if method == http.MethodGet { return adminRoutePolicy{Permission: "CREATIVE_DECISION_VIEW"} }
			if hasSuffix(route, "/postpone") { return adminRoutePolicy{Permission: "CREATIVE_DECISION_POSTPONE"} }
			if hasSuffix(route, "/reject") { return adminRoutePolicy{Permission: "CREATIVE_DECISION_REJECT"} }
			return adminRoutePolicy{Permission: "CREATIVE_DECISION_RESOLVE"}
		case hasSegment(route, "/workflow"):
			return adminRoutePolicy{Permission: "STORY_WORKFLOW_SETTINGS_MANAGE"}
		case method == http.MethodPost && route == "/admin/stories":
			return adminRoutePolicy{Permission: "STORY_CREATE"}
		case method == http.MethodGet:
			return adminRoutePolicy{Permission: "STORY_METADATA_EDIT"}
		}
	}

	if hasPrefix(route, "/admin/chapters") {
		if hasSegment(route, "/narration") {
			switch {
			case hasSuffix(route, "/approve"):
				return adminRoutePolicy{Permission: "NARRATION_APPROVE"}
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
		if hasSegment(route, "/plan") {
			if method == http.MethodGet { return adminRoutePolicy{Permission: "CHAPTER_PLAN_VIEW"} }
			return adminRoutePolicy{Permission: "CHAPTER_PLAN_MANAGE"}
		}
		if hasSegment(route, "/reviews") {
			return adminRoutePolicy{Permission: "CHAPTER_REVIEW"}
		}
		if hasSuffix(route, "/approve") {
			return adminRoutePolicy{Permission: "CHAPTER_APPROVE_CONTENT"}
		}
		if hasSegment(route, "/revision-impact") {
			return adminRoutePolicy{Permission: "CHAPTER_REVISE_PRE_PUBLISH"}
		}
	}

	if hasPrefix(route, "/admin/generation") || hasSegment(route, "/generation") {
		if method == http.MethodGet { return adminRoutePolicy{Permission: "GENERATION_VIEW"} }
		if hasSuffix(route, "/retry") { return adminRoutePolicy{Permission: "GENERATION_RETRY"} }
		if hasSuffix(route, "/cancel") { return adminRoutePolicy{Permission: "GENERATION_CANCEL"} }
		if hasSuffix(route, "/regenerate") { return adminRoutePolicy{Permission: "GENERATION_REGENERATE"} }
		return adminRoutePolicy{Permission: "GENERATION_START"}
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
		if value[i:i+len(segment)] == segment { return true }
	}
	return false
}
