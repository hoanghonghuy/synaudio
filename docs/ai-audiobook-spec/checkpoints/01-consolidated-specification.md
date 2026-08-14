# AI Audiobook Platform

## Consolidated Specification — Finalized Decisions

**Document status:** Architecture & Product Specification
**Scope:** Các quyết định đã được thống nhất đến thời điểm hiện tại.
**Purpose:** Làm source of truth cho các vòng thiết kế tiếp theo và implementation sau này.

---

# 1. Product Vision

## 1.1 Ý tưởng

Xây dựng một nền tảng nghe truyện audio tiếng Việt trong đó:

* nội dung truyện có thể được AI sáng tác từ đầu;
* AI hỗ trợ xây dựng cốt truyện dài hạn, nhân vật, world state và chapter;
* mỗi chapter được chuyển thành một narration script tối ưu cho việc kể chuyện;
* narration script được chuyển thành audio bằng AI Text-to-Speech;
* audio được hậu kỳ và lưu thành file trước khi publish;
* người dùng có thể nghe truyện theo Story/Chapter;
* người dùng có thể tiếp tục vị trí đang nghe;
* hệ thống tự động hóa phần lớn pipeline sản xuất audiobook.

Trải nghiệm mong muốn không phải:

* giọng đọc quảng cáo;
* giọng MC bản tin;
* giọng review phim;
* voice-over ngắn kiểu TikTok/Shorts;
* TTS robotic đọc văn bản đều đều.

Trải nghiệm mục tiêu là:

> Người nghe có cảm giác đang nghe một narrator thực sự kể truyện, với nhịp nghỉ, cảm xúc, pacing và cách diễn phù hợp với nội dung.

---

# 2. Product Direction

## 2.1 Loại truyện

Hệ thống phải hỗ trợ đồng thời:

### Short Series

Khoảng:

```text
10–50 chapters
```

### Long-running Series

Có thể:

```text
100–500+ chapters
```

Hai loại truyện sử dụng chung Story Engine.

Không thiết kế hai hệ thống khác nhau.

---

# 3. Chapter Duration Policy

Độ dài chapter là một yêu cầu cốt lõi của sản phẩm.

Không chấp nhận chapter audio quá ngắn kiểu:

```text
5 phút
7 phút
10 phút
```

trong pipeline thông thường.

Default policy đã chốt:

```text
Hard minimum audio duration:
20 minutes

Default target audio duration:
30 minutes
```

## 3.1 Minimum và Target

`minimum` là quality constraint.

Ví dụ:

```text
minimum = 20 minutes
```

Nếu audio thực tế:

```text
18:46
```

thì chapter không đạt policy thông thường.

`target` là mục tiêu planning.

Ví dụ:

```text
target = 30 minutes
```

Không bắt buộc chapter phải chính xác:

```text
30:00
```

Chapter có thể:

```text
27 phút
31 phút
34 phút
```

nếu nội dung và pacing hợp lý.

Nhưng minimum thông thường vẫn phải được bảo vệ.

---

# 4. Immutable Story Creation Contract

Một số setting là một phần bản chất của Story.

Chúng được resolve **khi Story được tạo** và sau đó bị khóa.

Flow:

```text
CODE DEFAULT
      ↓
ENV GLOBAL DEFAULT
      ↓
Create Story UI
      ↓
Advanced Settings
      ↓
Admin điều chỉnh nếu cần
      ↓
CREATE STORY
      ↓
Resolved Story Policy Snapshot
      ↓
IMMUTABLE
```

Ví dụ:

```text
minimumChapterDuration = 20m
targetChapterDuration  = 30m
```

Story A được tạo với policy đó.

Sau này `.env` đổi:

```text
target = 45m
```

thì:

```text
Story A vẫn target = 30m
```

Chỉ Story mới nhận default 45m.

---

# 5. Story Generation Policy vs Workflow Settings

Không phải setting nào cũng immutable.

Hệ thống phải phân biệt hai loại.

## 5.1 StoryGenerationPolicy

**Immutable sau khi Story được tạo.**

Ví dụ:

```text
minimumChapterDuration
targetChapterDuration

contentOrigin
language
defaultNarrationLanguage

policyVersion
```

Đây trả lời câu hỏi:

> Story này được định nghĩa như thế nào?

---

## 5.2 StoryWorkflowSettings

**Mutable.**

Ví dụ:

```text
batchGenerationSize

preferredStoryModel
preferredTTSProvider

autoRunAIReview

pauseAfterContentGeneration
pauseBeforeTTS

detailedChapterPlanningHorizon
roughChapterPlanningHorizon
futureArcPlanningHorizon
```

Đây trả lời:

> Hiện tại Admin muốn vận hành pipeline của Story này như thế nào?

Ví dụ:

```text
batchGenerationSize = 1
```

hôm nay.

Ngày mai Admin có thể đổi:

```text
batchGenerationSize = 5
```

mà không làm thay đổi identity của Story.

---

# 6. Content Origin

V1 tập trung vào:

```text
AI_GENERATED
```

Tức:

> truyện được AI sáng tác từ đầu.

Tuy nhiên domain phải future-proof cho:

```text
AI_GENERATED
HUMAN_WRITTEN
AI_ASSISTED
```

## 6.1 HUMAN_WRITTEN trong tương lai

Cho phép người viết import nội dung truyện.

Pipeline sẽ bypass:

```text
Story Writer
```

và đi:

```text
Human Content
      ↓
Narration Adaptation
      ↓
TTS
      ↓
Audio Pipeline
```

## 6.2 AI_ASSISTED

Có thể hỗ trợ người viết sử dụng AI cho:

```text
rewrite
editorial review
continuity
narration
```

nhưng không phải scope V1.

---

# 7. User-facing Roles ở mức Product

Các role baseline:

```text
GUEST
USER
ADMIN
```

RBAC chi tiết chưa chốt ở phase hiện tại.

## 7.1 Guest

Không bắt buộc login để:

```text
browse stories
xem story detail
xem chapter list
nghe audio
đọc chapter text
browse genre
search
```

## 7.2 User

Có thêm:

```text
favorites
listening progress
listening history
continue listening
```

## 7.3 Admin

Có thể:

```text
create story
generate story architecture

edit Story Bible
edit Character
edit Story Arc

generate chapter plan
generate chapter
review chapter

resolve Creative Decision

generate narration
generate audio

publish/unpublish

inspect Canon
inspect Story Facts
inspect Plot Threads

request Retcon

view Audit
```

Role/permission chi tiết sẽ thiết kế riêng.

---

# 8. Frontend Product Direction

Frontend:

```text
Vue 3 + Vite
```

Production:

```text
Vercel
```

Có thể dùng cùng một Vue application:

```text
/
├── listener application
└── /admin
    └── content production/control center
```

---

# 9. Reader / Listener Experience

Core journey:

```text
Home
 ↓
Browse
 ↓
Story Detail
 ↓
Select Chapter
 ↓
Play
 ↓
Pause / Seek
 ↓
Close
 ↓
Return
 ↓
Continue Listening
```

Story nên hỗ trợ cả:

```text
🎧 Listen
📖 Read
```

Chapter text vẫn được hiển thị.

Lợi ích:

* accessibility;
* SEO;
* user có thể vừa nghe vừa đọc;
* content đã tồn tại nên chi phí bổ sung thấp.

---

# 10. Audio Player Baseline

Player cần support:

```text
play
pause
seek

skip backward
skip forward

current time
duration

playback speed

resume
previous chapter
next chapter
```

Future speed options có thể gồm:

```text
0.75x
1.0x
1.25x
1.5x
2.0x
```

---

# 11. Technology Stack

Baseline technology đã thống nhất:

```text
Frontend
Vue 3 + Vite

Production Frontend
Vercel

Backend
Golang

Initial Backend Architecture
Modular Monolith

Production Backend
Render

Database
PostgreSQL

Production PostgreSQL
Neon

Local PostgreSQL
Docker PostgreSQL

Production Object Storage
Cloudflare R2

Local Object Storage
MinIO

Audio Processing
FFmpeg

Story AI
Provider abstraction

Initial TTS
Gemini TTS

Fallback TTS candidates
Google Cloud TTS
Azure Speech
FPT.AI

Local Orchestration
Docker Compose
```

Availability, free-tier quota và pricing của external provider phải được verify lại tại thời điểm triển khai/release.

---

# 12. High-level System Architecture

Production:

```text
                          Text AI
                            │
                            ▼
Vue / Vercel ───────→ Go Backend / Render
     │                      │
     │                      ├──────── Neon PostgreSQL
     │                      │
     │                      ├──────── TTS Providers
     │                      │
     │                      └──────── Cloudflare R2
     │                                  ▲
     └──────── direct audio stream ──────┘
```

Backend chịu trách nhiệm orchestration.

---

# 13. Backend Responsibilities

Go backend chịu trách nhiệm:

```text
REST API

Authentication / Authorization
    (thiết kế chi tiết sau)

Story domain

Story Planning

Canon management

Story Memory

AI generation orchestration

TTS orchestration

Generation Jobs

Audio metadata

Object storage

Listening Progress

Favorites

Audit

Provenance
```

Backend **không được**:

```text
stream/proxy toàn bộ MP3 từ R2 đến client

lưu permanent audio trên filesystem Render

generate TTS real-time mỗi khi listener bấm Play
```

---

# 14. Direct Audio Delivery

Không dùng:

```text
Browser
 ↓
Go
 ↓
R2
 ↓
Go
 ↓
Browser
```

Dùng:

```text
Browser
 ↓
Go API
 ↓
metadata/audio URL

Browser
 ↓
R2 / CDN
```

Backend chỉ trả metadata như:

```json
{
  "chapterId": "...",
  "audioUrl": "...",
  "duration": 1843
}
```

---

# 15. Audio Generation Principle

Audio phải:

```text
Generate once
     ↓
Store once
     ↓
Play many times
```

Không:

```text
User Play
     ↓
Generate TTS again
```

Điều này giảm:

```text
latency
cost
provider quota usage
inconsistency
failure rate
```

---

# 16. AI Story Engine

Đây là subsystem cốt lõi của sản phẩm.

High-level pipeline:

```text
STORY IDEA
    ↓
STORY ARCHITECT
    ↓
STORY BIBLE
CHARACTERS
ENDING PLAN
HIGH-LEVEL ARCS
    ↓
ARC PLANNER
    ↓
CHAPTER PLANNER
    ↓
STORY WRITER
    ↓
CONTENT GAP ANALYZER
    ↓
CONTINUITY REVIEWER
    ↓
QUALITY REVIEWER
    ↓
REWRITER
    ↓
ADMIN CONTENT REVIEW
    ↓
MEMORY EXTRACTOR
    ↓
CANON COMMIT
    ↓
NARRATION DIRECTOR
    ↓
NARRATION REVIEW
    ↓
TTS
    ↓
AUDIO QUALITY GATE
    ↓
PUBLISH
```

---

# 17. Story Creation From a Single Idea

Admin có thể nhập một idea đơn giản.

Ví dụ:

```text
Một sinh viên thuê trọ trong một căn nhà
mà mỗi đêm lại xuất hiện thêm một căn phòng mới.
```

Story Architect có thể đề xuất:

```text
Premise

Genre

Tone

Story Bible

Main characters

Main conflict

Long-term direction

Planned ending

Major arcs

Initial plot threads
```

Không generate chi tiết 500 chapter ngay từ đầu.

---

# 18. Story Bible

Story Bible là phần Canon tương đối ổn định.

Ví dụ structure:

```text
premise

genre

tone

target audience

writing style
├── POV
├── tense
├── dialogue style
└── description style

world
├── setting
├── time period
└── rules

main plot
├── starting situation
├── main conflict
├── long-term goal
└── planned ending direction

global constraints
```

Story Bible không phải toàn bộ memory của Story.

---

# 19. Characters

Character phải có hai lớp.

## 19.1 Static Character Profile

Ít thay đổi:

```text
name
aliases

age
gender

appearance

personality

background

motivation

fear

strength
weakness

speech style
```

---

## 19.2 Dynamic Character State

Thay đổi theo story:

```text
current location

physical condition

emotional state

inventory

knowledge

relationships

current goals

abilities

status
```

Ví dụ:

```json
{
  "location": "abandoned_house",
  "condition": "injured",
  "inventory": [],
  "knowledge": [
    "Lan lied",
    "the basement exists"
  ],
  "emotionalState": "suspicious"
}
```

Character State hoạt động gần giống game entity state.

---

# 20. Story Arc

Story được tổ chức:

```text
Story
│
├── Story Bible
├── Characters
│
├── Arc 1
│   ├── Chapter ...
│
├── Arc 2
│   ├── Chapter ...
│
└── ...
```

Mỗi Arc có thể chứa:

```text
objective

main conflict

starting state

ending condition

key events

required revelations

character development

expected chapter range

status
```

Arc giúp Story Writer biết:

> Chapter này đang đưa câu chuyện về đâu?

---

# 21. Chapter Plan

Chapter Plan là artifact bắt buộc.

Không generate prose trực tiếp chỉ dựa vào:

```text
Write next chapter.
```

Chapter Plan có thể chứa:

```text
chapter objective

opening

scene list

characters per scene

scene purpose

required Canon facts

planned new facts

plot threads to advance

cliffhanger

duration budget
```

---

# 22. Scene Duration Budget

Target duration được phân bố vào scenes.

Ví dụ target:

```text
30 minutes
```

Chapter Plan có thể:

```text
Scene 1 — Opening
4m

Scene 2 — Investigation
6m

Scene 3 — Character Conflict
5m

Scene 4 — Discovery
6m

Scene 5 — Consequence
5m

Scene 6 — Cliffhanger
4m
```

Tổng:

```text
≈ 30m
```

Đây là pacing budget, không phải hard exact duration cho từng scene.

---

# 23. Duration Estimation Pipeline

Không chờ tới audio cuối mới kiểm tra duration.

Pipeline:

```text
Chapter Plan Duration Budget
        ↓
Story Content
        ↓
Text Duration Estimate
        ↓
Narration Script
        ↓
Narration Duration Estimate
        ↓
TTS
        ↓
Actual Audio Duration
```

`Actual Audio Duration` là source of truth cuối cùng.

---

# 24. Under-duration Quality Gate

Nếu chapter không đạt duration:

```text
Actual:
18:54

Minimum:
20:00

Target:
30:00
```

Hệ thống không chỉ trả:

```text
FAILED
```

Mà phải chạy:

```text
Content Gap Analyzer
```

để tìm nguyên nhân.

Ví dụ:

```text
Scene 2 development chưa đủ.

Scene 4 giải quyết conflict quá nhanh.

Character interaction giữa Minh/Lan chưa được khai thác.

Plot Thread #17 có thể advance tự nhiên tại chapter này.
```

---

# 25. Under-duration Resolution

AI/System có thể đề xuất:

```text
Option A
Expand investigation scene

Option B
Add meaningful character interaction

Option C
Advance Plot Thread #17

Option D
Regenerate Chapter

Option E
Regenerate Chapter Plan

Option F
Admin edit manually
```

Có thể có:

```text
Approve Anyway
```

nhưng là advanced override.

Override:

* phải intentional;
* phải warning;
* phải audit.

---

# 26. No-Filler Rule

AI không được kéo dài chapter bằng:

```text
repetition

meaningless description

duplicated dialogue

artificially slow narration

fake pauses

văn phong lan man không đóng góp cho:
plot
character
world
atmosphere
```

Nếu thiếu duration:

```text
analyze content gap
→ add meaningful content
```

không phải:

```text
make text longer
```

---

# 27. Creative Freedom Model

AI sử dụng mô hình:

> **Controlled Creativity**

AI không bị ép tuyệt đối theo từng chữ trong outline.

Nhưng cũng không được tùy tiện phá Story Canon.

---

# 28. AI Allowed Minor Creativity

AI có thể tự quyết định các detail nhỏ như:

```text
dialogue

minor character interaction

environment description

minor clues

minor conflict

small plot thread

scene pacing

minor secondary character
```

miễn không conflict Canon.

---

# 29. Major Creative Decisions

Các decision quan trọng phải qua hệ thống kiểm soát.

Ví dụ:

```text
CHARACTER_DEATH

BETRAYAL

ROMANCE_CHANGE

IDENTITY_REVEAL

MAJOR_DISCOVERY

POWER_CHANGE

VILLAIN_REVEAL

ARC_CHANGE

ENDING_CHANGE

CUSTOM
```

---

# 30. Creative Decision Workflow

Khi cần quyết định lớn:

```text
AI identifies decision point
        ↓
AI proposes Option A
AI proposes Option B
AI proposes Option C
        +
Admin Custom Option
        ↓
Impact Analyzer
        ↓
Canon Conflict Check
        ↓
Future Plot Analysis
        ↓
Admin Review
        ↓
Confirm
        ↓
Apply to Canon
```

---

# 31. Admin Custom Decision

Admin không bị giới hạn bởi các option AI đưa ra.

Admin có thể tự viết:

```text
CUSTOM DECISION
```

Ví dụ:

```text
Lan giả vờ phản bội Minh để thâm nhập phe đối phương,
nhưng độc giả chưa được biết điều đó trong arc hiện tại.
```

AI phải phân tích Custom Decision:

```text
affected characters

affected relationships

affected plot threads

affected arcs

Canon conflicts

future opportunities

future risks
```

Sau đó Admin mới confirm.

AI đóng vai trò:

```text
advisor
impact analyzer
continuity protector
```

không phải người có authority cuối cùng.

---

# 32. Planned Ending

Story luôn nên có một:

```text
Current Planned Ending
```

kể cả OPEN_ENDED Story.

Ending giúp AI biết câu chuyện đang hướng về đâu.

Ending không hoàn toàn immutable.

Có thể revise có kiểm soát.

Ending change phải đi qua:

```text
Creative Decision
      ↓
Impact Analysis
      ↓
Admin Approval
      ↓
New Ending Plan Version
```

Không overwrite ending cũ mà mất lịch sử.

---

# 33. Story Planning Modes

Hệ thống hỗ trợ:

```text
FINITE
OPEN_ENDED
```

---

# 34. FINITE Mode

Khi tạo FINITE Story, AI phân tích:

```text
idea
complexity
genre
chapter target duration
main conflict
planned ending
```

rồi đề xuất scope.

Ví dụ:

```text
Option A
120–150 chapters
6 arcs

Option B
180–220 chapters
9 arcs

Option C
250–300 chapters
12 arcs

Custom
```

Admin:

```text
Select
```

hoặc nhập Custom.

Chapter count là:

```text
planning range
```

không phải exact hard limit.

Truyện có thể kết thúc:

```text
174
181
193
```

nếu pacing yêu cầu.

---

# 35. OPEN_ENDED Mode

OPEN_ENDED không có nghĩa:

```text
AI cứ nghĩ tiếp vô hạn.
```

Nó vẫn phải có:

```text
long-term destination

current planned ending

current arc

next arc direction
```

Story sẽ được mở rộng theo từng planning horizon.

---

# 36. Story Planning Phase

Story có planning phase:

```text
ONGOING

CLOSING

FINAL_ARC

COMPLETED
```

---

# 37. OPEN_ENDED → Closing

Ví dụ Story đang:

```text
OPEN_ENDED
ONGOING
```

sau 400 chapters, Admin quyết định:

```text
Begin Ending Plan
```

AI có thể đề xuất:

```text
Option A
2 arcs
~60 chapters

Option B
3 arcs
~90 chapters

Option C
Custom
```

Admin chọn.

Story chuyển:

```text
planningPhase = CLOSING
```

---

# 38. Final Arc Rules

Khi:

```text
planningPhase = FINAL_ARC
```

AI không được tự do mở thêm major plot threads.

Nếu AI muốn mở major thread:

```text
WARNING
```

hoặc:

```text
Creative Decision Gate
```

Mục tiêu FINAL_ARC:

```text
resolve major plot threads

complete character arcs

resolve major conflicts

deliver promised revelations

move toward ending
```

---

# 39. Story Completion

Khi:

```text
planningPhase = COMPLETED
storyStatus = COMPLETED
```

normal:

```text
Generate Next Chapter
```

bị disable.

Nếu muốn tiếp tục nội dung:

```text
Create Sequel
```

được ưu tiên hơn reopening Story đã hoàn thành.

---

# 40. Planning Horizon

Không plan detailed 500 chapters ngay từ đầu.

Rule:

> càng gần hiện tại càng detailed; càng xa càng abstract.

Ví dụ:

```text
Chapter 201
FULL PLAN

Chapter 202
FULL PLAN

Chapter 203
FULL PLAN

Chapter 204–210
ROUGH PLAN

Current Arc
DETAILED

Next Arc
MEDIUM DETAIL

Future Arcs
HIGH LEVEL

Ending
STRATEGIC
```

Flow:

```text
ROUGH
 ↓
REFINE
 ↓
DETAILED
 ↓
EXECUTE
```

---

# 41. Arc Completion Review

Không âm thầm tạo Arc tiếp theo.

Khi Arc gần kết thúc:

```text
Arc Completion Review
        ↓
AI Analysis
        ↓
Next Arc Proposal
```

AI kiểm tra:

```text
Arc objective completed?

Plot threads resolved/advanced?

Character development sufficient?

Pacing hợp lý?

Long-term ending tiến triển?

Current Canon state?
```

Sau đó:

```text
Option A
Option B
Option C
Custom
```

Admin/Semi-auto mới quyết định.

Proposal phải được lưu để trace.

---

# 42. Story Canon

Canon là **source of truth chính thức của Story**.

Canon gồm:

```text
Story Bible

Ending Plan

Story Arcs

Character Profiles

Character States

Story Facts

Plot Threads

Approved Creative Decisions

World State
```

Full chapter prose vẫn được lưu, nhưng không phải cách duy nhất để xác định current state.

