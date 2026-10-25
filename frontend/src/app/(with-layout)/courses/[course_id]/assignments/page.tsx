"use client"

import { useEffect, useMemo, useState } from "react"
import Link from "next/link"
import { useParams } from "next/navigation"
import { formatDateTime } from "@/lib/datetime"
import { useAuth } from "@/lib/auth-context"
import { BreadcrumbNav } from "@/components/breadcrumb-nav"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "@/components/ui/empty"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  AlertCircle,
  BookOpen,
  Calendar,
  CalendarClock,
  CalendarOff,
  CalendarPlus,
  Download,
  FileText,
  GraduationCap,
  MapPin,
  Search,
  Tag
} from "lucide-react"
import { CreateAssignmentDialog } from "@/components/create-assignment-dialog"

interface Course {
  id: string
  name: string
  course_code?: string
  day_of_week?: string
  start_time?: string
  end_time?: string
  room?: string
  credits?: number
  section: string
  semester: string
  teacher: string
}

interface Assignment {
  id: string
  title: string
  description: string
  point: number
  start_date: string
  due_date: string
  close_date: string
  tags: string[]
}

type AssignmentStatus = "closed" | "upcoming" | "overdue" | "active"

function formatCourseSchedule(course: Course) {
  const time = course.start_time && course.end_time
    ? `${course.start_time} - ${course.end_time}`
    : course.start_time || course.end_time || "-"

  if (course.day_of_week) {
    return `${course.day_of_week} ${time}`
  }

  return time
}

