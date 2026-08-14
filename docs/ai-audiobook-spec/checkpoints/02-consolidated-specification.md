# AI Audiobook Platform

## Consolidated Specification #2

### Story Lifecycle, Chapter Lifecycle, Generation Orchestration, Creative Decisions, Canon Revision, Approval & Publishing

**Document status:** Finalized Business & Domain Decisions
**Depends on:** Consolidated Specification #1
**Purpose:** Làm source of truth cho các nghiệp vụ và invariant đã được chốt trước khi thiết kế physical database schema, API contract và implementation architecture.

---

# 1. Scope của tài liệu

Tài liệu này chốt các nhóm nghiệp vụ:

```text
Story Lifecycle

Chapter Lifecycle

Artifact Revision / Versioning

GenerationRun / GenerationJob Lifecycle

Retry / Failure / Cancellation / Staleness

Batch Generation

Creative Decision Lifecycle

Retcon / Canon Revision

Canon Data Repair

Content Approval

Narration / Audio Approval

Quality Gates

READY / Publish / Unpublish

Story Visibility

Story Completion
```

Tài liệu này **chưa chốt**:

```text
Physical database schema

Exact PostgreSQL column types

Exact foreign keys

Indexes

Enum implementation

REST request/response schema

Authentication implementation

Detailed RBAC matrix

Queue technology

Worker claiming algorithm

Design pattern

Exact Go package structure

CI/CD

Testing implementation

Exact UI layout
```

---

# 2. Cross-domain terminology

Các khái niệm sau phải luôn được giữ riêng.

```text
Story Status
≠
Story Visibility
≠
Story Planning Phase
```

```text
Chapter Status
≠
Generation Job Status
≠
Canon Status
≠
Audio Status
```

```text
Official Canon
≠
Published Content
```

Một Chapter có thể đã nằm trong:

```text
Official Canon
```

nhưng chưa:

```text
PUBLISHED
```

Tương tự:

```text
PUBLISHED
```

không phải Canon status.

---

# 3. Story Status

Story sử dụng:

```text
DRAFT
ACTIVE
COMPLETED
ARCHIVED
```

---

# 4. Story DRAFT

`DRAFT` là Story đang trong giai đoạn thiết lập.

Có thể đang thực hiện:

```text
Story Architect

Story Bible generation/edit

Character creation

Ending Plan

Initial Arc planning

Workflow configuration
```

Admin có thể:

```text
edit metadata

edit Story Bible

edit Characters

edit initial Arcs

edit Ending Plan

generate planning artifacts

change mutable Workflow Settings
```

Story Generation Policy đã được snapshot từ lúc Story được tạo và vẫn immutable.

Thông thường:

```text
Story.status = DRAFT
Story.visibility = PRIVATE
```

---

# 5. Story Activation

Transition:

```text
DRAFT
  ↓
Activation Gate
  ↓
ACTIVE
```

Không được activate chỉ vì Admin bấm nút.

Activation Gate tối thiểu phải xác nhận:

```text
Story Generation Policy exists

Current Story Bible exists

Current Ending Plan exists

At least one valid initial Story Arc exists

Required main Characters exist

Planning Mode is configured
```

Nếu thiếu:

```text
Activation rejected
```

và hệ thống phải cho biết dependency còn thiếu.

Ví dụ:

```text
Cannot activate Story.

Missing:
- Current Ending Plan
- Initial Story Arc
```

Không cần Chapter hoặc Audio để activate.

---

# 6. Story ACTIVE

`ACTIVE` nghĩa là Story đã đủ nền tảng để vận hành Story Engine và sản xuất Chapter.

`ACTIVE` không đồng nghĩa với Public.

Ví dụ hợp lệ:

```text
status = ACTIVE
visibility = PRIVATE

Chapter 1 = PRODUCTION
Chapter 2 = DRAFT
```

Story có thể được sản xuất nội bộ khá lâu trước khi public.

---

# 7. Story COMPLETED

Story chuyển sang:

```text
COMPLETED
```

khi câu chuyện đã thật sự kết thúc.

Thông thường:

```text
planningPhase = COMPLETED
status        = COMPLETED
```

Sau khi completed:

```text
Generate Next Chapter
```

bị disable trong normal workflow.

Tương tự:

```text
Create Normal Next Arc
```

bị disable.

Story vẫn có thể:

```text
remain PUBLIC

be listened to

be read

receive non-canonical metadata corrections

receive audio repairs
```

Nếu muốn viết tiếp một Story đã kết thúc:

```text
Create Sequel
```

được ưu tiên hơn reopen Story.

---

# 8. Story ARCHIVED

`ARCHIVED` là operation quản trị.

Nó không có nghĩa:

```text
deleted
```

Có thể:

```text
DRAFT → ARCHIVED
ACTIVE → ARCHIVED
COMPLETED → ARCHIVED
```

Archived Story không được normal:

```text
generate new chapters

create new arcs

modify Canon

publish new content
```

---

# 9. Archived Story có thể vẫn Public

Hai trạng thái này độc lập.

Ví dụ:

```text
status     = ARCHIVED
visibility = PUBLIC
```

có nghĩa:

> Story không còn được tiếp tục sản xuất nhưng listener vẫn có thể nghe nội dung cũ.

Hoặc:

```text
status     = ARCHIVED
visibility = PRIVATE
```

để rút khỏi catalog.

---

# 10. Restore Archived Story

Hệ thống nên support:

```text
ARCHIVED
   ↓
Restore
```

Restore phải quay lại operational state hợp lệ trước archive.

Ví dụ:

```text
COMPLETED
→ ARCHIVED
→ Restore
→ COMPLETED
```

không tự trở thành:

```text
ACTIVE
```

Business concept cần biết trạng thái trước archive.

Exact persistence mechanism sẽ quyết định sau.

---

# 11. Story Visibility

Story Visibility baseline:

```text
PRIVATE
PUBLIC
```

Không cần `UNLISTED` trong V1.

---

# 12. PRIVATE

Story PRIVATE không xuất hiện qua public listener catalog/API.

Admin vẫn có thể:

```text
generate chapters

publish chapters internally

review

prepare launch
```

---

# 13. PUBLIC

Story PUBLIC xuất hiện cho Guest/User.

Nhưng listener chỉ nhìn thấy Chapter có:

```text
status = PUBLISHED
```

---

# 14. Story Public Gate

Transition:

```text
PRIVATE
   ↓
Public Validation
   ↓
PUBLIC
```

Tối thiểu phải có:

```text
Story status = ACTIVE or COMPLETED

at least one PUBLISHED Chapter

valid public title

valid description

valid cover

at least one Genre

no platform-level blocking issue
```

Nếu không:

```text
Cannot make Story PUBLIC
```

và phải trả rõ dependency thiếu.

---

# 15. Published Chapter không tự launch Story

Một Story có thể:

```text
visibility = PRIVATE

Chapter 1 = PUBLISHED
Chapter 2 = PUBLISHED
```

Listener vẫn không thấy.

Publish Chapter 1 và:

```text
Make Story Public
```

là hai action riêng.

Không auto-launch Story chỉ vì Chapter đầu tiên được publish.

---

# 16. Make Story Private Again

Admin có thể:

```text
PUBLIC
 ↓
PRIVATE
```

mà không cần unpublish từng Chapter.

Chapter vẫn:

```text
PUBLISHED
```

nhưng bị hidden bởi Story visibility.

Khi Story trở lại PUBLIC, chúng xuất hiện lại.

Đây là cách ưu tiên nếu cần tạm gỡ toàn bộ series khỏi listener.

---

# 17. Story Delete

Normal operation ưu tiên:

```text
ARCHIVE
```

thay vì Delete.

Một Story mới có thể được phép delete/soft-delete nếu:

```text
DRAFT

no Official Canon

no Published Chapters

no meaningful listening data

no important dependent history
```

Một Story đã có production history như:

```text
Official Canon

Published Chapters

Listening Progress

Audit History
```

không được normal Delete qua content-management UI.

Future physical purge/data deletion phải là workflow khác.

---

# 18. Story Lifecycle tổng thể

```text
                DRAFT
                  │
           Activation Gate
                  │
                  ▼
                ACTIVE
                  │
           Story Completion
                  │
                  ▼
              COMPLETED


DRAFT / ACTIVE / COMPLETED
             │
           Archive
             │
             ▼
          ARCHIVED
             │
           Restore
             │
             ▼
     previous valid state
```

Visibility độc lập:

```text
PRIVATE ↔ PUBLIC
```

---

# 19. Story Planning Phase

Story Planning Phase tiếp tục sử dụng:

```text
ONGOING

CLOSING

FINAL_ARC

COMPLETED
```

Planning Phase và Story Status không được gom vào cùng một field logic.

---

# 20. Chapter Status

Chapter lifecycle đã được chuẩn hóa thành:

```text
DRAFT

CONTENT_REVIEW

PRODUCTION

READY

PUBLISHED

ARCHIVED
```