---

# 43. Canon Lifecycle

Các trạng thái logic:

```text
DRAFT
     ↓
AI + SYSTEM GATES
     ↓
PROVISIONAL
     ↓
ADMIN APPROVAL
     ↓
OFFICIAL
```

---

# 44. Draft Rule

**Draft tuyệt đối không được mutate Official Canon.**

AI có thể generate:

```text
Draft Chapter
```

nhưng current Canon chưa thay đổi.

---

# 45. Memory Extraction After Human Approval

Memory extraction chỉ chạy **sau khi Admin approve final content**.

Ví dụ AI draft:

```text
Lan nhặt chiếc chìa khóa.
```

Admin sửa:

```text
Minh nhặt chiếc chìa khóa.
```

Nếu Memory Extractor chạy trước Admin edit, Canon sẽ sai.

Do đó:

```text
AI Draft
    ↓
AI Reviews
    ↓
Admin Edit
    ↓
ADMIN APPROVED CONTENT
    ↓
Memory Extractor
    ↓
Canon Change Set
    ↓
Validation
    ↓
Canon Commit
```

---

# 46. Canon Commit

Canon commit phải transactional.

Flow:

```text
Approved Chapter
      ↓
Memory Extractor
      ↓
Structured Change Set
      ↓
Backend Validation
      ↓
Database Transaction
      ↓
New Official Canon Version
```

Không được:

```text
update 3 table thành công
table 4 fail
Canon half-updated
```

---

# 47. StoryFact

StoryFact là Core Capability.

Không để StoryFact sang future vì long-running series cần nó ngay.

Ví dụ:

```text
Fact #120

subject:
Minh

type:
POSSESSION

content:
Minh possesses the old key.

status:
SUPERSEDED
```

sau đó:

```text
Fact #182

subject:
Minh

type:
POSSESSION

content:
Minh lost the old key.

status:
ACTIVE
```

StoryFact status:

```text
ACTIVE
SUPERSEDED
INVALIDATED
```

Không delete fact cũ chỉ vì nó không còn hiện tại.

Lịch sử vẫn phải được giữ.

---

# 48. PlotThread

PlotThread cũng là Core.

Ví dụ:

```text
Who entered the house before Minh?

status:
OPEN

importance:
MAJOR

introduced:
Chapter 6
```

State có thể:

```text
OPEN
ADVANCING
RESOLVED
ABANDONED
```

Chapter Planner phải nhận active Plot Threads.

Nó có thể quyết định:

```text
ADVANCE
RESOLVE
IGNORE THIS CHAPTER
OPEN NEW MINOR THREAD
```

---

# 49. Story Memory Engine

Mục tiêu:

> Cho AI hoạt động như thể nó nhớ được toàn bộ truyện mà không phải nhét hàng trăm chapter raw text vào mỗi prompt.

Không dựa vào:

```text
chat history
model memory
```

làm source of truth.

Story Memory nằm trong backend/database.

---

# 50. Full Chapter Archive

Không xóa raw chapter history.

Hệ thống vẫn lưu:

```text
Chapter 1 full text
Chapter 2 full text
...
Chapter 300 full text
...
```

Story Memory không có nghĩa bỏ raw story.

Nó là cách tổ chức và retrieval thông minh hơn.

---

# 51. Hierarchical Story Memory

Memory có nhiều layer.

## Level 0 — Canon Constitution

```text
Story Bible

World Rules

Writing Rules

Major Constraints

Current Planned Ending
```

---

## Level 1 — Current State

```text
Current Arc

Character States

World State

Relationships

Inventory

Knowledge
```

---

## Level 2 — Active Narrative Memory

```text
Active Facts

Open Plot Threads

Current Goals

Unresolved Mysteries

Unresolved Setup/Payoff
```

---

## Level 3 — Hierarchical Summary

```text
Chapter Summary
      ↓
Arc Summary
      ↓
Story-so-far Summary
```

---

## Level 4 — Recent Context

Ví dụ khi viết Chapter 289:

```text
Chapter 288
high detail / relevant excerpt

Chapter 287
detailed summary

Chapter 286
detailed summary

recent chapter summaries
```

---

## Level 5 — Historical Retrieval

Nếu Chapter 289 nhắc lại một vật từ Chapter 17:

```text
Context Builder
      ↓
Historical Retriever
      ↓
find relevant old facts/scenes
```

Ví dụ lấy:

```text
Chapter 17:
origin of the watch

Chapter 43:
watch was lost

Chapter 114:
someone saw it again
```

rồi đưa các context liên quan vào prompt.

---

# 52. Temporal Memory Rule

Càng gần hiện tại càng giữ nhiều detail.

```text
Recent
██████████████████

Current Arc
██████████████

Previous Arcs
██████

Very Old History
██
```

Nhưng old history có thể được retrieve lại nếu relevant.

---

# 53. Context Builder

Context Builder là subsystem quan trọng.

Chapter Writer không tự quyết định toàn bộ context.

Flow:

```text
Generation Request
        ↓
Context Builder
        │
        ├── Story Policy
        ├── Story Bible
        ├── Ending Plan
        ├── Current Arc
        ├── Chapter Plan
        ├── Relevant Characters
        ├── Character States
        ├── Active Facts
        ├── Plot Threads
        ├── Recent Context
        ├── Historical Retrieval
        └── Approved Creative Decisions
        ↓
Context Pack
        ↓
LLM
```

---

# 54. Story Memory Retrieval V1

Không cần vector database ngay.

V1 ưu tiên relational retrieval dựa trên:

```text
story_id

character_id

arc_id

entity

importance

status

fact type

plot thread
```

---

# 55. Semantic Retrieval Future

Khi Story lớn:

```text
chapter summaries

facts

important scenes

plot threads
```

có thể được embedding.

Ưu tiên hướng:

```text
PostgreSQL
+
pgvector
```

trước khi thêm standalone vector database.

Semantic retrieval:

```text
RETRIEVAL
```

không phải:

```text
TRUTH
```

Canon vẫn là source of truth.

---

# 56. Context Snapshot

Mỗi Generation Run cần lưu đủ metadata để biết:

> Model đã được cung cấp những gì?

Ví dụ:

```text
Canon Version:
288

Story Bible Version:
8

Ending Plan Version:
3

Arc Version:
4

Character States:
Minh v288
Lan v287

Fact IDs:
#21
#87
#183

Plot Thread IDs:
#12
#19

Historical retrieval:
Chapter 17
Chapter 43
Chapter 114

Prompt Version:
chapter-write-v7

Model:
...
```

Mục tiêu:

```text
debug

reproduce

audit

compare models

compare prompts
```

---

# 57. Fact Provenance

Mỗi fact phải biết nó xuất phát từ đâu.

Ví dụ:

```text
Fact #185

content:
Minh lost the old key.

sourceChapter:
73

sourceScene:
4

generationRun:
run_123

canonVersion:
73
```

Nếu AI/Developer hỏi:

> Tại sao hệ thống tin Minh mất chìa khóa?

ta có thể truy lại evidence.

---

# 58. AI Continuity Protection

Không phụ thuộc một lớp AI duy nhất.

Defense-in-depth:

```text
Context Builder
      ↓
Story Writer
      ↓
Continuity Reviewer
      ↓
Fact Conflict Detection
      ↓
Creative Decision Gate
      ↓
Admin Review
      ↓
Memory Extractor
      ↓
Canon Validation
      ↓
Canon Commit
```

