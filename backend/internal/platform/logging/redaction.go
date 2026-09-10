package logging

import (
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

// sensitiveKeyFragments mirrors audit metadata redaction but is owned by the
// application-log contract (#57). Audit (#10) and metrics (#37) remain separate.
var sensitiveKeyFragments = []string{
	"password", "passwd", "authorization", "access_token", "refreshtoken", "refresh_token",
	"tokenhash", "token_hash", "totp", "mfa_secret", "recovery_code", "recoverycode",
	"api_key", "apikey", "secret_key", "secretkey", "provider_secret", "cookie",
	"database_url", "presign", "presigned", "prompt", "story_text", "story_content",
	"chapter_text", "audio_bytes", "audio_data", "inline_data", "reset_token",
	"verification_token", "action_url", "action_link", "email_link",
}

var (
	bearerTokenPattern    = regexp.MustCompile(`(?i)Bearer\s+[^\s]+`)
	dbURLCredentialPattern = regexp.MustCompile(`(?i)(postgres|mysql|mongodb)(\+[a-z0-9]+)?://[^:]+:[^@]+@`)
	presignedURLPattern   = regexp.MustCompile(`(?i)(X-Amz-Signature|X-Amz-Credential|X-Amz-Algorithm|Signature=)[^\s&"]*`)
	queryTokenPattern     = regexp.MustCompile(`(?i)([?&](?:token|reset|verify|key|code|signature|access_token|refresh_token)=)[^&\s"]+`)
	apiKeyHeaderPattern   = regexp.MustCompile(`(?i)x-goog-api-key[=:\s]+[^\s,;]+`)
	longBase64Pattern     = regexp.MustCompile(`[A-Za-z0-9+/]{200,}={0,2}`)
)

func sensitiveAttributeKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	for _, fragment := range sensitiveKeyFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

// RedactString applies value-level redaction to free-form text such as error
// messages that may contain credentials, presigned URLs, or action links.
func RedactString(value string) string {
	if value == "" {
		return value
	}
	out := value
	out = bearerTokenPattern.ReplaceAllString(out, "Bearer "+redacted)
	out = dbURLCredentialPattern.ReplaceAllString(out, redacted+"://[REDACTED]@")
	out = presignedURLPattern.ReplaceAllString(out, redacted)
	out = queryTokenPattern.ReplaceAllString(out, redacted)
	out = apiKeyHeaderPattern.ReplaceAllString(out, "x-goog-api-key="+redacted)
	out = longBase64Pattern.ReplaceAllString(out, redacted)
	out = redactPresignedHosts(out)
	return out
}

func redactPresignedHosts(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		u, err := url.Parse(part)
		if err != nil || u.Host == "" {
			continue
		}
		q := u.Query()
		if q.Has("X-Amz-Signature") || q.Has("X-Amz-Credential") || q.Has("Signature") {
			parts[i] = redacted
		}
	}
	return strings.Join(parts, " ")
}

func redactValue(key string, value any) any {
	if sensitiveAttributeKey(key) {
		return redacted
	}
	switch typed := value.(type) {
	case string:
		return RedactString(typed)
	case []byte:
		return redacted
	case slog.Value:
		return redactSlogValue(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactValue(key, item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, v := range typed {
			out[k] = redactValue(k, v)
		}
		return out
	default:
		if err, ok := typed.(error); ok {
			return SafeError(err)
		}
		return typed
	}
}

func redactSlogValue(value slog.Value) slog.Value {
	switch value.Kind() {
	case slog.KindString:
		return slog.StringValue(RedactString(value.String()))
	case slog.KindGroup:
		attrs := value.Group()
		out := make([]slog.Attr, len(attrs))
		for i, attr := range attrs {
			out[i] = redactAttr(attr)
		}
		return slog.GroupValue(out...)
	case slog.KindLogValuer:
		return redactSlogValue(value.Resolve())
	default:
		return value
	}
}

func redactAttr(attr slog.Attr) slog.Attr {
	if attr.Equal(slog.Attr{}) {
		return attr
	}
	if sensitiveAttributeKey(attr.Key) {
		return slog.Any(attr.Key, redacted)
	}
	if attr.Value.Kind() == slog.KindGroup {
		attrs := attr.Value.Group()
		out := make([]any, 0, len(attrs)*2)
		for _, child := range attrs {
			redactedChild := redactAttr(child)
			out = append(out, redactedChild.Key, redactedChild.Value.Any())
		}
		return slog.Group(attr.Key, out...)
	}
	return slog.Any(attr.Key, redactValue(attr.Key, attr.Value.Any()))
}