---

# 21. Vì sao không dùng Job Status làm Chapter Status

Không thêm Chapter status kiểu:

```text
WRITING

AI_REVIEWING

TTS_GENERATING

TTS_FAILED

FFMPEG_RUNNING
```

Các state đó thuộc:

```text
GenerationRun
GenerationJob
AudioAsset
```

Chapter Status chỉ biểu diễn business lifecycle của Chapter.

---

# 22. Chapter DRAFT

`DRAFT` nghĩa là content chưa được accept vào canonical production flow.

Trong DRAFT:

```text
Chapter Plan có thể thay đổi

Content có thể regenerate

Content có thể rewrite

Admin có thể edit

Title có thể đổi

Scene structure có thể đổi
```

Draft không được mutate Official Canon.

---

# 23. Chapter CONTENT_REVIEW

Chapter chuyển:

```text
DRAFT
 ↓
CONTENT_REVIEW
```

khi candidate content đã đủ điều kiện để human review.

Thông thường pipeline trước đó đã chạy:

```text
Story Writer

Continuity Review

Quality Review

Rewrite
```

Admin có thể:

```text
Approve

Edit

Rewrite with feedback

Regenerate

Return to planning

Reject candidate
```

---

# 24. CONTENT_REVIEW semantic

Để tránh cần thêm một status trung gian vô nghĩa, `CONTENT_REVIEW` được hiểu là:

> Chapter đang ở pre-Canon stage; current content chưa hoàn tất successful Canon Commit.

Thông thường ban đầu nó đang chờ Admin.

Nếu Admin đã approve nhưng:

```text
Memory Extraction

Canon Validation

Canon Commit
```

chưa hoàn tất hoặc đang retry, Chapter vẫn có thể ở `CONTENT_REVIEW`.

Human approval là một revision-level event riêng, không phải Chapter Status.

Sau Canon Commit thành công mới:

```text
PRODUCTION
```

---

# 25. Admin Edit trong CONTENT_REVIEW

AI Review phải bind vào Content Revision.

Ví dụ:

```text
Content Revision 5
↓
Continuity Review
↓
Quality Review
```

Admin sửa:

```text
Revision 5
→ Revision 6
```

Review của Revision 5 không còn được xem là current report của Revision 6.

Không nhất thiết delete report cũ.

Nó vẫn tồn tại để audit/debug.

---

# 26. Content Revision

AI-generated content quan trọng không nên overwrite mất history.

Business concept:

```text
Chapter Content Revision 1

Chapter Content Revision 2

Chapter Content Revision 3
```

Một revision có thể là:

```text
candidate

current candidate

approved

stale

historical
```

Exact schema sau.

---

# 27. Content Approval

Admin approval luôn bind vào exact revision.

Ví dụ:

```text
Chapter 25
Content Revision 7
APPROVED
```

Không phải:

```text
Chapter 25 approved forever
```

Nếu revision đổi:

```text
7 → 8
```

approval của 7 không tự áp dụng cho 8.

---

# 28. CONTENT_REVIEW → DRAFT

Nếu Admin:

```text
rejects candidate

requests full regeneration

returns to Chapter Plan
```

thì:

```text
CONTENT_REVIEW
      ↓
     DRAFT
```

Official Canon vẫn không đổi.

---

# 29. Content Approval không phải Publish

Flow:

```text
Content Approval
       ↓
Memory Extraction
       ↓
Canon Change Set
       ↓
Canon Validation
       ↓
Canon Commit
```

Content Approval chỉ nói:

> Human chấp nhận prose/content revision này.

---

# 30. Memory Extraction sau Approval

Rule cũ được giữ tuyệt đối:

```text
Draft
 ↓
AI Review
 ↓
Admin Edit
 ↓
Final Content Approval
 ↓
Memory Extraction
```

Không extract canonical memory từ draft chưa chốt.

---

# 31. Canon Commit Gate

Trước Official Canon Commit phải thỏa:

```text
approved Content Revision exists

Memory Extraction succeeded

Canon Change Set passes validation

base Canon version still valid

no unresolved blocking Creative Decision

previous required Chapter Official Canon exists

Canon sequence remains contiguous

no hard Canon conflict
```

---

# 32. Canon thay đổi sau Content Approval

Nếu Content Revision được approve với:

```text
base Canon = v50
```

nhưng trước khi commit current Canon đã thành:

```text
v51
```

không blindly commit.

Phải:

```text
revalidate
```

và nếu không còn valid:

```text
STALE
```

hoặc quay lại review/planning phù hợp.

---

# 33. Chapter PRODUCTION

Khi Official Canon Commit thành công:

```text
CONTENT_REVIEW
      ↓
Official Canon Commit
      ↓
PRODUCTION
```

`PRODUCTION` nghĩa:

> Canonical content đã được chốt; các publishing artifacts đang được sản xuất.

Bao gồm:

```text
Narration Script

Narration Review

TTS

Audio Processing

Audio Quality
```

---

# 34. Failure trong PRODUCTION

Ví dụ:

```text
TTS Job FAILED
```

Chapter vẫn:

```text
PRODUCTION
```

Job mang failure state.

Không đổi Chapter thành:

```text
TTS_FAILED
```

Admin retry job hoặc thay configuration được phép.

---

# 35. Chapter READY

`READY` có semantic mạnh:

> Chapter hiện tại publishable.

Điều kiện:

```text
current Content Revision approved

Official Canon committed

current Narration Revision valid

active AudioAsset READY

all HARD publication gates pass

all OVERRIDABLE violations either resolved or explicitly overridden
```

---

# 36. READY không phải Public

Transition cuối:

```text
READY
 ↓
Admin Publish
 ↓
PUBLISHED
```

V1 không auto-publish.

---

# 37. Chapter PUBLISHED

`PUBLISHED` nghĩa:

> Chapter được publication system cho phép public.

Listener thực sự nhìn thấy Chapter khi đồng thời:

```text
Chapter = PUBLISHED
Story   = PUBLIC
```

---

# 38. Chapter ARCHIVED

Chapter Archive là rare administrative operation.

Không dùng như normal way để sửa Story.

Archive middle Chapter trong sequential Story có thể phá public sequence và phải có impact warning.

---

# 39. Chapter numbering

Normal Chapter creation:

```text
next chapter number
=
max canonical/active sequence + 1
```

Không cho normal workflow tạo:

```text
Chapter 999
```

tùy tiện giữa Story.

---

# 40. Chapter Gap Rule

Không cho canonical sequence:

```text
18
19
21
```

Draft/future planning có thể reorder trước khi Canon Commit.

Sau Official:

```text
chapter_number
```

được xem là immutable business identity.

---

# 41. Canon Sequential Rule

Normal Story flow yêu cầu:

```text
Chapter N
```

không được trở thành Official Canon nếu required previous Chapter chưa Official.

Official sequence phải contiguous.

---

# 42. Publish Sequential Rule

```text
Chapter N
```

chỉ được PUBLISHED nếu:

```text
Chapter N-1 = PUBLISHED
```

Ngoại lệ:

```text
Chapter 1
```

Không cho public sequence bị gap.

---

# 43. Chapter Plan có thể tồn tại trước Chapter Canon

Planning Horizon vẫn có thể có:

```text
Plan 21

Plan 22

Plan 23
```

trước khi Chapter 20 Official.

`ChapterPlan` không đồng nghĩa với:

```text
Official Chapter
```

---

# 44. Artifact Versioning Principle

Các AI artifact quan trọng phải có revision/version history về business concept.

Bao gồm:

```text
Story Bible

Ending Plan

Chapter Plan

Chapter Content

Narration Script

AudioAsset

Canon
```

Không overwrite historical evidence một cách mất dấu.

---

# 45. Derived Artifact Lineage

Một artifact phải biết nó derive từ version/revision nào.

Ví dụ:

```text
Content Revision 7
       ↓
Narration Revision 3
       ↓
Audio Version 2
```

Nếu upstream thay đổi, downstream phải được re-evaluate/stale.

---

# 46. Pre-publish Content Revision

Nếu Chapter đã:

```text
READY
```

nhưng chưa Published, Admin vẫn có thể phát hiện cần sửa prose.

Không normal Edit trực tiếp.

Dùng controlled:

```text
Revise Before Publish
```

Flow có thể:

```text
READY
 ↓
new Content Revision
 ↓
review
 ↓
Canon reconciliation
 ↓
new Narration
 ↓
new Audio
 ↓
READY
```

Nếu downstream Chapters đã generate từ old Canon:

```text
mark affected downstream STALE
```

---

# 47. Published Story Content

Nếu Chapter đã PUBLISHED, canonical prose không được normal edit.

Muốn đổi:

```text
Request Canon Revision / Retcon
```

---

# 48. Narration Repair không phải Retcon

Nếu Story Content không đổi nhưng:

```text
pronunciation wrong

pause wrong

tone wrong
```

