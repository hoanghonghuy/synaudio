package retcon

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetconLifecycleRejectsBackwardAndTerminalTransitions(t *testing.T) {
	tests := []struct {
		name   string
		status string
		act    func(*Service, string) error
	}{
		{
			name:   "approve draft before analysis",
			status: "DRAFT",
			act: func(svc *Service, id string) error {
				_, err := svc.ApproveRetconRequest(context.Background(), id, "admin-2")
				return err
			},
		},
		{
			name:   "analyze approved",
			status: "APPROVED",
			act: func(svc *Service, id string) error {
				_, err := svc.AnalyzeRetconRequest(context.Background(), id)
				return err
			},
		},
		{
			name:   "approve applied",
			status: "APPLIED",
			act: func(svc *Service, id string) error {
				_, err := svc.ApproveRetconRequest(context.Background(), id, "admin-2")
				return err
			},
		},
		{
			name:   "approve cancelled",
			status: "CANCELLED",
			act: func(svc *Service, id string) error {
				_, err := svc.ApproveRetconRequest(context.Background(), id, "admin-2")
				return err
			},
		},
		{
			name:   "cancel applied",
			status: "APPLIED",
			act: func(svc *Service, id string) error {
				_, err := svc.CancelRetconRequest(context.Background(), id)
				return err
			},
		},
		{
			name:   "cancel cancelled",
			status: "CANCELLED",
			act: func(svc *Service, id string) error {
				_, err := svc.CancelRetconRequest(context.Background(), id)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			store.retcons["story-1"] = []RetconRequest{{ID: "retcon-1", StoryID: "story-1", Status: tt.status}}
			svc := NewService(store)

			if err := tt.act(svc, "retcon-1"); !errors.Is(err, ErrRetconInvalidTransition) {
				t.Fatalf("error = %v, want ErrRetconInvalidTransition", err)
			}

			got, err := store.GetRetconRequest(context.Background(), "retcon-1")
			if err != nil {
				t.Fatalf("GetRetconRequest: %v", err)
			}
			if got.Status != tt.status {
				t.Fatalf("status mutated to %q, want %q", got.Status, tt.status)
			}
			if got.ApprovedBy != "" {
				t.Fatalf("approved_by mutated to %q on rejected transition", got.ApprovedBy)
			}
		})
	}
}

func TestRetconLifecycleAllowsDraftAnalyzeApproveReadyApply(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	ret, err := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "story-1", Reason: "repair", RequestedBy: "admin-1",
	})
	if err != nil {
		t.Fatalf("CreateRetconRequest: %v", err)
	}
	ret, err = svc.AnalyzeRetconRequest(context.Background(), ret.ID)
	if err != nil || ret.Status != "ANALYZING" {
		t.Fatalf("AnalyzeRetconRequest: status=%q err=%v", ret.Status, err)
	}
	ret, err = svc.ApproveRetconRequest(context.Background(), ret.ID, "admin-2")
	if err != nil || ret.Status != "APPROVED" {
		t.Fatalf("ApproveRetconRequest: status=%q err=%v", ret.Status, err)
	}
	if ret.ApprovedBy != "admin-2" {
		t.Fatalf("ApproveRetconRequest: approvedBy=%q, want admin-2", ret.ApprovedBy)
	}
	ret, err = svc.MarkReadyToApply(context.Background(), ret.ID)
	if err != nil || ret.Status != "READY_TO_APPLY" {
		t.Fatalf("MarkReadyToApply: status=%q err=%v", ret.Status, err)
	}
	ret, err = svc.ApplyRetconRequest(context.Background(), ret.ID, "admin-3")
	if err != nil || ret.Status != "APPLIED" {
		t.Fatalf("ApplyRetconRequest: status=%q err=%v", ret.Status, err)
	}
}

func TestAnalyzeRetconHTTPReturnsConflictForInvalidTransition(t *testing.T) {
	store := newFakeStore()
	store.retcons["story-1"] = []RetconRequest{{ID: "retcon-1", StoryID: "story-1", Status: "APPLIED"}}
	handler := NewHandler(NewService(store))
	req := httptest.NewRequest(http.MethodPost, "/admin/retcons/retcon-1/analyze", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RETCON_INVALID_TRANSITION") {
		t.Fatalf("body = %s, want RETCON_INVALID_TRANSITION", rec.Body.String())
	}
}

func TestRetconMutationErrorMapsInvalidTransitionToConflict(t *testing.T) {
	rec := httptest.NewRecorder()

	writeRetconMutationError(rec, ErrRetconInvalidTransition)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "RETCON_INVALID_TRANSITION") {
		t.Fatalf("body = %s, want RETCON_INVALID_TRANSITION", rec.Body.String())
	}
}
