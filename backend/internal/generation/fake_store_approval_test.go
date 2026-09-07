package generation

import (
	"context"
	"errors"
)

func (s *fakeStore) ApproveContentRevision(_ context.Context, a ContentApproval) (ContentApproval, error) {
	for chapterID, revisions := range s.revisions {
		for i, revision := range revisions {
			if revision.ID != a.ContentRevisionID {
				continue
			}
			if chapterID != a.ChapterID || revision.ChapterID != a.ChapterID {
				return ContentApproval{}, errors.Join(ErrContentRevisionNotFound, ErrContentRevisionChapterMismatch)
			}
			if revision.Status != "CANDIDATE" && revision.Status != "APPROVED" {
				return ContentApproval{}, errors.Join(ErrContentRevisionNotFound, ErrContentRevisionNotApprovable)
			}
			for _, existing := range s.approvals[a.ChapterID] {
				if existing.ContentRevisionID == a.ContentRevisionID {
					revision.Status = "APPROVED"
					s.revisions[chapterID][i] = revision
					return existing, nil
				}
			}

			revision.Status = "APPROVED"
			s.revisions[chapterID][i] = revision
			s.approvals[a.ChapterID] = append(s.approvals[a.ChapterID], a)
			return a, nil
		}
	}
	return ContentApproval{}, ErrContentRevisionNotFound
}