Admin có thể:

```text
new Narration Revision
      ↓
new Audio Version
```

không thay Canon.

---

# 49. Audio Repair không phải Retcon

Ví dụ:

```text
noise

volume issue

provider artifact

encoding issue
```

Flow:

```text
Audio v1 active
      ↓
Generate Audio v2
      ↓
Validate
      ↓
Promote v2
```

Chapter có thể vẫn `PUBLISHED`.

---

# 50. Published Audio Promotion

Logical swap phải không tạo khoảng trống.

```text
v1 active
   ↓
logical atomic promotion
   ↓
v2 active
```

Listener không được thấy trạng thái:

```text
no active audio
```

do swap nửa chừng.

Exact DB transaction sau.

---

# 51. GenerationRun

`GenerationRun` là một workflow instance cấp business.

Ví dụ:

```text
GenerationRun #1008

Intent:
CHAPTER_GENERATION

Story:
Story #12

Chapter:
Chapter #25

Base Canon:
v24

Started by:
Admin A
```

---

# 52. GenerationRun và GenerationJob

Một Run chứa nhiều Jobs.

```text
GenerationRun
├── BUILD_CONTEXT
├── GENERATE_CONTENT
├── ANALYZE_DURATION
├── REVIEW_CONTINUITY
├── REVIEW_QUALITY
└── REWRITE
```

Admin/UI nhìn Run như một workflow.

Worker xử lý Jobs.

---

# 53. GenerationRun Types

Target business vocabulary có thể gồm:

```text
STORY_ARCHITECTURE

ARC_PLANNING

CHAPTER_PLANNING

CHAPTER_GENERATION

CHAPTER_REVIEW

CHAPTER_REWRITE

MEMORY_EXTRACTION

NARRATION_GENERATION

AUDIO_GENERATION

RETCON_ANALYSIS

ARC_COMPLETION_REVIEW

STORY_COMPLETION_REVIEW
```

Không phải tất cả cần implement ngay.

---

# 54. GenerationRun Status

```text
PENDING

RUNNING

WAITING

SUCCEEDED

FAILED

CANCELLED

STALE
```

---

# 55. Run PENDING

Workflow instance được tạo nhưng chưa bắt đầu execution.

---

# 56. Run RUNNING

Run đang có execution activity.

---

# 57. Run WAITING

Run không lỗi.

Nó đang chờ dependency/business decision.

Ví dụ:

```text
WAITING_FOR_ADMIN_REVIEW

WAITING_FOR_CREATIVE_DECISION

WAITING_FOR_DURATION_OVERRIDE

WAITING_FOR_ARC_DECISION

WAITING_FOR_DEPENDENCY

WAITING_FOR_PROVIDER_RECOVERY
```

Waiting Reason tách khỏi status.

---

# 58. Human WAITING không timeout như technical failure

Nếu Admin để Chapter:

```text
WAITING_FOR_ADMIN
```

hai tuần, không biến thành:

```text
FAILED
```

Human business gate và technical timeout là hai khái niệm khác nhau.

---

# 59. Run SUCCEEDED

Workflow business intent đã hoàn tất.

---

# 60. Run FAILED

Workflow không thể hoàn thành sau retry/recovery policy.

---

# 61. Run CANCELLED

User/System chủ động cancel workflow.

---

# 62. Run STALE

Workflow/result được tạo dựa trên input/version không còn current.

Không được auto-apply.

---

# 63. GenerationJob

Job là execution unit nhỏ.

Ví dụ:

```text
REVIEW_CONTINUITY

GENERATE_TTS_SEGMENT

BUILD_CONTEXT

MEMORY_EXTRACTION
```

---

# 64. GenerationJob Status

```text
PENDING

RUNNING

SUCCEEDED

FAILED

CANCELLED

STALE
```

Không cần permanent `RETRYING` status.

Retry được biểu diễn bằng attempts/execution history.

---

# 65. JobAttempt Concept

Một Job có thể có nhiều attempts.

```text
Job #881

Attempt 1 → timeout
Attempt 2 → provider 503
Attempt 3 → success
```

Final:

```text
Job = SUCCEEDED
```

Attempt history vẫn tồn tại để:

```text
debug

provider health

cost analysis

reliability analysis
```

Exact persistence sau.

---

# 66. Failure Classification

Failure được phân loại:

```text
TRANSIENT

PERMANENT

VALIDATION

STALE_INPUT

POLICY_BLOCK
```

---

# 67. TRANSIENT

Ví dụ:

```text
timeout

HTTP 429

HTTP 503

temporary network issue
```

Có thể automatic retry.

---

# 68. PERMANENT

Ví dụ:

```text
invalid credential

unsupported model

invalid provider config
```

Retry giống hệt không có giá trị.

Fail sớm và yêu cầu config/action.

---

# 69. VALIDATION Failure

AI trả output không đúng contract.

Ví dụ:

```text
required structured JSON
```

nhưng AI trả malformed data.

Có thể:

```text
Repair Pass
```

hoặc limited regenerate.

---

# 70. STALE_INPUT Failure

Input đã đổi.

Ví dụ:

```text
Job generated against Revision 5

Current Revision = 6
```

Không retry same operation như transient error.

Mark stale.

---

# 71. POLICY_BLOCK

Business rules không cho operation tiếp tục.

Ví dụ:

```text
Trying to Canon Commit Chapter 31
while required previous Canon only reaches Chapter 29.
```

Đây không phải infrastructure error.

---

# 72. Retry Principle

```text
TRANSIENT
→ automatic retry

VALIDATION
→ repair / limited retry

PERMANENT
→ fail

STALE_INPUT
→ stale

POLICY_BLOCK
→ block with explanation
```

Không retry mù.

---

# 73. Retry Budget

Retry luôn finite.

Concept example:

```text
AI generation:
limited attempts

schema repair:
limited attempts

TTS segment:
limited attempts

provider transient:
limited attempts
```

Exact numbers configurable/implementation-level sau.

Invariant:

```text
No infinite retry.
```

---

# 74. Retry Backoff

Transient provider failure không retry tức thời vô hạn.

Concept:

```text
Attempt 1
 ↓
delay
 ↓
Attempt 2
 ↓
longer delay
 ↓
Attempt 3
```

Exact backoff algorithm sau.

---

# 75. HTTP 429

Rate limit được coi là transient khi appropriate.

Scheduler/executor nên tôn trọng provider retry hints như `Retry-After` nếu tồn tại.

---

# 76. Retry khác Regenerate

### Retry

```text
same logical intended result

failure recovery
```

Ví dụ:

```text
provider timeout
→ Retry
```

### Regenerate

```text
create a new candidate artifact
```

Ví dụ:

```text
Chapter quality unsatisfactory
→ Regenerate
```

Hai operation không được dùng lẫn terminology.

---

# 77. Rewrite khác Regenerate

### Rewrite

Sửa candidate hiện có dựa trên:

```text
review issues

admin feedback
```

### Regenerate

Sinh candidate mới rộng hơn từ plan/context.

UI/API sau này phải phân biệt rõ.

---

# 78. AI Output Validation

Không:

```text
AI Response
→ Database Apply
```

Phải:

```text
AI Response
      ↓
Parse
      ↓
Schema Validation
      ↓
Semantic Validation
      ↓
Business Validation
      ↓
Accept / Reject
```

---

# 79. Memory Extractor Validation

Ví dụ AI muốn update Character.

Backend kiểm tra:

```text
Character exists?

Character belongs to Story?

Base Canon still current?

Fact references valid?

Status valid?

Business invariant violated?

World Rule violated?
```

AI output không phải trusted command.

---

# 80. Repair Pass

Malformed structured output có thể được repair trước khi expensive regeneration.

```text
Invalid Output
 ↓
Repair Attempt
 ↓
Validate
```

Nếu vẫn fail:

```text
limited regenerate
```

Sau retry budget:

```text
FAILED
```

---

# 81. Provider Fallback

Không fallback provider/model ngẫu nhiên.

### Utility operations

Có thể fallback tự động hơn:

```text
format repair

simple extraction

embedding
```

### Quality-sensitive operations

Như:

```text
Story Writer

Narration

TTS Voice
```

fallback phải tuân theo configured/approved provider policy.

---

# 82. Preferred Provider vs Actual Provider

Story Workflow Settings có thể nói:

```text
preferredStoryModel = X
```

Nhưng GenerationJob phải trace:

```text
actualProvider

actualModel
```

---

# 83. Idempotency

Admin double-click:

```text
Generate Chapter 25
Generate Chapter 25
```

không được silently tạo hai competing Runs nếu business policy không cho.

System phải detect compatible active intent.

---

# 84. Duplicate Generation

Nếu Chapter đang có:

```text
CHAPTER_GENERATION RUNNING
```

Admin bấm Generate Again phải nhận explicit choice/response.

Ví dụ:

```text
current generation is running
```

không silently tạo concurrency conflict.

---