---

# 59. Continuity Reviewer

Kiểm tra:

```text
timeline

character state

character knowledge

relationships

inventory

world rules

Story Facts

Plot Threads

Arc objective

ending constraints
```

Ví dụ Writer viết:

```text
Minh lấy chiếc chìa khóa khỏi túi.
```

nhưng Canon:

```text
Minh.inventory = []
Fact:
Minh lost old key.
```

Reviewer phải phát hiện:

```text
CANON CONFLICT
```

---

# 60. Writing Quality Reviewer

Kiểm tra:

```text
repetition

pacing

dialogue quality

AI-like wording

over-explanation

boring sections

chapter opening

chapter ending

filler

artificial lengthening
```

---

# 61. Rewriter Separation

Reviewer không nhất thiết tự sửa prose.

Pipeline:

```text
Reviewer
    ↓
issues[]
    ↓
Rewriter
```

Điều này giúp:

```text
debug
compare
regenerate
audit
```

tốt hơn.

---

# 62. Semi-auto Workflow

Default workflow ưu tiên:

```text
Plan
 ↓
Generate
 ↓
Continuity Review
 ↓
Quality Review
 ↓
Rewrite
 ↓
STOP
 ↓
ADMIN CONTENT REVIEW
```

Sau Admin approve mới:

```text
Memory Extraction
 ↓
Canon Commit
```

---

# 63. Manual Workflow

Admin có thể cấu hình pipeline dừng từng stage:

```text
Plan Review

Content Review

Narration Review

Audio Review
```

để kiểm soát kỹ hơn.

---

# 64. Generation Batch

Admin có thể chỉnh:

```text
batchGenerationSize
```

Ví dụ:

```text
1
3
5
10
Custom
```

Nhưng chapter generation phải **sequential**, không parallel theo chapter dependency.

---

# 65. Provisional Canon

Ví dụ:

```text
Official Canon v100

      ↓

Chapter 101
      ↓
Provisional Canon v101

      ↓

Chapter 102
      ↓
Provisional Canon v102

      ↓

Chapter 103
```

Điều này cho phép batch generation.

---

# 66. Stale Downstream Content

Nếu Admin sửa Chapter 101 làm Canon thay đổi:

```text
Chapter 102
Chapter 103
```

có thể được generate từ Canon cũ.

Hệ thống phải:

```text
mark STALE
```

và cảnh báo:

```text
These chapters were generated from an outdated Canon state.
```

Có thể đề xuất:

```text
Regenerate affected chapters
```

---

# 67. Retcon

Retcon là việc sửa Canon lịch sử.

Ví dụ:

```text
Story đã publish Chapter 1–200

Admin muốn thay đổi sự kiện Chapter 50
```

Hệ thống **hỗ trợ**, nhưng phải xem đây là:

```text
DANGEROUS OPERATION
```

Product nên strongly discourage nếu downstream đã lớn.

---

# 68. Published Content Lock

Chapter:

```text
DRAFT
```

được edit bình thường.

Chapter:

```text
PROVISIONAL
```

được edit nhưng có thể invalidate downstream.

Chapter/Canon:

```text
OFFICIAL
```

bị lock.

Chapter:

```text
PUBLISHED
```

bị strongly locked.

Không hiển thị normal Edit cho historical Published Canon.

Thay vào đó:

```text
Request Canon Revision
```

---

# 69. Retcon Workflow

```text
Request Canon Revision
        ↓
Describe Proposed Change
        ↓
Retcon Impact Analyzer
        ↓
Dependency Analysis
        ↓
Impact Report
        ↓
Repair Plan
        ↓
Admin Confirmation
        ↓
Apply Retcon
        ↓
Mark Downstream Stale
        ↓
Controlled Repair
```

---

# 70. Retcon Impact Analysis

Phải phân tích:

```text
direct conflicts

indirect conflicts

affected facts

affected characters

affected character states

affected plot threads

affected arcs

affected chapter plans

affected generated drafts

affected published chapters
```

---

# 71. Retcon Example

Admin thay:

```text
Chapter 50:
Minh did NOT obtain the key.
```

Analyzer có thể trả:

```text
Direct conflicts:

Chapter 52
Minh uses the key.

Chapter 56
Lan asks Minh about the key.


Indirect conflicts:

Chapter 83 assumes the basement was opened.


Affected:

Fact #87
Plot Thread #18
Minh inventory state
Minh knowledge state
```

Repair Plan:

```text
1. Rewrite Chapter 52

2. Rewrite Chapter 56

3. Update affected facts

4. Re-evaluate Plot Thread #18

5. Verify Chapter 57–83
```

---

# 72. No Automatic Massive Retcon Rewrite

Không:

```text
Edit Chapter 50
 ↓
AI automatically rewrites Chapter 51–200
```

Hệ thống:

```text
analyze
propose
plan
mark stale
```

Admin quyết định repair scope.

---

# 73. Narration Script

Story text và narration script là **hai artifact khác nhau**.

Ví dụ Story Content:

```text
Cánh cửa bật mở.

Minh giật mình quay lại.

Không có ai ở đó.
```

Narration Script có thể chứa:

```text
[tense, slow]

Cánh cửa... bật mở.

[pause]

Minh giật mình quay lại.

[lower voice]

Không có ai ở đó.
```

Format cụ thể phụ thuộc TTS provider abstraction.

---

# 74. Voice Strategy V1

V1 ưu tiên:

```text
ONE MAIN NARRATOR
```

Voice mục tiêu:

```text
Vietnamese

adult

warm

natural

storytelling

controlled emotion

medium / slow pace

clear pauses
```

Dialogue có thể thay đổi delivery/tone nhưng không bắt buộc đổi voice.

---

# 75. Multi-character Voice Future

Không bắt đầu bằng:

```text
Narrator
Male Lead
Female Lead
Villain
Character 1
Character 2
...
```

vì tăng:

```text
complexity
TTS requests
voice consistency problems
mixing complexity
cost
```

Future architecture vẫn support:

```text
Narrator
Character A
Character B
...
```

qua scene/segment generation.

---

# 76. Voice Rights

Không clone hoặc mô phỏng chính xác giọng narrator/người thật cụ thể nếu chưa có quyền phù hợp.

Sản phẩm nên tạo:

```text
own narrator identity

own voice style

own prompt presets
```

Mục tiêu:

```text
brand identity
lower legal risk
less dependency
commercial flexibility
```

---

# 77. Audio Pipeline

```text
Approved Chapter Content
       ↓
Narration Script
       ↓
Segmenter
       ↓
TTS Segments
       ↓
Temporary Audio
       ↓
FFmpeg
       ↓
Normalize
       ↓
Join
       ↓
Encode
       ↓
Audio Quality Check
       ↓
Upload Object Storage
       ↓
AudioAsset READY
```

---

# 78. TTS Chunking

Không generate chapter rất dài trong một TTS request.

Chia:

```text
Chapter
   ↓
Scene
   ↓
Segment
Segment
Segment
```

Lợi ích:

```text
API limit safety

retry riêng segment lỗi

scene-level voice direction

multi-speaker future

lower regeneration cost

easier audio composition
```

---

# 79. FFmpeg Responsibilities

FFmpeg dùng cho:

```text
concatenate

normalize loudness

format conversion

bitrate conversion

duration inspection

optional silence processing
```

