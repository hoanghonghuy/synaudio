package httpapi

import "net/http"

type adminRoutePolicy struct {
	Permission string
	RecentAuth bool
}

// adminPolicyFor returns the minimum permission for each current privileged
// route family. Unknown privileged routes intentionally return no policy so the
// router can fall back to the existing assured-admin gate until a concrete
// permission is defined, avoiding accidental product-contract invention.
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

	return adminRoutePolicy{}
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
