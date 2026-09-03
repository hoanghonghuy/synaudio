package audit

import (
	"context"
	"net/http"
)

// TransactionBoundary executes one audited HTTP mutation inside a persistence
// transaction. The callback context must route domain and audit DB calls to the
// same transaction. A successful callback is committed by the boundary; an
// error must roll the transaction back.
type TransactionBoundary func(context.Context, func(context.Context) error) error

// WrapRouteTransactional is the durable form of WrapRoute. The business handler
// and semantic audit write share one transaction, so an audit persistence outage
// cannot occur after a committed DB mutation: either both commit or neither does.
func WrapRouteTransactional(next http.Handler, method, route string, record RecordFunc, resolveActor ActorResolver, boundary TransactionBoundary) http.Handler {
	if boundary == nil || record == nil || !isMutation(method) {
		return WrapRoute(next, method, route, record, resolveActor)
	}
	desc, shouldAudit := auditDescriptor(method, route)
	if !shouldAudit {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorID := ""
		if resolveActor != nil {
			actorID, _ = resolveActor(r.Context(), r)
		}

		buffer := newBufferedResponse()
		err := boundary(r.Context(), func(txCtx context.Context) error {
			txReq := r.WithContext(txCtx)
			next.ServeHTTP(buffer, txReq)
			status := buffer.status
			if status == 0 {
				status = http.StatusOK
			}
			event := eventForResponse(txReq, method, route, desc, actorID, status, buffer.body.Bytes())
			_, recordErr := record(txCtx, event)
			return recordErr
		})
		if err != nil {
			status := buffer.status
			if status == 0 {
				status = http.StatusOK
			}
			if status >= http.StatusBadRequest {
				buffer.header.Set("X-Synaudio-Audit-Status", "unavailable")
				buffer.flushTo(w)
				return
			}
			writeAuditTransactionFailure(w)
			return
		}
		buffer.flushTo(w)
	})
}

// WrapAuthTransactional provides the same atomic durability boundary for auth
// mutations without inspecting request bodies or weakening the existing secret
// redaction guarantees.
func WrapAuthTransactional(next http.Handler, record RecordFunc, resolveActor ActorResolver, boundary TransactionBoundary) http.Handler {
	if boundary == nil || record == nil {
		return WrapAuth(next, record, resolveActor)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMutation(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		route := authRoute(r.URL.Path)
		desc := describeAuthRoute(r.Method, route)
		actorID := ""
		if resolveActor != nil {
			actorID, _ = resolveActor(r.Context(), r)
		}

		buffer := newBufferedResponse()
		err := boundary(r.Context(), func(txCtx context.Context) error {
			txReq := r.WithContext(txCtx)
			next.ServeHTTP(buffer, txReq)
			status := buffer.status
			if status == 0 {
				status = http.StatusOK
			}
			event := eventForResponse(txReq, r.Method, route, desc, actorID, status, buffer.body.Bytes())
			_, recordErr := record(txCtx, event)
			return recordErr
		})
		if err != nil {
			status := buffer.status
			if status == 0 {
				status = http.StatusOK
			}
			if status >= http.StatusBadRequest {
				buffer.header.Set("X-Synaudio-Audit-Status", "unavailable")
				buffer.flushTo(w)
				return
			}
			writeAuditTransactionFailure(w)
			return
		}
		buffer.flushTo(w)
	})
}

func writeAuditTransactionFailure(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Synaudio-Audit-Status", "unavailable")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":{"code":"AUDIT_TRANSACTION_FAILED","message":"request could not be committed"}}`))
}