Baseline output:

```text
MP3
64–96 kbps
```

Exact audio engineering parameters như LUFS/sample-rate sẽ thiết kế ở Audio Specification sau.

---

# 80. AudioAsset Versioning

Một Chapter có thể có nhiều audio versions.

Ví dụ:

```text
v1
Gemini Voice A

v2
Gemini with improved prompt

v3
FPT voice
```

Một version được active.

Không overwrite audio cũ ngay lập tức.

Logical concept:

```text
chapter_id

version

provider
model
voice

object_key

duration

bitrate

status

is_active
```

---

# 81. Audio Object Key

Dùng ID thay vì raw title.

Ví dụ:

```text
audio/
  stories/
    {story_id}/
      chapters/
        {chapter_id}/
          v1.mp3
```

Future:

```text
{voice_id}/v2.mp3
```

---

# 82. Audio Retry

Nếu segment TTS fail:

```text
retry segment
```

không generate lại toàn chapter.

Job cần biết:

```text
attempts

provider

model

error
```

---

# 83. Audio Idempotency

Nếu Chapter đã có:

```text
READY audio
```

request generate lại không được âm thầm tạo duplicate.

Có thể:

```text
reject

force regeneration

create new version
```

Exact API behavior sẽ chốt sau.

---

# 84. Database Role

PostgreSQL lưu structured/domain data:

```text
Story

Story Bible

Arc

Characters

Character States

Chapters

Narration Script

Canon

Facts

Plot Threads

Creative Decisions

Jobs

Audit

Users

Progress

Audio metadata
```

Không lưu:

```text
MP3 binary

cover binary

large media blobs
```

---

# 85. Core Logical Entities

Target domain gồm:

```text
Story

Genre
StoryGenre

StoryGenerationPolicy
StoryWorkflowSettings

StoryBible
StoryArc
StoryEndingPlan

Character
CharacterState

ChapterPlan
Chapter
ChapterSummary

StoryFact
PlotThread
CreativeDecision

CanonVersion

GenerationRun
GenerationJob

ContextSnapshot

AudioAsset

User
Favorite
ListeningProgress

AuditEvent
```

---

# 86. Story Status

Baseline:

```text
DRAFT

ACTIVE

COMPLETED

ARCHIVED
```

`ARCHIVED` không có nghĩa bị delete.

---

# 87. Planning Mode

```text
FINITE
OPEN_ENDED
```

---

# 88. Planning Phase

```text
ONGOING
CLOSING
FINAL_ARC
COMPLETED
```

---

# 89. Chapter Status

Không nhồi toàn bộ AI pipeline vào một chapter status khổng lồ.

Chapter domain status baseline:

```text
DRAFT
REVIEW
READY
PUBLISHED
ARCHIVED
```

Generation state nằm trong:

```text
GenerationRun
GenerationJob
```

Audio state nằm trong:

```text
AudioAsset
```

---

# 90. Audio Status

```text
PENDING

GENERATING

READY

FAILED

ARCHIVED
```

---

# 91. Generation Job Status

```text
PENDING

RUNNING

SUCCEEDED

FAILED

CANCELLED
```

---

# 92. StoryFact Status

```text
ACTIVE

SUPERSEDED

INVALIDATED
```

---

# 93. PlotThread Status

```text
OPEN

ADVANCING

RESOLVED

ABANDONED
```

---

# 94. CreativeDecision Status

```text
PROPOSED

WAITING_FOR_ADMIN

SELECTED

REJECTED

APPLIED
```

---

# 95. Audit Field Convention

Không thêm field chỉ để table trông giống nhau.

Mutable entity nếu cần:

```text
created_at
created_by

updated_at
updated_by

deleted_at
deleted_by
```

Immutable entity:

```text
created_at
created_by
```

và không có meaningless `updated_at`.

---

# 96. Immutable Table Example

`StoryGenerationPolicy` immutable.

Có thể có:

```text
created_at
created_by
```

Không có:

```text
updated_at
updated_by
```

vì không support update.

---

# 97. Soft Delete Convention

Không cần đồng thời:

```text
delete_flag
+
deleted_at
```

Thông thường:

```text
deleted_at IS NULL
```

đã đủ.

Nếu cần actor:

```text
deleted_by
```

---

# 98. Business State vs Delete

Không dùng generic delete để thay thế domain semantics.

Ví dụ StoryFact:

```text
SUPERSEDED
```

không phải deleted.

Story:

```text
ARCHIVED
```

không phải deleted.

PlotThread:

```text
RESOLVED
```

không phải deleted.

---

# 99. Timestamp Convention

Database timestamp:

```text
TIMESTAMPTZ
UTC
```

Client chịu trách nhiệm presentation timezone.

---

# 100. Audit Log

Audit Log là Core Capability.

Target:

```text
audit_events
```

Audit event là:

```text
APPEND ONLY
```

Không được sửa historical audit event.

Không cần:

```text
updated_at
deleted_at
```

---

# 101. Audit Responsibilities

Audit trả lời:

```text
WHO?

WHAT?

WHEN?
```

Ví dụ event:

```text
STORY_CREATED

STORY_ARCHIVED

STORY_BIBLE_CHANGED

CHARACTER_CHANGED

CHARACTER_STATE_MANUALLY_CHANGED

CHAPTER_GENERATED

CHAPTER_APPROVED

CHAPTER_PUBLISHED

CREATIVE_DECISION_PROPOSED

CREATIVE_DECISION_APPROVED

CANON_COMMITTED

RETCON_REQUESTED

RETCON_ANALYZED

RETCON_APPLIED

AUDIO_GENERATED

AUDIO_REPLACED
```

---

# 102. AI Is Not a Fake User

Không tạo:

```text
ai@system
system@local
bot@local
```

trong users table chỉ để điền FK.

AI event có thể:

```text
actor_user_id = NULL

generation_run_id = run_123
```

Admin event:

```text
actor_user_id = admin_uuid
```

---

# 103. Audit vs Provenance

Hai khái niệm khác nhau.

Audit:

> Ai thực hiện hành động?

Provenance:

> Dữ liệu được sinh ra từ đâu?

Ví dụ:

```text
created_by:
Admin A

source:
Memory Extractor

generation_run:
run_123

source_chapter:
Chapter 73
```

Có thể cùng tồn tại.

---

# 104. Provenance Fields

Tùy entity có thể có:

```text
source_type

source_chapter_id

source_segment

source_entity

generation_run_id

canon_version_id
```

---

# 105. Traceability Chain

Mục tiêu có thể trace:

```text
HTTP Request
      ↓
Generation Run
      ↓
Generation Jobs
      ↓
Context Snapshot
      ↓
AI Provider Calls
      ↓
Generated Output
      ↓
Admin Action
      ↓
Memory Extraction
      ↓
Canon Commit
```

---

# 106. GenerationRun

Một GenerationRun gom nhiều jobs thuộc cùng intent.

Ví dụ:

```text
Chapter Generation Run #17

├── Chapter Plan
├── Story Writer
├── Continuity Review
├── Quality Review
├── Rewrite
└── Memory Extraction
```

Giúp:

```text
audit
retry
debug
cost tracking
observability
reproducibility
```

---

# 107. Background Work

Các task dài không nên nằm trong HTTP request kéo dài.

Flow:

```text
HTTP Request
     ↓
Create GenerationRun
     ↓
Create GenerationJob
     ↓
Return Job/Run ID
     ↓
Background Processor
```

