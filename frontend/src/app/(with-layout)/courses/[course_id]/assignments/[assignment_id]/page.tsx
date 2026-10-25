"use client"

import { useEffect, useState } from "react"
import { useParams } from "next/navigation"
import { useRouter } from "next/navigation"
import { useAuth } from "@/lib/auth-context"
import { canEditSubmission, mergeCurrentSubmission, selectCurrentSubmission } from "@/lib/submission-record"
import { BreadcrumbNav } from "@/components/breadcrumb-nav"
import { AssignmentInfo } from "./assignment-info"
import { SubmissionList } from "./submission-list"
import { CommentGrade } from "./comment-grade"
import { AnswerBox } from "./answer-box"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription } from "@/components/ui/empty"
import { AlertCircle, X, MessageSquare } from "lucide-react"
import { AssignmentActions } from "./assignment-actions"
import { CodeSection } from "./code-section"
import { GradingPanel } from "./grading-panel"
import { CommentSection } from "./comment-section"


type Attachment = {
  id: string
  file_name: string
  file_type: string
  size: number
  created_at: string
}

type Assignment = {
  id: string
  title: string
  description: string
  point: number
  attachments: Attachment[]
  start_date: string
  due_date: string
  close_date: string
  tags: string[]
  ai_agent: boolean
  ai_config_id?: string
  assignment_prompt: string
  visible: boolean
}

type Comment = {
  id: string
  comment: string
  commentator: string
  visible: boolean
}

type Submission = {
  is_current?: boolean
  id: string
  submitter: string
  answer: string
  point: number
  graded_by: string
  attachment_id: string
  file_name: string
  comments: Comment[]
}

