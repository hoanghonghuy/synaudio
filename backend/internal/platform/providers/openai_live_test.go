package providers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/planning"
)

func TestOpenAILiveEndpoint(t *testing.T) {
	if os.Getenv("LIVE_OPENAI_TEST") == "" {
		t.Skip("skipping live endpoint test; set LIVE_OPENAI_TEST=1 to run")
	}

	client, err := newOpenAIClient(
		"https://yuhh-9router.duckdns.org/v1",
		"sk-d21f2272f14871ca-gizpjq-e919334a",
		"code",
	)
	if err != nil {
		t.Fatalf("newOpenAIClient: %v", err)
	}

	adapter := &openAIAI{client: client}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. Test GenerateText
	out, err := adapter.GenerateText(ctx, generation.TextAIInput{
		Prompt: "Xin chào! Hãy giới thiệu ngắn gọn trong 1 câu bạn là ai.",
	})
	if err != nil {
		t.Fatalf("GenerateText failed: %v", err)
	}
	t.Logf("Live GenerateText response: %s", out.Text)
	if out.Text == "" {
		t.Fatal("expected non-empty generated text")
	}

	// 2. Test ProposeFoundation
	proposal, err := adapter.ProposeFoundation(ctx, planning.FoundationInput{
		StoryID: "story-live-test",
		Premise: "Một thợ sửa đồng hồ phát hiện cỗ máy có khả năng đảo ngược 1 phút thời gian.",
	})
	if err != nil {
		t.Fatalf("ProposeFoundation failed: %v", err)
	}
	t.Logf("Live ProposeFoundation Bible: %+v", proposal.Bible)
	t.Logf("Live ProposeFoundation Ending: %+v", proposal.Ending)
	t.Logf("Live ProposeFoundation Arcs count: %d", len(proposal.Arcs))
	t.Logf("Live ProposeFoundation Characters count: %d", len(proposal.Characters))
	if len(proposal.Arcs) == 0 {
		t.Fatal("expected at least one arc")
	}
	if len(proposal.Characters) == 0 {
		t.Fatal("expected at least one character")
	}
}