---

# 108. Initial Job System

Giai đoạn đầu:

```text
PostgreSQL-backed jobs
```

Không thêm:

```text
Kafka
RabbitMQ
Redis
NATS
```

chỉ vì “sau này có thể cần”.

Exact claiming/retry semantics sẽ thiết kế ở Job System Specification.

---

# 109. Future Task Queue

Khi có nhu cầu:

```text
dedicated task queue

priority

retry/backoff

dead-letter

scheduling

worker pools
```

Có thể cân nhắc:

```text
Redis

RabbitMQ

NATS

managed queue
```

tùy requirements lúc đó.

---

# 110. Initial Backend Architecture

Dùng:

```text
MODULAR MONOLITH
```

không microservice từ đầu.

Logical modules:

```text
Go Application
│
├── HTTP/API
│
├── Story
│
├── Planning
│
├── Canon
│
├── Memory
│
├── Generation
│
├── Audio
│
├── User
│
├── Listening
│
├── Audit
│
└── Infrastructure
```

---

# 111. Initial Source Boundary

Concept:

```text
internal/
  story/
  planning/
  canon/
  memory/
  generation/
  audio/
  user/
  listening/
  audit/

  platform/
    postgres/
    storage/
    ai/
    tts/
```

Exact Go folder structure sẽ khóa khi implementation spec bắt đầu.

---

# 112. Provider Abstraction

Business logic không phụ thuộc trực tiếp một vendor.

Story AI abstraction:

```text
StoryAI
```

TTS:

```text
TTSProvider
```

Storage:

```text
ObjectStorage
```

Story Memory:

```text
StoryMemory
```

---

# 113. Story Memory Logical Boundary

Memory subsystem có thể gồm:

```text
ContextBuilder

MemoryRetriever

HistoricalRetriever

FactResolver

ContextSnapshot
```

Generation layer không biết trực tiếp:

```text
SQL details

pgvector details

embedding provider
```

---

# 114. Microservice Strategy

V1:

```text
Modular Monolith
```

Không tách Story Memory thành microservice ngay.

Nhưng logical boundary phải được giữ để future extraction dễ.

---

# 115. Microservice Trigger

Chỉ cân nhắc tách khi có lý do như:

```text
independent scaling

complex hybrid retrieval

many worker consumers

different technology stack

failure isolation
```

Không tách chỉ để:

> “project có microservice”.

---

# 116. Worker Separation Before Memory Microservice

Evolution dự kiến:

## Phase 1

```text
Go Application
├── API
└── Worker
```

## Phase 2

```text
Go API
   │
DB / Queue
   │
Generation Worker
```

## Phase 3

Nếu thực sự cần:

```text
API

Generation Workers

Audio Workers

Story Memory Service

Task Queue
```

---

# 117. Local Development

Docker Compose được sử dụng **ngay từ đầu**.

Local environment phải độc lập production.

---

# 118. Local Docker Stack

Baseline:

```text
Docker Compose
│
├── PostgreSQL
│
├── MinIO
│
├── Go API
│
├── Worker / in-process worker
│
└── Vue optional
```

---

# 119. Local PostgreSQL

Local development sử dụng PostgreSQL container.

Persistent volume lưu:

```text
Stories

Chapters

Canon

Facts

Characters

Jobs

Users

Audit
```

Restart container không mất data nếu volume vẫn còn.

---

# 120. Local Object Storage

Production:

```text
Cloudflare R2
```

Development:

```text
MinIO
```

Lý do:

```text
S3-compatible

local persistent data

không làm bẩn production bucket

không phụ thuộc network cho storage testing
```

---

# 121. Local Audio Pipeline

```text
TTS
 ↓
temporary file
 ↓
FFmpeg
 ↓
MinIO
```

---

# 122. Development AI Modes

## Real Provider Mode

Ví dụ:

```text
AI_PROVIDER=gemini

TTS_PROVIDER=gemini
```

Data vẫn local.

Chỉ inference đi Internet.

Dùng để test:

```text
prompt

voice

real pipeline
```

---

## Mock Provider Mode

```text
AI_PROVIDER=mock

TTS_PROVIDER=mock
```

Không gọi external inference.

Dùng test:

```text
frontend

API

database

job flow

storage

audio player

workflow
```

mà không đốt quota.

---

# 123. Future Local AI Provider

Architecture không được cấm:

```text
AI_PROVIDER=local
```

hoặc provider local như Ollama/OpenAI-compatible local service.

Không phải MVP requirement.

---

# 124. Configuration Precedence

Global:

```text
Code Default
    ↓
Environment Variable
```

Story creation:

```text
Resolved Global Default
    ↓
Create Story UI
    ↓
Advanced Settings
    ↓
CREATE
    ↓
Immutable Story Policy Snapshot
```

---

# 125. No Scattered `if production`

Không viết:

```go
if production {
    useR2()
} else {
    useMinio()
}
```

khắp business code.

Bootstrap/configuration layer chọn adapter.

Ví dụ:

```text
STORAGE_PROVIDER=minio
      ↓
MinIOStorage
```

Production:

```text
STORAGE_PROVIDER=r2
      ↓
R2Storage
```

Application layer chỉ biết:

```text
ObjectStorage
```

Tương tự:

```text
StoryAI
TTS
Queue
```

---

# 126. Development Safety Guard

Local development mặc định **không được** kết nối production DB/storage.

Default:

```text
ALLOW_REMOTE_DATABASE_IN_DEV=false

ALLOW_REMOTE_STORAGE_IN_DEV=false
```

Nếu:

```text
APP_ENV=development
```

nhưng config trỏ production remote data:

```text
startup should fail
```

trừ khi developer chủ động explicit override.

---

# 127. Production Safety Guard

Production cũng cần validation.

Ví dụ có thể reject:

```text
AI_PROVIDER=mock

TTS_PROVIDER=mock

local-only storage

unsafe debug mode
```

---

# 128. Environment Separation

Hỗ trợ:

```text
LOCAL DEVELOPMENT

PRODUCTION-LIKE LOCAL

PRODUCTION
```

Local data không được làm bẩn:

```text
Neon production

R2 production

Production Canon

Production Users
```

---

# 129. Migration Principle

Local và Production phải dùng:

```text
same migration history
```

Không có chuyện:

```text
local schema
≠
production schema
```

về logical schema.

---

# 130. Development Seed

Dev có thể có:

```text
demo admin

demo story

demo chapters

mock audio
```

Production không chạy development seed.

Reference data như Genres sẽ có strategy riêng khi physical DB được chốt.

---

# 131. Production Infrastructure Baseline

```text
Vue
→ Vercel

Go
→ Render

PostgreSQL
→ Neon

Object Storage
→ Cloudflare R2

AI / TTS
→ External Providers
```

---

# 132. Render Health Check

Có:

```text
GET /healthz
```

Response nhẹ:

```json
{
  "status": "ok"
}
```

Không:

```text
query Neon

call R2

call AI

call TTS
```

trong basic health endpoint.

Có thể có future:

```text
/readyz
```

cho dependency readiness.

---

# 133. Local vs Production Storage

Production filesystem của backend không được coi là persistent.

Temporary use:

```text
/tmp
```

Flow:

```text
Generate
 ↓
Temp File
 ↓
FFmpeg
 ↓
Upload R2
 ↓
Delete Temp
```

