# Teaching assistant permissions

Decision: 2026-10-25. The user delegated selection of TA permissions: support
routine teaching like a teacher, with fewer critical powers, and record each
choice and its reason. R11 implements the choices below. Teacher/student grants
are preserved; admin and unknown roles receive no new grants.

## Scope and safeguards

Course actions use the **active course membership's role**, independent of the
global user role. A global teacher who is a TA in a course has TA powers there.
A global TA who is a student in a course has student powers there. Global role
continues to govern course creation/listing, personal templates/attachments, AI
credentials/jobs and Python. Course membership does not grant or revoke independent
global teacher powers. Login still assigns global student/teacher roles using the
existing university mapping; no global-role administration is added here.

TAs may read hidden/upcoming assignments, all submissions and feedback in their
active courses, summaries, and individual referenced files. A TA cannot access
other courses through guessed IDs. R10's uploader exception and any-authorized-
reference policy remain in force. Individual read access permits saving those
bytes; denying bulk ZIP/CSV limits bulk operations, not copying of visible data.

TAs may grade only fully ungraded work (both score and grader null) or replace an
AI-attributed grade. Teachers retain authority to change any manual grade. TAs
cannot replace teacher, TA (including their own), unknown, empty-grader, or
unattributed existing grades. A teacher handles corrections to manual grades.
Authority is rechecked while holding assignment then submission locks, so a stale
TA request cannot overwrite a concurrent human grade. Existing bounds still apply.
AI cannot overwrite the new human TA grade under R4.

TAs can create/update their own feedback, including its visibility, independently
of grade eligibility. Toggling feedback requires matching author ID **and** TA
role; another TA's, teacher's or AI feedback is protected. Teachers retain
`assignment:comment_any`. Feedback visibility/release defaults are unchanged.
Grade-and-comment runtime failure atomicity remains R12.

## Every permission choice

“Yes” applies in the scope above. Missing grants fail closed; new permission names
require a deliberate decision. Some existing names have no live endpoint.

| Permission | TA | Reason |
| --- | --- | --- |
| `course:create` | No | Creating courses and assigning their teacher stays with global teachers. |
| `course:view_own` | Yes | A global TA can list courses with active membership. |
| `course:view_all` | No | No institution-wide course discovery; this existing checked name has no grant. |
| `course:update` | No | Course identity, schedule and section settings belong to teachers. |
| `course:delete` | No | Cascading destruction of course work is critical. |
| `course:enroll` | No | Enrollment can assign roles; teachers control access. |
| `course:export` | No | Bulk course grades/student data require teacher authority. |
| `member:view_all` | Yes | Identify students for support/grading using the existing roster fields. |
| `member:update` | No | Prevent self-promotion and changes to others' role/status. |
| `member:delete` | No | Removing access or enrollment is a teacher decision. |
| `assignment:create` | No | Publishing assessed work and choosing its grading settings stays with teachers. |
| `assignment:view_all` | Yes | Prepare support before release and inspect assignment summaries. |
| `assignment:view_visible` | No separate grant | Covered by broader assignment view access; student filtering does not apply to TAs. |
| `assignment:update` | No | The existing gate also changes deadlines, maxima, visibility, AI settings and overrides. |
| `assignment:delete` | No | Deleting assignments risks student files and grades. |
| `assignment:comment` | Yes | Provide/update authored teaching feedback. |
| `assignment:comment_own` | Yes | Control visibility of one's own feedback only. |
| `assignment:comment_any` | No | Protect feedback authored by other people and AI. |
| `assignment:grade` | Yes, restricted | Initial grading and AI review are routine TA work; manual-grade corrections stay with teachers. |
| `assignment:export` | No | Bulk grade and identity exports stay with teachers. |
| `submission:create` | No | TAs must not submit as students through their staff role. |
| `submission:view_all` | Yes | Review all attempts/feedback in the authorized course. |
| `submission:view_own` | No separate grant | Staff review uses all-submission access, not the student's override path. |
| `submission:update` | No | Preserve student authorship and uploaded evidence. |
| `submission:delete` | No | Preserve student evidence; this existing name currently has no registered delete route. |
| `template:create` | No | Personal reusable authoring/upload operations stay with global teachers. |
| `template:view` | No | Course staff access must not imply personal template/library access. |
| `template:update` | No | Personal reusable content remains teacher-managed. |
| `template:delete` | No | Avoid destructive personal library/storage operations. |
| `file:read_all` | Yes | Read/download individual handouts and student files through active course references. |
| `file:read_own` | No separate grant | Broader scoped staff reads and the independent uploader exception cover access. |
| `file:download_all` | No | Bulk student-source ZIPs require teacher authority. |
| `run:python` | No | External code execution/resource use stays with global teachers. |
| `ai` | No | Provider credentials, spend, job control and batch regrading stay with teachers. |

## Verification and limits

`backend/internal/handler/ta_permissions_test.go` exercises actual router, auth,
course/submission guards and disposable PostgreSQL. It covers hidden/upcoming
reads, roster/summaries, active membership and parent scoping, global/course role
differences, both grade routes across teacher/TA/student roles and prior grader
states, feedback ownership, denied critical operations, and real concurrent human
grading. Rejections preserve data and avoid storage/provider/queue calls.

Backend permissions are authoritative. Existing frontend controls are not a
complete permission UI: some use global teacher checks or treat every nonstudent
as staff; the grading panel combines grade/comment submission and identifies
feedback by role rather than author ID. They may show actions the API rejects or
require a separate comment request after a manual grade. A focused UI alignment
task is tracked in the checklist; this change does not expand frontend scope.
Duplicate membership precedence, feedback publication policy and job recovery
remain separate decisions/tasks. No existing role assignments are changed.