# 85. Business Generation Lock

Các incompatible operations trên cùng resource không được chạy cạnh tranh.

Ví dụ hai:

```text
GENERATE_CHAPTER_CONTENT
```

cho cùng current Chapter Revision.

Logical lock có thể dựa trên:

```text
Story / Chapter / Operation Type
```

Exact technical lock sau.

---

# 86. Parallel Generation

Có dependency thì sequential.

Ví dụ:

```text
Chapter 101
→ Canon 101
→ Chapter 102
```

Có thể parallel khi independent và input frozen.

Ví dụ:

```text
TTS Segment 1
TTS Segment 2
TTS Segment 3
```

Hoặc:

```text
Continuity Review
Quality Review
```

nếu cả hai đọc cùng Content Revision/Canon snapshot.

---

# 87. Generation Workflow Dependency

GenerationRun conceptually có dependency graph.

Ví dụ:

```text
             BUILD_CONTEXT
                   │
                   ▼
            GENERATE_CONTENT
                   │
          ┌────────┴────────┐
          ▼                 ▼
CONTINUITY_REVIEW     QUALITY_REVIEW
          │                 │
          └────────┬────────┘
                   ▼
                REWRITE
                   │
                   ▼
             HUMAN REVIEW
```

V1 không bắt buộc generic DAG engine.

---

# 88. Human Gate

Human Review không phải Job.

Nó là:

```text
Business Gate
```

Run chuyển:

```text
WAITING
```

---

# 89. Cancellation

Admin có thể cancel generation chưa hoàn tất.

System phải:

```text
stop scheduling future jobs

request cancellation of interruptible running work

ignore late result if it cannot be interrupted
```

Run:

```text
CANCELLED
```

---

# 90. Late Result after Cancellation

Provider call có thể không hỗ trợ cancel.

Nếu result về sau Run đã CANCELLED:

```text
do not apply
```

Có thể giữ result cho diagnostics/history.

---

# 91. Cancel trước Canon Commit

Official Canon không đổi.

Partial candidate output không tự apply.

---

# 92. Cancel sau Canon Commit

Không rollback Canon.

Ví dụ:

```text
Content Approved
Canon Committed
Narration generating
```

Admin cancel audio generation.

Result:

```text
Canon remains Official

Chapter remains PRODUCTION

Audio workflow cancelled
```

Admin có thể resume/regenerate audio sau.

---

# 93. Stale Input Binding

Important generation phải bind với input context như:

```text
Content Revision

Canon Version

Story Bible Version

Ending Plan Version

Arc Version

Prompt Version

Workflow Version
```

---

# 94. Stale Result

Ví dụ:

```text
Job starts:
Content Revision 5

Admin edits:
Revision 6

Job completes
```

Result:

```text
STALE
```

Không overwrite Revision 6.

---

# 95. Preserve Stale Output

Stale result có thể được giữ:

```text
Generated against Revision 5

Current Revision 6
```

để:

```text
audit

debug

compare

manual reuse
```

Nhưng không auto-apply.

---

# 96. Canon Staleness

Nếu job/review dùng:

```text
Canon v49
```

và Canon đã đổi thành:

```text
v50
```

baseline safe behavior:

```text
do not trust automatically
```

Revalidate hoặc mark stale.

---

# 97. Upstream Invalidation

Nếu upstream artifact đổi, downstream có thể stale.

Ví dụ:

```text
Chapter Plan v3
       ↓
Content v7
       ↓
Narration v2
       ↓
Audio v2
```

Regenerate Content thành v8:

```text
Narration v2 = STALE

Audio v2 = STALE
```

---

# 98. Generation không tự promote Candidate

AI sinh:

```text
Content Revision 8 candidate
```

không tự thành current/approved nếu workflow yêu cầu human review.

Admin có thể:

```text
Accept

Reject

Compare
```

---

# 99. Regenerate with Feedback

Admin có thể cung cấp instruction:

```text
Đoạn giữa quá dài.
Không đổi ending.
Tăng tension scene 3.
```

Generation input gồm:

```text
current candidate

reviewer issues

admin feedback

current Canon
```

---

# 100. Admin Feedback không bypass Canon

Nếu Admin feedback yêu cầu:

```text
Minh dùng súng
```

nhưng Canon nói Minh không có súng:

```text
Canon Conflict
```

Không silently comply.

Nếu cần thay Canon:

```text
Creative Decision
```

hoặc appropriate Canon Revision workflow.

---

# 101. Batch Generation

Batch size là mutable Story Workflow Setting.

Ví dụ:

```text
1
3
5
10
Custom
```

---

# 102. Batch Generation luôn sequential theo Canon dependency

```text
Official Canon v100
       ↓
Chapter 101
       ↓
Provisional v101
       ↓
Chapter 102
       ↓
Provisional v102
```

Không parallel Chapter 101–105 như independent jobs.

---

# 103. Provisional Canon

Provisional Canon chỉ dùng để tiếp tục generation trong staged batch.

Nó không phải public truth.

Listener/public publication chỉ dựa vào Official Canon.

---

# 104. Batch Failure

Ví dụ:

```text
Ch101 success

Ch102 success

Ch103 FAILED
```

Không tiếp tục Ch104/105 nếu chúng phụ thuộc Ch103.

Batch dừng/block tại Ch103.

Admin có thể:

```text
Retry Chapter 103

Edit Chapter 103

Regenerate

Cancel remaining batch
```

---

# 105. Batch không phải giant transaction

Không rollback Ch101/102 chỉ vì Ch103 fail.

Valid upstream results được giữ.

---

# 106. Batch Approval Order

Admin có thể review candidate theo UX khác nhau nhưng Official Canon Commit phải sequential.

```text
101
→ 102
→ 103
```

Không Official Chapter 103 trước 101/102.

---

# 107. Upstream Batch Edit

Nếu Chapter 101 thay đổi:

```text
Chapter 102

Chapter 103
```

được đánh giá dependency và thường:

```text
STALE
```

---

# 108. Generation Resume

Run WAITING có thể resume từ valid point.

Không regenerate mọi upstream stage nếu output vẫn valid.

Ví dụ:

```text
Plan = valid

Writer failed
```

retry Writer.

Không regenerate Plan.

---

# 109. Partial TTS Success

Ví dụ:

```text
Segment 1 ✓
Segment 2 ✓
Segment 3 ✗
Segment 4 ✓
```

Không regenerate successful segments.

Retry Segment 3.

Chỉ khi tất cả required segments successful mới tiếp tục FFmpeg join.

---

# 110. FFmpeg Failure

Nếu TTS hoàn tất nhưng FFmpeg fail:

```text
retry FFmpeg
```

không gọi lại TTS.

---

# 111. Upload Failure

Nếu final MP3 đã xử lý nhưng object storage upload fail:

```text
retry upload
```

không regenerate TTS/FFmpeg nếu artifact vẫn còn usable.

---

# 112. Technical Timeout

Technical job không được:

```text
RUNNING forever
```

Provider calls/tasks cần technical timeout.

Exact timeout sau.

---

# 113. Orphan Job Recovery

Worker chết giữa Job không được khiến Job vĩnh viễn RUNNING.

System phải có recovery semantics.

Cơ chế:

```text
lease
heartbeat
DB ownership
```

sẽ quyết khi implementation.

---

# 114. Persistent Generation State

Application restart không được làm mất workflow.

GenerationRun/Job state phải nằm trong persistent infrastructure.

Không dựa hoàn toàn vào in-memory goroutine.

---

# 115. PostgreSQL-backed Jobs ban đầu

Initial implementation direction:

```text
PostgreSQL-backed GenerationRun / GenerationJob
```

Sau này executor/queue có thể đổi mà business workflow không đổi.

---

# 116. Workflow Definition vs Workflow Instance

Concept:

```text
Workflow Definition:
CHAPTER_GENERATION_V1
```

Instance:

```text
GenerationRun #1008
```

---

# 117. Workflow Versioning

Pipeline có thể evolve:

```text
CHAPTER_GENERATION_V1

CHAPTER_GENERATION_V2
```

Old Runs vẫn trace được workflow version thật đã sử dụng.

---

# 118. Prompt / Model Binding

Run/Job phải trace:

```text
prompt version

provider

model

workflow version

input versions
```

Historical output không rewrite metadata theo config hiện tại.

---

# 119. Manual Admin Instruction Binding

Admin instruction phải thuộc Run input/context snapshot.

Ví dụ:

```text
"Giảm horror, tăng conflict Minh/Lan."
```

Sau này phải biết output bị ảnh hưởng bởi instruction này.

---

# 120. Cost / Usage Traceability

Target observability cần khả năng aggregate:

```text
AI input usage

AI output usage

provider calls

TTS usage

duration

estimated cost
```

theo Run/Chapter.

Không bắt buộc V1 UI hiển thị đầy đủ.

---

# 121. Provider Call Trace

Safe metadata:

```text
provider

model

operation

start

finish

latency

status

usage

generationRun

job
```

Không log secret.

---

# 122. Generation Priority Concept

Target có thể có:

```text
INTERACTIVE

NORMAL

BACKGROUND
```

Exact queue implementation sau.

Public listener workload phải logical-isolated khỏi heavy generation workload.

---

# 123. Generation Concurrency

Platform cần conceptual limits:

```text
per provider

per Story

per Job Type

global
```

Không để một action tạo hàng trăm uncontrolled provider calls.

---

# 124. Story-level Sequential Constraint

Chapter sequence trong cùng Story phải respect Canon dependency.

Các Story khác nhau có thể process concurrently.

---

# 125. Creative Decision Purpose

`CreativeDecision` dùng cho story development có impact đủ lớn để cần explicit control.

Không dùng cho từng chi tiết nhỏ.

---

# 126. Decision Severity

Business severity:

```text
MINOR

SIGNIFICANT

MAJOR

CRITICAL
```

---

# 127. MINOR

AI có thể tự xử lý.

Ví dụ:

```text
minor dialogue

environment detail

small interaction

minor NPC

small clue

scene transition

temporary emotion
```

Thường không tạo CreativeDecision record.

---

# 128. SIGNIFICANT

Có narrative impact nhưng có thể nằm trong Creative Contract.

Ví dụ:

```text
secondary plot thread

minor faction

important item discovery

relationship suspicion

supporting character becoming more important
```

AI có thể tự thực hiện nếu policy/Arc cho phép.

Reviewer phải nhận biết/report appropriate impact.

Không nhất thiết block generation.

---

# 129. MAJOR

Bắt buộc CreativeDecision.

Ví dụ:

```text
betrayal

major relationship change

major character disappearance

new major antagonist

major power gain/loss

major world revelation

major PlotThread resolution

Arc direction change
```

---

# 130. CRITICAL

Ảnh hưởng Story toàn cục.

Ví dụ:

```text
main character death

main villain identity

planned ending change

World Rule change

major timeline rewrite

core premise change
```

Bắt buộc explicit Admin gate.

---

# 131. Severity không chỉ do AI quyết

AI có thể suggest severity.

Nhưng System Rule Classifier phải bảo vệ invariant.

Ví dụ:

```text
CHARACTER_DEATH
+
main_character = true
→ CRITICAL
```

dù AI gọi là MINOR.

---

# 132. Decision Types

Business vocabulary target:

```text
CHARACTER_DEATH

CHARACTER_DISAPPEARANCE

BETRAYAL

ROMANCE_CHANGE

RELATIONSHIP_MAJOR_CHANGE

IDENTITY_REVEAL

MAJOR_DISCOVERY

POWER_GAIN

POWER_LOSS

NEW_MAJOR_CHARACTER

NEW_MAJOR_ANTAGONIST

FACTION_CHANGE

WORLD_RULE_CHANGE

MAJOR_PLOT_THREAD_OPEN

MAJOR_PLOT_THREAD_RESOLVE

ARC_CHANGE

ENDING_CHANGE

CUSTOM
```

Chưa chốt DB enum.

---

# 133. Decision Origin

Decision có thể được trigger bởi:

```text
AI

ADMIN

SYSTEM
```

SYSTEM có thể escalate một event mà Writer không tự khai báo.

---

# 134. CreativeDecision Status

```text
PROPOSED

ANALYZING

WAITING_FOR_ADMIN

SELECTED

REJECTED

POSTPONED

APPLIED

SUPERSEDED

CANCELLED
```

---

# 135. PROPOSED

Decision point vừa được xác định.

---

# 136. ANALYZING

AI/System đang:

```text
generate options

analyze impact

check Canon

simulate future consequences
```

---

# 137. WAITING_FOR_ADMIN

Proposal đã đủ thông tin để human quyết định.

Blocking GenerationRun có thể chuyển:

```text
WAITING
```

---

# 138. SELECTED không phải APPLIED

Admin chọn direction:

```text
SELECTED
```

không có nghĩa event đã xảy ra trong Canon.

---

# 139. APPLIED

Decision đã xảy ra/được áp vào appropriate canonical planning/event point.

---

# 140. Planned Decision vs Actual Canon Event

Ví dụ Admin chọn:

```text
Lan will die in Arc 5 finale.
```

Hiện tại:

```text
Lan = alive
```

vẫn đúng.

Không update Character State thành dead từ lúc SELECTED.

Khi actual Chapter xảy ra event và Memory Extractor commit:

```text
Lan = dead
```

mới trở thành Canon state.

---

# 141. REJECTED

Admin reject direction.

Generation phải replan.

AI không được đơn giản đề xuất lại cùng option ngay sau đó.

---

# 142. POSTPONED

Admin có thể trì hoãn decision.

Phải có revisit concept.

Ví dụ:

```text
at end of Arc 4

before Chapter 120

when PlotThread #18 resolves
```

---

# 143. SUPERSEDED

Decision direction cũ bị thay thế bởi một later decision hợp lệ trước khi apply.

Không delete history.

---

# 144. Decision Blocking Level

Concept:

```text
NON_BLOCKING

BLOCK_BEFORE_CHAPTER

BLOCK_BEFORE_CANON_COMMIT

BLOCK_IMMEDIATELY
```

---

# 145. NON_BLOCKING

Current work có thể tiếp tục.

---

# 146. BLOCK_BEFORE_CHAPTER

Decision phải resolved trước một future Chapter/point.

---

# 147. BLOCK_BEFORE_CANON_COMMIT

Draft có thể tồn tại nhưng Canon không được commit trước khi decision resolved.

---

# 148. BLOCK_IMMEDIATELY

Critical conflict làm workflow dừng ngay.

---

# 149. Decision Context

Decision phải có đủ context:

```text
current Story situation

current Arc

relevant Characters

relevant Story Facts

relevant Plot Threads

trigger

decision question
```

---

# 150. AI Options

Option phải chứa hơn một câu summary.

Có thể gồm:

```text
what happens

benefits

risks

Canon impact

Character impact

Arc impact

Ending impact

new opportunities

threads opened/resolved

future complexity
```

---

# 151. AI Recommendation

AI có thể đưa:

```text
Recommended Option
```

kèm lý do.

Recommendation không tự select Major/Critical Decision trong V1.

---

# 152. Admin Custom Option

Major/Critical Decision luôn nên cho:

```text
CUSTOM
```

Admin không bị giới hạn vào option AI.

---

# 153. Custom Decision Analysis

```text
Admin Custom
      ↓
Intent Normalization
      ↓
Impact Analysis
      ↓
Canon Conflict Detection
      ↓
Planning Impact
      ↓
AI Feedback
      ↓
Admin Final Confirmation
```

---

# 154. AI không được silent rewrite Admin intent

Nếu Admin custom:

```text
Lan giả vờ phản bội.
```

AI không được biến thành:

```text
Lan thật sự phản bội.
```

Nếu thấy vấn đề:

```text
warn

explain

suggest adjustment
```

Admin quyết.

---

# 155. Custom Decision conflict World Rule

Nếu Admin muốn event vi phạm World Rule:

```text
CRITICAL CANON CONFLICT
```

System đưa lựa chọn như:

```text
Cancel

Modify Decision

Request World Rule Revision
```

Không silently phá rule.

---

# 156. World Rule change

Nếu muốn thay World Rule thật:

```text
WORLD_RULE_CHANGE
```

phải đi major/critical impact analysis.

Nếu published history bị ảnh hưởng, operation có thể trở thành Retcon/Canon Revision.

---

# 157. Decision Planning Constraint

Selected future Decision có thể trở thành constraint cho Planner.

Ví dụ:

```text
Lan must survive until Arc 5 finale.
```

Planner không được giết Lan trước đó.

---

# 158. Decision Scheduling

Decision có thể target:

```text
Arc

Chapter range

Narrative condition
```

không cần exact Chapter nếu Story OPEN_ENDED.

---

# 159. Rejected Direction Memory

Rejected direction phải được Story planning context nhớ trong appropriate scope.

Nếu không, AI dễ đề xuất lại ngay.

Possible scope concept:

```text
GLOBAL

ARC

UNTIL_EVENT

CURRENT_PLANNING_HORIZON
```

Exact representation sau.

---

# 160. Decision Revalidation

Trước khi APPLY một future selected decision:

```text
revalidate against current Canon
```

Nếu Canon đã thay đổi và old decision không còn hợp lý:

```text
do not blindly force it
```

Có thể:

```text
reopen decision

adapt

postpone

supersede
```

---

# 161. Applied Decision Immutability

`APPLIED` historical Decision không được edit để giả lịch sử khác.

Nếu cần thay trước Publication:

```text
Pre-publish Revision
```

Nếu event đã Published:

```text
Retcon
```

---

# 162. Decision Audit

Expected events:

```text
CREATIVE_DECISION_CREATED

CREATIVE_DECISION_ANALYZED

CREATIVE_DECISION_SELECTED

CREATIVE_DECISION_CUSTOMIZED

CREATIVE_DECISION_REJECTED

CREATIVE_DECISION_POSTPONED

CREATIVE_DECISION_REOPENED

CREATIVE_DECISION_APPLIED

CREATIVE_DECISION_SUPERSEDED
```

---

# 163. Creative Autonomy Setting

Target Story Workflow có thể support:

```text
CONSERVATIVE

BALANCED

EXPRESSIVE
```

Default:

```text
BALANCED
```

Đây là mutable workflow behavior.

Không phải immutable Story identity.

---

# 164. Creative Autonomy không phá invariant

Ngay cả EXPRESSIVE cũng không bypass:

```text
World Rules

Critical Decision Gate

Published Canon protection

Ending change gate
```

---

# 165. Decision Fatigue Requirement

Creative Decision System phải tránh hỏi Admin quá nhiều.

Chỉ escalate meaningful:

```text
irreversible

high-impact

Canon-changing

Arc-changing
```

decisions.

---

# 166. Retcon Classification

Phải phân biệt bốn loại modification.

### Non-canonical Metadata Correction

Không Retcon.

### Narration / Audio Repair

Không Retcon nếu Story Content không đổi.

### Canon Data Repair

Canonical structured memory bị extract sai nhưng published prose đúng.

### True Retcon

Published story history/prose thực sự bị thay đổi.

---

# 167. Canon Data Repair

Ví dụ Published prose nói:

```text
Minh lost the key.
```

nhưng StoryFact sai:

```text
Minh still has the key.
```

Không rewrite Chapter.

Flow:

```text
Published Evidence
       ↓
Detect incorrect memory
       ↓
Re-extract / Correct Canon Data
       ↓
Impact Analysis
       ↓
New Canon Revision
```

Downstream draft generation có thể stale nếu từng dùng memory sai.

---

# 168. True Retcon

Ví dụ published Chapter nói:

```text
Minh obtained the key.
```

Admin muốn Canon mới nói:

```text
Minh never obtained the key.
```

Đây là dangerous historical change.

---

# 169. Published Canon không Normal Edit

UI không có normal:

```text
Edit
Save
```

cho canonical published history.

Thay bằng:

```text
Request Canon Revision
```

---

# 170. Admin không bypass Retcon

ADMIN permission không cho:

```text
DIRECT_MUTATE_PUBLISHED_CANON
```

Admin chỉ có thể sử dụng approved Retcon workflow.

---

# 171. Retcon Request

Retcon phải có explicit request.

Concept:

```text
Target

Current Canon

Proposed Change

Reason
```

Reason bắt buộc cho True Retcon.

---

# 172. Retcon Status

```text
DRAFT

ANALYZING

WAITING_FOR_ADMIN

APPROVED

REPAIRING

READY_TO_APPLY

APPLYING

APPLIED
```

Exit states:

```text
REJECTED

CANCELLED

FAILED

SUPERSEDED
```

---

# 173. Retcon DRAFT

Proposal chưa ảnh hưởng Official Canon.

Có thể sửa/cancel freely.

---

# 174. Retcon ANALYZING

Retcon Impact Analyzer kiểm tra:

```text
Official Canon

Story Facts

Character States

Plot Threads

Story Arcs

Ending Plan

Creative Decisions

Chapter Summaries

Published Chapters

Draft Chapters

Provisional Chapters

Future Chapter Plans
```

---

# 175. Impact Analysis Sources

Kết hợp:

```text
structured dependencies

Story Memory

Historical Retrieval

AI reasoning
```

AI hỗ trợ indirect analysis.

Structured Canon vẫn là source of truth.

---

# 176. Retcon Impact Scope

```text
LOCAL

PROPAGATING

STRUCTURAL
```

---

# 177. LOCAL

Impact hạn chế.

---

# 178. PROPAGATING

Ảnh hưởng nhiều downstream events/states.

---

# 179. STRUCTURAL

Ảnh hưởng nền Story:

```text
World Rule

main Character

main antagonist

major relationship

major Arc

Ending

core premise
```

Warning mạnh nhất.

---

# 180. Retcon Impact Report

Phải trả cụ thể:

```text
Direct Conflicts

Indirect Conflicts

Affected Story Facts

Affected Character States

Affected Plot Threads

Affected Arcs

Affected Plans

Affected Drafts

Affected Provisional Chapters

Affected Published Chapters
```

---

# 181. Dependency Classification

Affected items có thể:

```text
DIRECTLY_AFFECTED

POTENTIALLY_AFFECTED

STALE

UNAFFECTED
```

Không rewrite mọi `POTENTIALLY_AFFECTED` item một cách mù quáng.

---

# 182. Historical ContextSnapshot không được sửa

Old ContextSnapshot phản ánh đúng:

> AI đã nhận context gì lúc đó.

Retcon không rewrite historical GenerationRun, ContextSnapshot hoặc Audit event.

---

# 183. Repair Plan

Sau Impact Analysis phải có Repair Plan.

Ví dụ:

```text
Revise Chapter 50

Re-extract memory

Supersede Fact #87

Recompute Character State

Rewrite Chapter 52

Review Chapter 56

Re-evaluate PlotThread #18

Mark future plans stale

Regenerate affected drafts

Run continuity checks
```

---

# 184. Repair Plan không Auto Execute

System:

```text
analyze

propose

estimate
```

Admin:

```text
approve

modify scope

cancel
```

Không auto rewrite hàng trăm Chapters.

---

# 185. Retcon Approval

Sau Impact Report + Repair Plan:

```text
WAITING_FOR_ADMIN
```

Admin thấy severity và affected scope trước khi approve.

---

# 186. Revision Workspace

Target architecture có concept:

```text
Retcon / Canon Revision Workspace
```

Mục tiêu:

> Không expose half-repaired Canon cho listener.

---

# 187. Current Canon và Revision Line

Trong Retcon repair:

```text
CURRENT OFFICIAL LINE
```

vẫn phục vụ public.

Song song:

```text
REVISION WORKSPACE
```

được sửa/validate.

---

# 188. Revision Workspace không bắt buộc Git-like engine trong MVP

Đây là business requirement:

```text
Retcon repair must not expose a partially repaired Canon.
```

Exact V1 mechanism có thể đơn giản hơn.

---

# 189. Generation trong khi Retcon đang ở DRAFT/ANALYZING

Có thể tiếp tục vì chưa chắc Retcon được apply.

---

# 190. Generation sau Retcon APPROVED

Khi Retcon đã APPROVED và target sửa Canon sau một affected point:

```text
generation relying on affected future Canon
```

phải bị block/pause.

Không cần tạo Story Status mới.

Đây là workflow lock.

---

# 191. Listener trong lúc Retcon Repair

Listener tiếp tục dùng current coherent Official/Published line.

Retcon repair không bắt public system phải downtime.

---

# 192. Retcon READY_TO_APPLY

Tối thiểu cần:

```text
required Content Revisions complete

required Memory Extraction complete

Facts reconciled

Character States reconciled

Plot Threads reconciled

blocking continuity checks pass

required replacement audio ready

no blocking repair step unresolved
```

---

# 193. Retcon và Audio

Nếu canonical Story Content thay đổi, affected public Narration/Audio phải được regenerated tương ứng trước promotion.

Không promote text mới với audio cũ không khớp.

---

# 194. Không regenerate Audio nếu prose không đổi

Downstream Chapter prose không thay thì audio không tự regenerate chỉ vì một old Canon fact changed.

Chỉ artifact thực sự affected mới cần replace.

---

# 195. Retcon Apply

```text
READY_TO_APPLY
      ↓
Admin Confirm
      ↓
APPLYING
      ↓
APPLIED
```

Mục tiêu:

```text
one coherent new Official Canon line
```

---

# 196. Logical Atomic Promotion

Không bắt giant SQL transaction hàng trăm Chapter.

Nhưng public listener không được thấy:

```text
half old
half new
```

Logical promotion phải coherent.

Exact implementation sau.

---

# 197. Historical Versions after Retcon

Không delete old history.

Ví dụ:

```text
Chapter Content v1
→ historical

Chapter Content v2
→ current
```

Old StoryFact có thể:

```text
SUPERSEDED
```

hoặc:

```text
INVALIDATED
```

theo semantics.

---

# 198. Downstream Draft / Provisional sau Retcon

Affected:

```text
DRAFT

PROVISIONAL
```

thường trở thành:

```text
STALE
```

và được regenerate/review theo Repair Plan.

---

# 199. Published downstream content

Không blind regenerate.

Phân loại:

```text
REQUIRES_REVISION

REQUIRES_REVIEW

UNAFFECTED
```

---

# 200. Retcon Failure

Repair failure không phá current Official line.

Retry repair step.

Current production remains coherent.

---

# 201. Retcon Cancellation

Trước final apply, cancellation không thay current Official Canon.