---

# 134. Prompt Versioning

Prompt generation nên version-controlled.

Initial approach:

```text
prompts/

story-architect.md

arc-planner.md

chapter-plan.md

chapter-write.md

continuity-review.md

quality-review.md

rewrite.md

memory-extract.md

narration.md

tts-director.md
```

Git cung cấp:

```text
history

diff

review

rollback
```

Dynamic prompt database có thể nghiên cứu sau.

---

# 135. Future Architecture Direction

Spec phải mô tả target architecture ngay cả khi implementation chưa tới đó.

Target có thể tiến hóa thành:

```text
Web
 │
API Service
 │
 ├── Story Core
 ├── User Domain
 └── Job System
        │
     Task Queue
        │
 ┌──────┴──────────┐
 │                 │
AI Workers     Audio Workers
 │                 │
Story Memory      FFmpeg
 │                 │
PostgreSQL      Object Storage
```

Possible future components:

```text
API Service

Story Generation Service

Story Memory Service

Audio/TTS Service

Worker Pool

Task Queue

Notification Service
```

Đây là roadmap, không phải MVP deployment requirement.

---

# 136. Core Target Capability List

Target architecture phải support:

```text
Story Bible

Story Arc

Story Ending Plan

Character

Character State

Chapter Plan

Chapter

Story Fact

Plot Thread

Creative Decision

Canon Version

Story Memory Engine

Context Builder

Context Snapshot

Historical Retrieval

Generation Run

Generation Job

Duration Quality Gate

Content Gap Analyzer

Continuity Reviewer

Quality Reviewer

Memory Extractor

Retcon Impact Analysis

Audit

Provenance

Narration

TTS

Audio Versioning

Environment Isolation

Docker Compose
```

---

# 137. MVP / Early Implementation Direction

MVP tập trung trước vào:

```text
Docker Compose foundation

PostgreSQL local

MinIO local

Go Modular Monolith

Vue frontend

Story domain

Story Generation Policy

Story Workflow Settings

Story Bible

Story Arc

Character

Chapter Plan

Chapter

Story Fact

Plot Thread

Generation Run

Generation Job

Story Architect

Arc Planner

Chapter Planner

Story Writer

Continuity Review

Quality Review

Memory Extractor

Context Builder v1

Narration

TTS abstraction

Gemini TTS integration

FFmpeg

AudioAsset

Listener player

Audit Log
```

---

# 138. Phase 2 Direction

Sau baseline có thể triển khai:

```text
semantic memory

pgvector

advanced historical retrieval

context ranking

full Retcon Impact Analyzer

repair planning

advanced Provisional Canon batch generation

generation dashboard

provider fallback

worker separation

task queue nếu justified
```

---

# 139. Scale Phase

Khi tải/complexity thực sự yêu cầu:

```text
Story Memory microservice

dedicated generation workers

dedicated audio workers

hybrid retrieval

reranking

advanced observability

distributed tracing

autoscaling

notification subsystem
```

---

# 140. Future Product Capabilities

Target system không đóng cửa với:

```text
Human-written Story import

AI-assisted writer mode

Multiple narrator/character voices

Subscriptions

Payments

Creator tools

Recommendations

Native mobile applications

Multilingual narration
```

---

# 141. Non-functional Principles

## Reliability

Generation phải:

```text
retryable

recoverable

auditable

stateful

idempotent where applicable
```

---

## Canon Integrity

```text
Draft cannot mutate Official Canon.

Memory extraction occurs after approved content.

Canon commit is transactional.

Major creative changes require decision gate.

Retcon requires impact analysis.
```

---

## Maintainability

Cho phép thay:

```text
Gemini
→ provider khác

MinIO
→ R2

PostgreSQL job queue
→ dedicated task queue

in-process Memory
→ remote Memory Service
```

mà không rewrite domain core.

---

## Security Principle

Secrets:

```text
database credentials

AI API keys

TTS keys

R2 secrets
```

chỉ nằm backend/environment.

Không đưa vào Vue bundle.

Security chi tiết sẽ được spec ở phase riêng.

---

## Cost Control

External generation call nên có khả năng trace:

```text
provider

model

input size

output size

duration

generation run

estimated cost
```

Mock provider hỗ trợ development không tốn quota.

---

# 142. What Is Explicitly NOT Finalized Yet

Các phần sau **chưa được xem là đã chốt chi tiết** trong tài liệu này:

```text
Authentication implementation

Session/JWT strategy

Refresh token strategy

OAuth/social login

Detailed RBAC/permission matrix

Exact REST API contract

HTTP status/error contract

Pagination/filtering contract

Physical PostgreSQL schema

Exact column types

Enum vs CHECK/VARCHAR

Exact foreign keys

ON DELETE behavior

Indexes

Partial indexes

Migration library

Exact pgvector schema

Job claiming algorithm

Job lease/heartbeat

Retry/backoff policy

Dead-letter design

Frontend screen design

Admin Control Center detailed UX

Audio engineering exact LUFS/sample-rate parameters

Testing strategy

AI regression testing

CI/CD

Observability implementation

Security hardening details

Backup/Disaster Recovery details

Audit retention period
```

Những phần này phải được tiếp tục bàn và chốt trước khi coi toàn bộ System Specification là hoàn chỉnh.

---

# 143. Architecture Philosophy

Các nguyên tắc xuyên suốt:

### 1. Chất lượng nội dung quan trọng hơn việc chỉ đạt metric.

Ví dụ chapter thiếu duration không được kéo dài bằng filler.

### 2. AI sáng tạo nhưng không được tự phá Canon.

### 3. Admin có authority cuối cùng cho major creative decisions.

### 4. Story Memory thuộc hệ thống, không phụ thuộc model memory.

### 5. Raw history luôn được giữ.

### 6. Context đưa cho model phải có selection/retrieval có chủ đích.

### 7. Story Canon phải traceable và versionable.

### 8. Retcon được hỗ trợ nhưng phải khó thực hiện và có cảnh báo mạnh.

### 9. Local development phải độc lập production.

### 10. Bắt đầu bằng Modular Monolith, chỉ tách microservice khi có lý do.

### 11. Spec phải mô tả cả target architecture và implementation roadmap.

### 12. Những capability chưa làm ngay vẫn phải được ghi lại để dự án luôn có hướng phát triển tiếp theo.

---

# 144. Current Design Status

Các nhóm kiến trúc sau hiện được xem là đã thống nhất ở mức product/architecture:

```text
Product direction

Long/short story strategy

Chapter duration contract

Immutable Story Policy

Mutable Workflow Settings

FINITE / OPEN_ENDED planning

Story Bible

Story Arc

Character Static/Dynamic State

Chapter Planning

Creative Decision System

Canon

Story Facts

Plot Threads

Story Memory Engine

Context Builder

Historical Retrieval strategy

AI quality gates

Memory Extraction

Batch generation concept

Provisional Canon

Retcon philosophy

Audit

Provenance

Audio/TTS baseline

Object storage strategy

Docker Compose development

Local/Production isolation

Modular Monolith baseline

Worker/Queue/Microservice evolution direction
```

Đây là nền tảng để tiếp tục thiết kế:

```text
Identity
Authentication
Authorization
RBAC
API Contract
Physical Database
Jobs
Admin UX
Listener UX
Testing
Security
Operations
Deployment
```

mà không phải thiết kế lại Story Engine từ đầu.