export default function AssignmentDetailPage() {
  const { course_id, assignment_id } = useParams()

  const { user } = useAuth()
  const router = useRouter()

  const [courseRole, setCourseRole] = useState<string | undefined>();
  const [courseName, setCourseName] = useState("")
  const [assignment, setAssignment] = useState<Assignment | null>(null)
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [selectedSubmission, setSelectedSubmission] = useState<Submission | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isStudent, setIsStudent] = useState(true)
  const [isAnonymous, setIsAnonymous] = useState(false)
  const [gradingError, setGradingError] = useState<string | null>(null)
  const [isGrading, setIsGrading] = useState(false)
  const [isEvaluate, setIsEvaluate] = useState(false)
  const [submissionContent, setSubmissionContent] = useState("")
  const [isContentLoading, setIsContentLoading] = useState(false)
  const [contentError, setContentError] = useState<string | null>(null)
  const [isGradingPanelCollapsed, setIsGradingPanelCollapsed] = useState(false)
  const [contentRefreshKey, setContentRefreshKey] = useState(0)

  useEffect(() => {
    if (!user?.token) {
      router.push("/")
      return
    }

    const fetchData = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const courseRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}`, {
          headers: { Authorization: `Bearer ${user.token}` },
        })
        if (courseRes.ok) {
          const courseData = await courseRes.json()
          setCourseName(courseData.name || "")
        }

        const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/${assignment_id}`, {
          headers: { Authorization: `Bearer ${user.token}` },
        })
        if (!res.ok) throw new Error(res.status === 409 ? "Course membership data conflicts. Ask a teacher to resolve your enrollment." : "Failed to fetch assignment")
        const data = await res.json()
        setAssignment(data.assignment)
        setSubmissions(data.submissions || [])
        setIsStudent(data.role == "student")
        setCourseRole(data.role)

        const records: Submission[] = data.submissions || []
        setSelectedSubmission(data.role === "student" ? selectCurrentSubmission(records) : records[0] ?? null)

      } catch (err) {
        setError(err instanceof Error ? err.message : "An error occurred")
      } finally {
        setIsLoading(false)
      }
    }

    fetchData()
  }, [user, course_id, assignment_id, router])

  const handleSelectSubmission = (submission: Submission) => {
    setSelectedSubmission(submission)
  }

  const selectedSubmissionIndex = selectedSubmission
    ? submissions.findIndex((submission) => submission.id === selectedSubmission.id)
    : -1

  const handlePreviousSubmission = () => {
    if (selectedSubmissionIndex <= 0) return
    setSelectedSubmission(submissions[selectedSubmissionIndex - 1])
  }

  const handleNextSubmission = () => {
    if (selectedSubmissionIndex < 0 || selectedSubmissionIndex >= submissions.length - 1) return
    setSelectedSubmission(submissions[selectedSubmissionIndex + 1])
  }

  const handleEnterEvaluateMode = () => {
    setIsEvaluate(true)
  }

  const handleLeaveEvaluateMode = () => {
    setIsEvaluate(false)
  }

  useEffect(() => {
    if (!user?.token || !selectedSubmission?.id) {
      setSubmissionContent("")
      setContentError(null)
      setIsContentLoading(false)
      return
    }

    let cancelled = false

    const fetchSubmissionContent = async () => {
      setIsContentLoading(true)
      setContentError(null)

      try {
        const response = await fetch(
          `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/${assignment_id}/submissions/${selectedSubmission.id}/read`,
          {
            method: "GET",
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${user.token}`,
            },
          }
        )

        if (!response.ok) {
          throw new Error("Failed to fetch answer content")
        }

        const content = await response.json()
        if (!cancelled) {
          setSubmissionContent(content)
        }
      } catch (err) {
        if (!cancelled) {
          setContentError(err instanceof Error ? err.message : "An error occurred")
        }
      } finally {
        if (!cancelled) {
          setIsContentLoading(false)
        }
      }
    }

    fetchSubmissionContent()

    return () => {
      cancelled = true
    }
  }, [user, course_id, assignment_id, selectedSubmission?.id, contentRefreshKey])

  const handleSubmitGrade = async (
    grade: number,
    comment: string,
    visible: boolean
  ) => {
    if (!selectedSubmission || !user) return
    setIsGrading(true)
    setGradingError(null)

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/${assignment_id}/submissions/${selectedSubmission.id}/grade-with-comment`,
        {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${user.token}`,
          },
          body: JSON.stringify({
            point: grade,
            comment,
            visible,
          }),
        }
      )

      if (!response.ok) {
        throw new Error("Failed to submit grade")
      }

      const result = await response.json()
      const normalizedComment = (result.comment ?? "").trim()
      setSelectedSubmission((prev) => {
        if (!prev) return prev

        const newComment = {
          id: result.comment_id,
          comment: normalizedComment,
          commentator: result.graded_by,
          visible: result.visible,
        }

        const exists = prev.comments.some(
          (c) => c.commentator === newComment.commentator
        )

        const nextComments = normalizedComment
          ? exists
            ? prev.comments.map((c) =>
              c.commentator === newComment.commentator ? newComment : c
            )
            : [newComment, ...prev.comments]
          : prev.comments.filter((c) => c.commentator !== newComment.commentator)

        return {
          ...prev,
          point: result.point,
          graded_by: result.graded_by,
          comments: nextComments,
        }
      })
      setSubmissions((prevList) =>
        prevList.map((s) => {
          if (s.id !== result.submission_id) return s;

          const normalizedComment = (result.comment ?? "").trim()
          const newComment = {
            id: result.comment_id,
            comment: normalizedComment,
            commentator: result.graded_by,
            visible: result.visible,
          };

          const exists = s.comments.some(
            (c) => c.commentator === newComment.commentator
          );

          const nextComments = normalizedComment
            ? exists
              ? s.comments.map((c) =>
                c.commentator === newComment.commentator ? newComment : c
              )
              : [newComment, ...s.comments]
            : s.comments.filter((c) => c.commentator !== newComment.commentator)

          return {
            ...s,
            point: result.point,
            graded_by: result.graded_by,
            comments: nextComments,
          };
        })
      );
    } catch (err) {
      setGradingError(
        err instanceof Error ? err.message : "An error occurred"
      )
    } finally {
      setIsGrading(false);
    }
  }

  const handleSubmit = async (file?: File | null) => {
    if (!user?.token) return

    if (!canEditSubmission(selectedSubmission, submissions.length)) throw new Error("Historical submissions are read-only")

    const isUpdate = !!selectedSubmission?.id

    // For new submissions, file is required
    if (!isUpdate && !file) return

    const formData = new FormData()
    if (file) {
      formData.append("file", file)
    }

    try {
      const url = `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/${assignment_id}/submissions${isUpdate ? `/${selectedSubmission.id}` : ""}`

      const response = await fetch(url, {
        method: isUpdate ? "PUT" : "POST",
        headers: {
          Authorization: `Bearer ${user.token}`,
        },
        body: formData,
      })

      if (!response.ok) {
        throw new Error(isUpdate ? "Failed to update submission" : "Failed to submit assignment")
      }

      const newSubmission = await response.json()

      setSubmissions((previous) => mergeCurrentSubmission(previous, newSubmission))

      // Set as selected submission
      setSelectedSubmission(newSubmission)
      setContentRefreshKey((k) => k + 1)
    } catch (err) {
      console.error("Submission error:", err)
      throw err // Re-throw so the AnswerBox component can handle the error
    }
  }

  // Student has valid submission only if attachment exists
  const studentHasSubmission = isStudent && selectedSubmission && selectedSubmission.attachment_id

  const isAssignmentClosed = assignment ? new Date() > new Date(assignment.close_date) : false

  if (isLoading) {
    return (
      <div className="space-y-6 p-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <Skeleton className="h-64 w-full" />
          </div>
          <div className="space-y-6">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-64 w-full" />
          </div>
        </div>
      </div>
    )
  }

  if (error || !assignment) {
    return (
      <div className="p-6">
        <Empty className="py-16">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <AlertCircle className="h-5 w-5" />
            </EmptyMedia>
            <EmptyTitle>Failed to load assignment</EmptyTitle>
            <EmptyDescription>{error || "Assignment not found"}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className={isEvaluate ? "flex h-screen flex-col overflow-hidden" : "space-y-6"}>
      {/* Breadcrumb */}
      <BreadcrumbNav courseName={courseName} assignmentName={assignment.title} />
      {isStudent && submissions.length > 1 && (
        <div className="mx-6 space-y-2">
          <label htmlFor="submission-record" className="text-sm font-medium">Submission history</label>
          <select id="submission-record" className="block w-full rounded-md border bg-background p-2 text-sm"
            value={selectedSubmission?.id ?? ""}
            onChange={(event) => setSelectedSubmission(submissions.find((record) => record.id === event.target.value) ?? null)}>
            {submissions.map((record, index) => (
              <option key={record.id} value={record.id}>
                {record.is_current ? "Current submission" : `History ${index}`} — {record.file_name || "No attachment"}
              </option>
            ))}
          </select>
        </div>
      )}
      {selectedSubmission && !selectedSubmission.is_current && (
        <p className="mx-6 rounded-md border p-3 text-sm text-muted-foreground">
          Historical submission. Excluded from current reports and automatic grading.
          {isStudent && " This record is read-only."}
        </p>
      )}
      {isEvaluate && !isStudent ? (
        <div className="flex-1 flex min-h-0 overflow-hidden">
          <div className={isGradingPanelCollapsed ? "flex-1" : "w-3/4 min-w-0"}>
            <CodeSection
              submissionId={selectedSubmission?.id || ""}
              submissions={submissions}
              hasSubmission={!!selectedSubmission}
              isAnonymous={isAnonymous}
              toggleAnonymous={setIsAnonymous}
              leaveEvaluate={handleLeaveEvaluateMode}
              onPreviousSubmission={handlePreviousSubmission}
              onNextSubmission={handleNextSubmission}
              onSelectSubmission={(submissionId) => {
                const submission = submissions.find((item) => item.id === submissionId)
                if (submission) setSelectedSubmission(submission)
              }}
              canGoPrevious={selectedSubmissionIndex > 0}
              canGoNext={selectedSubmissionIndex >= 0 && selectedSubmissionIndex < submissions.length - 1}
              currentSubmissionOrder={selectedSubmissionIndex >= 0 ? selectedSubmissionIndex + 1 : 0}
              totalSubmissions={submissions.length}
              answerContent={submissionContent}
              isContentLoading={isContentLoading}
              contentError={contentError}
            />
          </div>
          {isGradingPanelCollapsed ? (
            <div className="w-8 flex-none cursor-pointer hover:bg-muted/50 transition-colors flex items-center justify-center border border-border" onClick={() => setIsGradingPanelCollapsed(false)}>
              <MessageSquare className="h-5 w-5 text-muted-foreground" />
            </div>
          ) : (
            <div className="w-1/4 min-w-0">
              <GradingPanel
                key={selectedSubmission?.id ?? "no-submission"}
                submissions={submissions}
                selectedId={selectedSubmission?.id ?? ""}
                onSelect={setSelectedSubmission}
                isAnonymous={isAnonymous}
                role={courseRole}
                comments={selectedSubmission?.comments || []}
                maxPoints={assignment.point}
                currentGrade={selectedSubmission?.point || 0}
                gradedBy={selectedSubmission?.graded_by || ""}
                onSubmitGrade={handleSubmitGrade}
                onToggleCollapse={() => setIsGradingPanelCollapsed(!isGradingPanelCollapsed)}
              />
            </div>
          )}
        </div>
      ) : (
        <div className="grid gap-6 lg:grid-cols-3 px-6">
          {/* Left Section */}
          <div className={`flex flex-col gap-4 ${isStudent ? "lg:col-span-3" : "lg:col-span-2"}`}>
            {/* 1. Assignment Info */}
            <AssignmentInfo assignment={assignment} studentSubmission={isStudent ? selectedSubmission : null} />

            {/* 2. Answer Box */}
            {isStudent ? (
              <div className="grid gap-4 lg:grid-cols-5">
                <div className="lg:col-span-3">
                  <AnswerBox
                    attachmentId={selectedSubmission?.attachment_id}
                    fileName={selectedSubmission?.file_name}
                    isStudent={canEditSubmission(selectedSubmission, submissions.length)}
                    hasSubmission={!!studentHasSubmission}
                    isClosed={isAssignmentClosed}
                    onSubmit={handleSubmit}
                    answerContent={submissionContent}
                    isContentLoading={isContentLoading}
                    contentError={contentError}
                  />
                </div>
                <div className="lg:col-span-2">
                  <CommentSection
                    comments={selectedSubmission?.comments || []}
                  />
                </div>
              </div>
            ) : selectedSubmission ? (
              <AnswerBox
                attachmentId={selectedSubmission.attachment_id}
                fileName={selectedSubmission.file_name}
                isStudent={false}
                hasSubmission={true}
                isClosed={false}
                answerContent={submissionContent}
                isContentLoading={isContentLoading}
                contentError={contentError}
              />
            ) : (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground">
                  Select a submission to view
                </CardContent>
              </Card>
            )}
          </div>
          {/* Right Section */}
          {!isStudent && (
            <div className="flex flex-col gap-4">
              <AssignmentActions
                courseId={`${course_id}`}
                assignment={assignment}
                evaluateMode={handleEnterEvaluateMode}
                onAssignmentUpdated={setAssignment}
              />
              <SubmissionList
                submissions={submissions}
                selectedId={selectedSubmission?.id || null}
                onSelect={handleSelectSubmission}
                maxPoints={assignment.point}
                isAnonymous={isAnonymous}
              />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

{/*{(isStudent && studentHasSubmission) || (!isStudent && selectedSubmission) ? (
  <CommentGrade
    role={courseRole}
    comments={selectedSubmission?.comments || []}
    currentGrade={selectedSubmission?.point || 0}
    maxPoints={assignment.point}
    gradedBy={selectedSubmission?.graded_by || ""}
    isStudent={isStudent}
    onSubmitGrade={handleSubmitGrade}
  />
) : null}*/}