Revision work có thể retained/discarded theo retention policy sau.

---

# 202. Retcon + CreativeDecision

Applied historical Decision không được edit.

Nếu new Canon direction thay nó:

```text
preserve old Decision

create/supersede through new history
```

tùy case.

---

# 203. Retcon + Ending Plan

Nếu Retcon ảnh hưởng Ending:

```text
Ending Impact Analysis
```

và nếu Ending thay đổi:

```text
new Ending Plan version
```

Không overwrite history.

---

# 204. Retcon Audit

Expected:

```text
RETCON_REQUESTED

RETCON_ANALYSIS_STARTED

RETCON_ANALYZED

RETCON_APPROVED

RETCON_REPAIR_STARTED

RETCON_REPAIR_STEP_COMPLETED

RETCON_READY_TO_APPLY

RETCON_APPLY_STARTED

RETCON_APPLIED

RETCON_CANCELLED

RETCON_FAILED
```

---

# 205. Approval Layers

Không có một generic:

```text
Approved = true
```

cho toàn Chapter.

Phải phân biệt:

```text
Content Approval

Canon Validation / Commit

Narration Approval or Automated Gate

Audio Quality

Publication Approval
```

---

# 206. Approval Revision Binding

Approval bind exact artifact revision.

Nếu artifact đổi:

```text
old approval does not transfer automatically
```

---

# 207. Content Approval Inputs

Admin nên được xem:

```text
Continuity Report

Quality Report

Duration Estimate

Blocking Creative Decisions

Warnings
```

trước khi approve.

---

# 208. Content Approval và Memory Extraction Failure

Nếu Content đã approved nhưng Memory Extraction fail:

```text
Content approval remains historical/current for that revision
```

Chapter chưa chuyển `PRODUCTION`.

Retry Memory Extraction.

---

# 209. Narration Approval

Human gate này configurable bằng mutable workflow settings.

Ví dụ:

```text
pauseBeforeTTS = true
```

thì:

```text
Narration Generated
      ↓
WAITING FOR ADMIN
      ↓
Narration Approved
      ↓
TTS
```

Nếu false, AI/System gates có thể auto advance.

---

# 210. Narration Revision Binding

Narration Revision phải derive từ exact Content Revision.

Nếu Content đổi:

```text
Narration derived from old content
→ STALE
```

---

# 211. Audio Lineage

Audio version phải derive từ exact Narration Revision.

Nếu current Narration thay:

```text
old Audio
→ STALE / non-current
```

---

# 212. Audio Quality Gate

Sau TTS + FFmpeg, kiểm tra:

```text
audio file exists

duration readable

no required segments missing

FFmpeg succeeded

audio derives from current Narration

Narration derives from current Content

artifact is not stale
```

---

# 213. Quality Gate Levels

Quality Gate chuẩn hóa thành:

```text
HARD

OVERRIDABLE

ADVISORY
```

---

# 214. HARD Gate

Không bypass.

Ví dụ:

```text
No approved content

No Official Canon

Missing audio

Audio processing failed

Stale audio lineage

Invalid Chapter sequence

Required previous Canon missing

Blocking Creative Decision unresolved
```

Không có `Approve Anyway`.

---

# 215. OVERRIDABLE Gate

Normally blocks nhưng Admin có thể explicit override.

Ví dụ:

```text
audio duration below 20m minimum

quality score slightly below configured acceptance threshold
```

Override bắt buộc:

```text
actor

timestamp

reason

violated policy
```

---

# 216. ADVISORY

Warning nhưng không block.

Ví dụ:

```text
24m actual duration
vs
30m target

minor quality issue

minor PlotThread inactivity
```

---

# 217. Duration Gate

Default Story Contract:

```text
Target  = 30 minutes
Minimum = 20 minutes
```

Examples:

```text
31m
→ PASS
```

```text
24m
→ PASS
→ Advisory: below target
```

```text
19m48s
→ OVERRIDABLE BLOCK
```

---

# 218. Under-minimum Override

Reason bắt buộc.

Ví dụ:

```text
Intentional short Arc finale.
```

Audit event phải record override.

---

# 219. No-Filler Rule vẫn áp dụng

System không tự thêm filler để đưa:

```text
19:48
```

thành:

```text
20:00
```

Content Gap Analyzer phải đề xuất meaningful expansion.

Admin có thể intentional override.

---

# 220. READY Definition

`READY` chỉ khi:

```text
Current Content Revision approved

Official Canon committed

Current Narration valid

Current Audio READY

All HARD gates pass

Every OVERRIDABLE violation resolved or explicitly overridden
```

---

# 221. Publish Action

```text
READY
 ↓
Admin Publish
 ↓
Final Publish Validation
 ↓
PUBLISHED
```

Publish idempotent.

Double-click không tạo duplicated semantic events.

---

# 222. Final Publish Revalidation

Ngay lúc Publish phải recheck:

```text
current revisions unchanged

active audio still valid

Story Status permits publish

previous Chapter still Published

no Retcon maintenance block

no new hard blocker
```

READY trước đó không được blind trust nếu state đã thay đổi.

---

# 223. No Auto-Publish V1

V1:

```text
AI / System
      ↓
READY
      ↓
ADMIN
      ↓
PUBLISHED
```

Không:

```text
Generation
→ automatic public release
```

---

# 224. Scheduled Publishing Future

Có thể support trong future.

Không thêm `SCHEDULED` vào Chapter Status V1 chỉ để future-proof.

Thiết kế riêng khi cần.

---

# 225. Unpublish Chapter

Unpublish không rollback Canon.

Concept:

```text
PUBLISHED
    ↓
Unpublish
    ↓
READY
```

Official Canon vẫn tồn tại.

Audit vẫn biết Chapter từng Published.

---

# 226. Middle Chapter Unpublish

Nếu:

```text
Chapter 20
```

đang giữa sequence có Chapter 21–30 Published, không được unpublish riêng tạo gap.

System có thể:

```text
Cancel
```

hoặc:

```text
Unpublish Chapter 20 and all subsequent Published Chapters
```

với impact warning.

---

# 227. Unpublish không phải Retcon

Nếu chỉ remove publication:

```text
Canon unchanged
```

Không Retcon.

Nếu canonical Story Content bị đổi:

```text
Canon Revision / Retcon
```

---

# 228. Published Audio Replacement

Có thể:

```text
PUBLISHED
Audio v1 active

→ Generate v2
→ Validate v2
→ Promote v2
```

Chapter vẫn PUBLISHED.

---

# 229. Approval Invalidation

Nếu Content Revision đổi:

```text
old Content Approval no longer applies to new revision

old Narration becomes stale

old Audio becomes stale

relevant AI Reviews become stale
```

Canon/downstream impact tùy lifecycle.

---

# 230. Narration-only Change

Nếu chỉ Narration đổi:

```text
Content Approval remains valid

Official Canon remains valid

old Audio becomes stale
```

---

# 231. Audio-only Change

Nếu chỉ Audio regenerate:

```text
Content remains valid

Canon remains valid

Narration remains valid
```

---

# 232. AI Review Invalidation

AI Review phải bind:

```text
Content Revision

Canon Version
```

Nếu input thay đổi:

```text
Review may become STALE
```

Không trình review cũ như current truth.

---

# 233. Approval Traceability

Human approval phải trace:

```text
actor

artifact/revision

timestamp

warnings visible at approval time

overrides used
```

Không chỉ lưu boolean.

---

# 234. Approval Reason

Normal approval:

```text
reason optional
```

Override/risky operation:

```text
reason required
```

---

# 235. Story Completion Review

Trước:

```text
ACTIVE
→ COMPLETED
```

phải chạy Story Completion Review.

Kiểm tra:

```text
Planned Ending reached

Final Arc completed

Major Plot Threads handled

Major Character Arcs handled

Critical Canon contradictions absent

Final Chapter Published
```

---

# 236. Completion Report

Ví dụ:

```text
✓ Planned Ending reached

✓ Final Arc objective complete

✓ Main antagonist resolved

⚠ PlotThread #18 still OPEN

⚠ Lan final status unresolved
```

Admin xem warnings trước khi mark completed.

---

# 237. Story không tự Completed bởi AI

AI/System cung cấp review/proposal.

Final transition:

```text
ACTIVE
→ COMPLETED
```

cần business-level Admin decision.

---

# 238. Completed Story bình thường vẫn Public

```text
status     = COMPLETED
visibility = PUBLIC
```

là normal listener state.

---

# 239. Concurrent Editing Protection

Hai Admin không được silent overwrite cùng revision.

Ví dụ:

```text
Admin A opens Revision 7

Admin B opens Revision 7

Admin A saves → Revision 8

Admin B later saves old Revision 7
```

System phải detect conflict.

Không silently overwrite Revision 8.

Exact optimistic concurrency implementation sau.

---

# 240. Generation vs Concurrent Edit

Nếu AI job chạy với:

```text
Revision 5
```