function formatTeacherName(name: string) {
  return name
    .trim()
    .split(/\s+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ")
}

export default function CourseDetailPage() {
  const { course_id } = useParams()
  const { user } = useAuth()
  const [course, setCourse] = useState<Course | null>(null)
  const [assignments, setAssignments] = useState<Assignment[]>([])
  const [searchTerm, setSearchTerm] = useState("")
  const [isLoading, setIsLoading] = useState(true)
  const [isDownloadingAllSubmissions, setIsDownloadingAllSubmissions] = useState(false)
  const [error, setError] = useState("")

  const handleDownloadAllSubmissionsCsv = async () => {
    if (!user?.token) return

    try {
      setIsDownloadingAllSubmissions(true)

      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/export`,
        {
          headers: { Authorization: `Bearer ${user.token}` },
        }
      )

      if (!res.ok) {
        throw new Error("Failed to export submissions")
      }

      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement("a")
      const safeCourseName = (course?.name || "course")
        .trim()
        .replace(/[\\/:*?"<>|]+/g, "_")
        .replace(/\s+/g, "_")

      link.href = url
      link.download = `${safeCourseName}_all_submissions.csv`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (err) {
      console.error("Failed to download submissions csv", err)
      alert("Failed to download submissions csv")
    } finally {
      setIsDownloadingAllSubmissions(false)
    }
  }

  useEffect(() => {
    const fetchData = async () => {
      if (!user?.token) return

      try {
        // Fetch course details
        const courseRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}`, {
          headers: { Authorization: `Bearer ${user.token}` },
        })
        if (!courseRes.ok) throw new Error("Failed to fetch course")
        const courseData = await courseRes.json()
        setCourse(courseData)

        // Fetch assignments
        const assignmentsRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments`, {
          headers: { Authorization: `Bearer ${user.token}` },
        })
        if (!assignmentsRes.ok) throw new Error("Failed to fetch assignments")
        const assignmentsData = await assignmentsRes.json()
        setAssignments(assignmentsData.assignments)
        console.log(assignmentsData)
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load data")
      } finally {
        setIsLoading(false)
      }
    }

    fetchData()
  }, [user, course_id])

  const getAssignmentStatus = (assignment: Assignment): AssignmentStatus => {
    const now = new Date()
    const startDate = new Date(assignment.start_date)
    const dueDate = new Date(assignment.due_date)
    const closeDate = new Date(assignment.close_date)

    if (now > closeDate) {
      return "closed"
    } else if (now < startDate) {
      return "upcoming"
    } else if (now > dueDate) {
      return "overdue"
    } else {
      return "active"
    }
  }

  const getStatusBadge = (status: AssignmentStatus) => {
    switch (status) {
      case "closed":
        return <Badge variant="secondary">Closed</Badge>
      case "upcoming":
        return <Badge className="bg-blue-500/10 text-blue-600 hover:bg-blue-500/20">Upcoming</Badge>
      case "overdue":
        return <Badge className="bg-red-500/10 text-red-600 hover:bg-red-500/20">Overdue</Badge>
      case "active":
        return <Badge className="bg-green-500/10 text-green-600 hover:bg-green-500/20">Active</Badge>
    }
  }

  const formatDate = formatDateTime

  const filteredAssignments = useMemo(() => {
    const normalizedQuery = searchTerm.trim().toLowerCase()

    if (!normalizedQuery) {
      return assignments
    }

    return assignments.filter((assignment) => {
      const titleMatches = assignment.title.toLowerCase().includes(normalizedQuery)
      const tagMatches = assignment.tags.some((tag) => tag.toLowerCase().includes(normalizedQuery))

      return titleMatches || tagMatches
    })
  }, [assignments, searchTerm])

  const hasAssignments = assignments.length > 0
  const hasSearchResults = filteredAssignments.length > 0

  return (
    <>
      <BreadcrumbNav courseName={course?.name} />
      <div className="flex-1 p-6">
        {/* Loading State */}
        {isLoading && (
          <div className="space-y-6">
            <div>
              <Skeleton className="h-8 w-64" />
              <Skeleton className="mt-2 h-4 w-48" />
            </div>
            <div className="grid gap-4">
              {[...Array(3)].map((_, i) => (
                <Card key={i}>
                  <CardHeader>
                    <Skeleton className="h-6 w-3/4" />
                    <Skeleton className="h-4 w-1/2" />
                  </CardHeader>
                </Card>
              ))}
            </div>
          </div>
        )}

        {/* Error State */}
        {error && !isLoading && (
          <Empty className="py-16">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <AlertCircle className="h-5 w-5" />
              </EmptyMedia>
              <EmptyTitle>Failed to load course</EmptyTitle>
              <EmptyDescription>{error}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button onClick={() => window.location.reload()}>Try Again</Button>
            </EmptyContent>
          </Empty>
        )}

        {/* Course Content */}
        {!isLoading && !error && course && (
          <div className="space-y-6">
            {/* Course Header */}
            <div>
              <div className="flex items-center gap-2">
                <BookOpen className="h-6 w-6 text-primary" />
                <h1 className="text-2xl font-bold">{course.name}</h1>
              </div>
              <div className="mt-3 grid gap-2 text-sm text-muted-foreground sm:grid-cols-2 lg:grid-cols-4">
                <div className="flex items-center gap-2">
                  <GraduationCap className="h-4 w-4 shrink-0" />
                  <span>{formatTeacherName(course.teacher)}</span>
                </div>
                {course.course_code && (
                  <div className="flex items-center gap-2">
                    <BookOpen className="h-4 w-4 shrink-0" />
                    <span className="font-medium uppercase tracking-wide text-foreground/90">
                      {course.course_code}
                    </span>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <Calendar className="h-4 w-4 shrink-0" />
                  <span>{formatCourseSchedule(course)}</span>
                </div>
                <div className="flex items-center gap-2">
                  <MapPin className="h-4 w-4 shrink-0" />
                  <span>{course.room ? `Room ${course.room}` : "Room -"}</span>
                </div>
                <Badge variant="outline">Section {course.section}</Badge>
                <Badge variant="secondary">{course.semester}</Badge>
              </div>
            </div>

            {/* Assignments Section */}
            <div>
              <div className="mb-4 flex items-center justify-between">
                <h2 className="text-lg font-semibold">Assignments</h2>
                {user?.role === "teacher" && (
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      onClick={handleDownloadAllSubmissionsCsv}
                      disabled={isDownloadingAllSubmissions}
                    >
                      <Download className="h-4 w-4 mr-2" />
                      {isDownloadingAllSubmissions ? "Downloading..." : "Download all submissions from all assignments"}
                    </Button>
                    <CreateAssignmentDialog
                      courseId={course_id as string}
                      onCreated={(assignment) => {
                        setAssignments(prev => [...prev, assignment])
                      }}
                    />
                  </div>
                )}
              </div>

              <div className="mb-4">
                <div className="relative max-w-md">
                  <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={searchTerm}
                    onChange={(event) => setSearchTerm(event.target.value)}
                    placeholder="Search by title or tag"
                    className="pl-9"
                  />
                </div>
              </div>

              {!hasAssignments ? (
                <Empty className="py-12">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <FileText className="h-5 w-5" />
                    </EmptyMedia>
                    <EmptyTitle>No assignments yet</EmptyTitle>
                    <EmptyDescription>
                      Check back later for new assignments
                    </EmptyDescription>
                  </EmptyHeader>
                </Empty>
              ) : !hasSearchResults ? (
                <Empty className="py-12">
                  <EmptyHeader>
                    <EmptyMedia variant="icon">
                      <Search className="h-5 w-5" />
                    </EmptyMedia>
                    <EmptyTitle>No matching assignments</EmptyTitle>
                    <EmptyDescription>
                      Try a different title or tag.
                    </EmptyDescription>
                  </EmptyHeader>
                  <EmptyContent>
                    <Button variant="outline" onClick={() => setSearchTerm("")}>Clear search</Button>
                  </EmptyContent>
                </Empty>
              ) : (
                <div className="grid gap-4">
                  {filteredAssignments.map((assignment) => {
                    const status = getAssignmentStatus(assignment)
                    return (
                      <Link key={assignment.id} href={`/courses/${course.id}/assignments/${assignment.id}`}>
                        <Card className="transition-shadow hover:shadow-md">
                          <CardHeader>
                            <div className="flex items-start justify-between gap-4">
                              <div className="flex-1">
                                <CardTitle className="text-base">{assignment.title}</CardTitle>
                              </div>
                              <div className="flex flex-col items-end gap-2">
                                {getStatusBadge(status)}
                                <span className="text-sm font-medium">{assignment.point} pts</span>
                              </div>
                            </div>
                          </CardHeader>
                          <CardContent>
                            <div className="flex flex-col gap-3">
                              <div className="flex flex-col gap-1 text-sm text-muted-foreground">
                                <div className="flex items-center gap-1">
                                  <CalendarPlus className="h-4 w-4" />
                                  <span>Start: {formatDate(assignment.start_date)}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                  <CalendarClock className="h-4 w-4" />
                                  <span className={status === "overdue" ? "text-red-600 font-medium" : ""}>Due: {formatDate(assignment.due_date)}</span>
                                </div>
                                <div className="flex items-center gap-1">
                                  <CalendarOff className="h-4 w-4" />
                                  <span>Close: {formatDate(assignment.close_date)}</span>
                                </div>
                              </div>
                              <div className="flex flex-wrap items-center gap-2">
                                <Tag className="h-3.5 w-3.5 text-muted-foreground" />
                                {assignment.tags.length > 0 ? (
                                  assignment.tags.map((tag) => (
                                    <Badge key={tag} variant="outline" className="text-xs">
                                      {tag}
                                    </Badge>
                                  ))
                                ) : (
                                  <span className="text-sm text-muted-foreground">No tag</span>
                                )}
                              </div>
                            </div>
                          </CardContent>
                        </Card>
                      </Link >
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </>
  )
}
