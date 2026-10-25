# Current submissions and duplicate records

R13 decisions approved on 2026-10-26. Implementation status and measured checks
are in [development-checklist.md](development-checklist.md) and
[quality-baseline.md](quality-baseline.md).

## One mutable submission

Each student/assignment has one current submission: the record with the greatest
`Submission.CreatedAt`, then descending UUID for ties or unknown creation times.
A repeated POST upload replaces that record's attachment instead of creating a
new record. Its identity, grades and comments remain under R6. Student PUTs to
older records are rejected; older records remain readable history. No historical
row or retained storage object is deleted.

Current summaries, both CSV exports, bulk ZIP downloads and automatic grading exclude historical
records. Detaching the current file does not bring an older file back into use.
This supersedes R8's selection among multiple file-bearing attempts for legacy
duplicates. R8's upload-time lateness, inclusive deadlines, Bangkok display,
unknown-timestamp handling and extension timing remain unchanged. Historical
manual grades still count for R9's maximum-reduction protection; manual grading
permissions and R6's grade-preservation policy are unchanged.

Statistics count each active student once: Submitted has a current file,
Incomplete has a current record without a file, and Not Started has no records.
These categories describe recorded activity, not time remaining. Detail/write
DTOs add `is_current`; summary rows add `has_grade` so absent grades are distinct
from a valid zero. Counts and CSV totals do not use the persisted int16 score
width. CSV student/assignment order and ZIP student order are deterministic. Submission rate
is Submitted divided by active students. Graded/ungraded and score statistics use
only the current file-bearing record; an explicitly graded zero is a valid score.

The detail page opens the current record. Students can choose earlier records in
Submission history; their content and visible feedback remain readable while edit
controls and historical PUTs are blocked. Membership conflicts show a resolution
message. Frontend state/grade tests run with `npm run test:submissions` using Node
24, alongside existing datetime tests; no test dependency was added.

## Memberships and extensions

Identical membership role/status duplicates collapse in reads. Conflicting rows
deny that user's course access until resolved; nonowner file access cannot use
one conflicting row to obtain another role's powers. Any separately authorized
reference in a different, consistent course still grants access under R10.
Uploader access to the uploader's own files remains independent of membership.
Course discovery excludes ambiguous memberships. Summaries and both exports return
409 for any conflicting membership instead of silently changing their denominator.

The roster retains distinct conflicting role/status rows for staff inspection.
An authorized teacher may update both role and status through the existing member
endpoint, which updates all matching rows. An affected teacher needs another
authorized teacher to resolve their membership; this does not create a bypass.
Concurrent enrollment serializes on the course row and preserves existing inactive
enrollment rather than silently reactivating it.

`AssignmentOverride` has no creation/update timestamps. The final approved rule
selects the greatest UUID per student/assignment and updates that row in place.
UUID order is a deterministic tie-break, not historical chronology. Other legacy
override rows remain unchanged. List/detail DTOs, admission, summary and exports
must use the same row. Extension writes serialize on the assignment row.

## Constraints and future schema review

No persistence fields, indexes, database migrations or historical cleanup are
added by R13. The is_current/has_grade fields are computed response metadata.
Course/assignment row locks serialize supported application writes. They do not
protect against direct SQL, imports or old writers that bypass these paths.
Canonical selection and conflict checks remain necessary for legacy data.

A future unique membership or extension index requires an inventory, approved
duplicate-resolution plan, backup, migration/rollback review and explicit
authorization. A naive unique student/assignment submission index would reject
the history retained by this policy; enforcing one current record in the schema
would require a separately reviewed history/current representation. Do not add
GORM unique tags that AutoMigrate would apply at application startup.
