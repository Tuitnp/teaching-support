"use client"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Calendar, Clock, FileText, Tag } from "lucide-react"
import { TipTapTextEditor } from "@/components/ui/tiptap"
import { formatDateTime } from "@/lib/datetime"
import { useAuth } from "@/lib/auth-context"
import { toast } from "sonner"

type Attachment = {
  id: string
  file_name: string
  file_type?: string
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
}

type AssignmentStatus = "closed" | "upcoming" | "overdue" | "active"

interface AssignmentInfoProps {
  assignment: Assignment
  studentSubmission?: Submission | null
}

type Submission = {
  id: string
  submitter: string
  answer: string
  point: number
  graded_by: string
  attachment_id: string
  file_name: string
  comments: Comment[]
}

type Comment = {
  id: string
  comment: string
  commentator: string
  visible: boolean
}

export function AssignmentInfo({ assignment, studentSubmission }: AssignmentInfoProps) {
  const { user } = useAuth()

  const formatDate = formatDateTime

  const handleDownloadAttachment = async (attachment: Attachment) => {
    if (!user?.token) return

    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/attachments/${attachment.id}/download`, {
        headers: { Authorization: `Bearer ${user.token}` },
      })

      if (!res.ok) {
        toast.error("Download failed", {
          description: "Something went wrong. Please try again.",
        })
        return
      }

      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = attachment.file_name
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
    } catch (err) {
      console.error(err)
      toast.error("Download failed", {
        description: "Something went wrong. Please try again.",
      })
    }
  }

  const getStatus = (): AssignmentStatus => {
    const now = new Date()
    const start = new Date(assignment.start_date)
    const due = new Date(assignment.due_date)
    const close = new Date(assignment.close_date)

    if (now < start) return "upcoming"
    if (now > close) return "closed"
    if (now > due) return "overdue"
    return "active"
  }

  const status = getStatus()

  const getStatusBadge = () => {
    switch (status) {
      case "closed":
        return <Badge variant="secondary">Closed</Badge>
      case "upcoming":
        return <Badge variant="outline">Upcoming</Badge>
      case "overdue":
        return <Badge variant="destructive">Overdue</Badge>
      case "active":
        return <Badge className="bg-green-600 hover:bg-green-700">Active</Badge>
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1">
            <CardTitle className="text-xl">{assignment.title}</CardTitle>
            <CardDescription className="mt-2 whitespace-pre-wrap">
              <TipTapTextEditor
                value={assignment.description}
                onChange={() => { }}
                editable={false}
              />
            </CardDescription>
          </div>
          <div className="flex flex-col items-end gap-2">
            {getStatusBadge()}
            {studentSubmission && studentSubmission.graded_by ? (
              <div className="flex items-baseline gap-1">
                <span
                  className={`text-lg font-semibold ${
                    assignment.point > 0
                      ? studentSubmission.point / assignment.point >= 0.8
                        ? "text-green-500"
                        : studentSubmission.point / assignment.point >= 0.5
                          ? "text-yellow-500"
                          : "text-red-500"
                      : "text-muted-foreground"
                  }`}
                >
                  {studentSubmission.point}
                </span>
                <span className="text-sm text-muted-foreground">/ {assignment.point} pts</span>
              </div>
            ) : (
              <span className="text-lg font-semibold">{assignment.point} pts</span>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Dates */}
        <div className="flex flex-wrap gap-6 text-sm">
          <div className="flex items-center gap-2">
            <Calendar className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground">Start:</span>
            <span>{formatDate(assignment.start_date)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <span className={status === "overdue" ? "text-red-600 font-medium" : "text-muted-foreground"}>Due:</span>
            <span className={status === "overdue" ? "text-red-600 font-medium" : ""}>{formatDate(assignment.due_date)}</span>
          </div>
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground">Close:</span>
            <span>{formatDate(assignment.close_date)}</span>
          </div>
        </div>

        {/* Attachments */}
        {assignment.attachments && assignment.attachments.length > 0 && (
          <div className="space-y-2">
            <h4 className="text-sm font-medium">Attachments</h4>
            <div className="flex flex-wrap gap-2">
              {assignment.attachments.map((attachment) => (
                <button
                  key={attachment.id}
                  type="button"
                  onClick={() => handleDownloadAttachment(attachment)}
                  title={`Download ${attachment.file_name}`}
                  className="flex items-center gap-2 rounded-md border px-3 py-2 text-sm hover:bg-muted transition-colors"
                >
                  <FileText className="h-4 w-4" />
                  {attachment.file_name}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Tags */}
        {assignment.tags && assignment.tags.length > 0 && (
          <div className="flex items-center gap-2">
            <Tag className="h-4 w-4 text-muted-foreground" />
            <div className="flex flex-wrap gap-2">
              {assignment.tags.map((tag) => (
                <Badge key={tag} variant="outline">
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