và Admin tạo:

```text
Revision 6
```

late result của Revision 5:

```text
STALE
```

không overwrite current content.

---

# 241. Immutable Story Policy invariant

StoryGenerationPolicy đã resolve khi Story được tạo:

```text
immutable
```

Không lifecycle operation nào ở tài liệu này được phép sửa nó.

---

# 242. Mutable Workflow invariant

Các setting như:

```text
batchGenerationSize

preferredModel

preferredTTSProvider

review pause points

planning horizon

creative autonomy
```

vẫn mutable.

---

# 243. Canon Lifecycle invariant

Canonical states tiếp tục là:

```text
DRAFT
 ↓
PROVISIONAL
 ↓
OFFICIAL
```

`PUBLISHED` không được thêm vào Canon lifecycle.

---

# 244. Draft Canon invariant

Draft generation không mutate Official Canon.

---

# 245. Provisional Canon invariant

Provisional Canon chỉ phục vụ controlled generation dependency.

Không được dùng làm public truth.

---

# 246. Memory Extraction invariant

Memory Extraction cho canonical commit chạy trên final approved content.

---

# 247. Canon Commit invariant

Official Canon Commit phải transactional/logically consistent.

Partial Canon change không được coi là successful commit.

---

# 248. Publication invariant

Publication không làm Canon trở thành Official.

Canon phải Official trước khi Chapter có thể READY/PUBLISH.

---

# 249. Admin invariant

Admin quyền cao không có nghĩa business invariant bị bỏ qua.

Đặc biệt không được bypass:

```text
Published Canon protection

sequential Canon

sequential Publish

Critical Creative Decision

Retcon workflow

hard quality gates
```

---

# 250. Audit invariant

Major operations phải audit.

Audit records append-only.

Không rewrite audit history khi Retcon/revision xảy ra.

---

# 251. Provenance invariant

AI-generated data phải trace được nguồn:

```text
GenerationRun

Job

Provider

Model

Prompt Version

Context Snapshot

Source Chapter / Artifact
```

khi relevant.

---

# 252. Historical Integrity invariant

Không rewrite historical:

```text
ContextSnapshot

GenerationRun metadata

Job attempts

Audit events

old Artifact versions

Applied Creative Decisions
```

để giả như current state luôn tồn tại từ đầu.

---

# 253. Core Story/Chapter/Generation state machines

## Story

```text
DRAFT
 ↓
ACTIVE
 ↓
COMPLETED

DRAFT / ACTIVE / COMPLETED
 ↓
ARCHIVED
```

Visibility:

```text
PRIVATE ↔ PUBLIC
```

---

## Chapter

```text
DRAFT
 ↓
CONTENT_REVIEW
 ↓
Content Approval
 ↓
Memory Extraction
 ↓
Official Canon Commit
 ↓
PRODUCTION
 ↓
Narration / TTS / Audio
 ↓
READY
 ↓
Admin Publish
 ↓
PUBLISHED
```

Rare management:

```text
→ ARCHIVED
```

---

## GenerationRun

```text
PENDING
 ↓
RUNNING
```

Có thể:

```text
RUNNING ↔ WAITING
```

Terminal/alternate:

```text
SUCCEEDED

FAILED

CANCELLED

STALE
```

---

## GenerationJob

```text
PENDING
 ↓
RUNNING
 ↓
SUCCEEDED / FAILED / CANCELLED / STALE
```

---

## Canon

```text
DRAFT
 ↓
PROVISIONAL
 ↓
OFFICIAL
```

---

## CreativeDecision

```text
PROPOSED
 ↓
ANALYZING
 ↓
WAITING_FOR_ADMIN
 ↓
SELECTED
 ↓
APPLIED
```

Alternative:

```text
REJECTED

POSTPONED

SUPERSEDED

CANCELLED
```

---

## Retcon

```text
DRAFT
 ↓
ANALYZING
 ↓
WAITING_FOR_ADMIN
 ↓
APPROVED
 ↓
REPAIRING
 ↓
READY_TO_APPLY
 ↓
APPLYING
 ↓
APPLIED
```

Alternative:

```text
REJECTED

CANCELLED

FAILED

SUPERSEDED
```

---

# 254. Final consolidated invariants của checkpoint #2

Các rule sau được xem là **đã chốt**:

```text
Story Status, Visibility và Planning Phase là các khái niệm riêng.

Chapter Status không chứa AI/TTS technical status.

CONTENT_REVIEW là toàn bộ pre-Canon stage của Chapter.

Human Content Approval là revision-bound event.

Official Canon Commit xảy ra sau approved content + Memory Extraction + validation.

Chapter chuyển PRODUCTION sau successful Official Canon Commit.

READY nghĩa là publishable.

PUBLISHED cần Admin action trong V1.

Chapter publication phải sequential.

Official Canon Chapter sequence phải sequential.

Draft không mutate Official Canon.

Provisional Canon chỉ dùng cho staged generation.

Generation results bind input revisions/Canon versions.

Stale results không overwrite current state.

Cancelled results không được late-apply.

Retry khác Regenerate.

Rewrite khác Regenerate.

Retry luôn finite.

Valid upstream artifacts không bị regenerate vô ích sau downstream failure.

Batch generation chạy sequential theo Canon dependency.

Batch failure dừng các downstream Chapter phụ thuộc.

AI output phải được parse + validate trước khi apply.

Major/Critical Creative Decisions cần explicit control.

SELECTED Creative Decision không đồng nghĩa APPLIED.

Future Decision không mutate current Character State trước khi event thực sự xảy ra.

Selected Decision phải revalidate trước khi apply.

Admin Custom Decision vẫn phải qua impact + Canon analysis.

Rejected Decision phải được nhớ theo appropriate planning scope.

Postponed Decision cần revisit trigger.

Applied historical Decision không được edit.

Published canonical content không normal edit.

Canonical metadata/memory extraction error có Canon Data Repair riêng.

True historical story change dùng Retcon.

Retcon phải có impact analysis + Repair Plan trước apply.

Current public Canon phải coherent trong lúc Retcon repair.

Retcon không rewrite historical Audit/ContextSnapshot.

Affected Draft/Provisional content có thể trở thành STALE.

Published downstream content được analyze, không blind-regenerate.

Text-changing Retcon phải chuẩn bị matching Narration/Audio trước public promotion.

Story Visibility và Chapter publication là các khái niệm riêng.

Story có thể PRIVATE dù Chapter đã PUBLISHED.

Unpublish không rollback Official Canon.

Middle-Chapter unpublish không được tạo public sequence gap.

Narration/Audio repair không thay Canon nếu Story Content không đổi.

Quality Gate gồm HARD / OVERRIDABLE / ADVISORY.

HARD Gate không được override.

OVERRIDABLE Gate cần explicit actor + reason + audit.

Target duration 30m là mục tiêu, không phải hard exact duration.

Minimum 20m là normal blocking policy nhưng được intentional Admin override theo workflow.

No-Filler Rule luôn giữ.

No Auto-Publish trong V1.

Story Completion phải qua Completion Review và Admin decision.

Story Generation Policy immutable.

Story Workflow Settings mutable.

Admin không được bypass business invariant.

Audit append-only.

Historical provenance/version/context không bị rewrite.
```

---

# 255. Những phần cố ý chưa quyết định

Sau checkpoint này vẫn **chưa được phép suy diễn thành schema/code implementation** các điểm sau:

```text
Một artifact version nằm cùng table hay table riêng

Approval dùng table riêng hay event model

Retcon Workspace physical representation

Canon branch implementation

Optimistic concurrency column design

JobAttempt table hay event/log

Generation lock implementation

Job lease/heartbeat implementation

Queue/provider technology

Retry attempt exact numbers

Exact timeout durations

Exact concurrency limits

Exact enum persistence

Exact transaction strategy

Exact index strategy

REST endpoint design

Detailed permission matrix

Exact Go architecture/design patterns
```

Các quyết định này phải chờ tới khi business/domain specification của toàn project được chốt đủ sâu.

---

# 256. Trạng thái sau Consolidated Specification #2

Các cụm nghiệp vụ hiện đã được chốt ở mức đủ để tiếp tục:

```text
Product Direction

Story Planning

Story Generation Policy

Story Workflow Settings

Story Bible / Arc / Ending

Character / Character State

StoryFact

PlotThread

Canon

Story Memory

Context Builder

AI Review

Duration Quality

Audio/TTS baseline

Audit / Provenance

Environment isolation

Story Lifecycle

Chapter Lifecycle

Artifact Revision

Generation Lifecycle

Retry / Failure / Staleness

Batch / Provisional Canon

Creative Decision Lifecycle

Retcon / Canon Revision

Canon Data Repair

Approval

Quality Gates

Publish / Unpublish

Story Completion
```

Những cụm nghiệp vụ lớn tiếp theo nên tiếp tục được chốt **trước khi physical schema và architecture implementation được khóa**.
