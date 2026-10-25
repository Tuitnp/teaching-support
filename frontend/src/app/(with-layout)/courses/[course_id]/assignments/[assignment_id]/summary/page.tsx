"use client"

import { useParams } from "next/navigation"
import React, { useEffect, useState, useMemo } from "react"
import {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { Download, ChevronLeft, ChevronRight, MoreHorizontal } from "lucide-react"
import SummaryChart from "./summary-chart"
import { useAuth } from "@/lib/auth-context"
import { summaryGradeValue } from "@/lib/submission-record"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import ExtendDueDateDialog from "@/components/extend-due-date-dialog"
import { BreadcrumbNav } from "@/components/breadcrumb-nav"

/* ── Data interfaces ── */

interface Distribution {
  range_start: number
  range_end: number
  count: number
}

interface SubmissionStatistic {
  students: number
  submitted: number
  incomplete: number
  not_started: number
  submission_rate: number
  graded: number
  ungrade: number
  avg_score: number
  highest_score: number
  lowest_score: number
  median_score: number
  distribution: Distribution[]
}

type SubmissionList = {
  has_grade?: boolean
  user_id: string
  student_id: string
  en_name: string
  th_name: string
  email: string
  point: number
  percentage: number
  submission_status: string
}

type AssignmentDetail = { title: string; point?: number }
type CourseDetail = { name: string }

/* ── Status badge ── */

function StatusBadge({ status }: { status: string }) {
  switch (status.toLowerCase()) {
    case "submitted":
      return (
        <Badge className="bg-emerald-500 hover:bg-emerald-600">
          {status}
        </Badge>
      )

    case "overdue":
      return (
        <Badge className="bg-orange-500 hover:bg-orange-600">
          {status}
        </Badge>
      )

    case "incomplete":
      return (
        <Badge className="bg-amber-500 hover:bg-amber-600 text-black">
          {status}
        </Badge>
      )

    case "no submitted":
      return (
        <Badge variant="destructive">
          Not started
        </Badge>
      )

    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

/* ── Inline columns definition (avoids external dialog state issues) ── */

function useColumns(fullPoint: number | null) {
  return useMemo<ColumnDef<SubmissionList>[]>(
    () => [
      {
        accessorKey: "student_id",
        header: "Student ID",
        enableSorting: true,
      },
      {
        accessorKey: "en_name",
        header: "Name",
        cell: ({ row }) =>
          (row.getValue("en_name") as string).replace(/\b\w/g, (c) => c.toUpperCase()),
        enableSorting: true,
      },
      {
        accessorKey: "th_name",
        header: "Thai Name",
      },
      {
        accessorKey: "email",
        header: "Email",
      },
      {
        accessorKey: "submission_status",
        header: "Status",
        cell: ({ row }) => (
          <div className="flex justify-center">
            <StatusBadge status={row.getValue("submission_status") as string} />
          </div>
        ),
        filterFn: (row, _id, value) => {
          if (!value || value === "all") return true
          return (row.getValue("submission_status") as string) === value
        },
      },
      {
        accessorKey: "point",
        header: () => <div className="text-right">{fullPoint != null ? `Point (${fullPoint})` : "Point"}</div>,
        cell: ({ row }) => (
          <div className="text-right tabular-nums">{summaryGradeValue(row.original.has_grade, row.original.point) === "" ? "—" : row.original.point}</div>
        ),
        enableSorting: true,
      },
      {
        accessorKey: "percentage",
        header: "Percentage",
        cell: ({ row }) => (
          <div className="text-right tabular-nums">
            {summaryGradeValue(row.original.has_grade, row.original.percentage) === "" ? "—" : `${row.original.percentage.toFixed(2)}%`}
          </div>
        ),
      },
    ],
    [fullPoint]
  )
}

/* ── Page component ── */

export default function Page() {
  const [statistic, setStatistic] = useState<SubmissionStatistic | null>(null)
  const [submissionList, setSubmissionList] = useState<SubmissionList[]>([])
  const [courseName, setCourseName] = useState("")
  const [assignmentName, setAssignmentName] = useState("")
  const [assignmentPoint, setAssignmentPoint] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { course_id, assignment_id } = useParams()
  const { user } = useAuth()

  /* Extend due date dialog */
  const [dialogOpen, setDialogOpen] = useState(false)
  const [selectedUserId, setSelectedUserId] = useState("")
  const handleExtendDueDate = (userId: string) => {
    setSelectedUserId(userId)
    setDialogOpen(true)
  }

  /* Table state */
  const baseColumns = useColumns(assignmentPoint)
  const columns = useMemo<ColumnDef<SubmissionList>[]>(
    () => [
      ...baseColumns,
      {
        id: "actions",
        header: "",
        cell: ({ row }) => {
          const userId = row.original.user_id
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0">
                  <span className="sr-only">Open menu</span>
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => handleExtendDueDate(userId)}>
                  Extend due date
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    ],
    [baseColumns]
  )
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({})
  const [rowSelection, setRowSelection] = useState({})
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const [globalFilter, setGlobalFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")

  const table = useReactTable({
    data: submissionList,
    columns,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
      globalFilter,
    },
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: setRowSelection,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: {
      pagination: { pageSize: 10 },
    },
  })

  // Sync status filter to column filter
  useEffect(() => {
    table.getColumn("submission_status")?.setFilterValue(statusFilter === "all" ? undefined : statusFilter)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter])

  /* CSV export */
  const handleExportCsv = () => {
    if (submissionList.length === 0) return

    const headers = [
      "student_id",
      "en_name",
      "th_name",
      "email",
      "submission_status",
      assignmentPoint != null ? `point (${assignmentPoint})` : "point",
      "percentage",
    ]

    const escapeCsv = (value: string | number) => {
      const text = String(value ?? "")
      if (text.includes(",") || text.includes('"') || text.includes("\n")) {
        return `"${text.replace(/"/g, '""')}"`
      }
      return text
    }

    const rows = submissionList.map((item) => [
      escapeCsv(item.student_id),
      escapeCsv(item.en_name),
      escapeCsv(item.th_name),
      escapeCsv(item.email),
      escapeCsv(item.submission_status),
      escapeCsv(summaryGradeValue(item.has_grade, item.point)),
      escapeCsv(summaryGradeValue(item.has_grade, item.percentage)),
    ])

    const csvContent = [headers.join(","), ...rows.map((row) => row.join(","))].join("\n")
    const blob = new Blob([`\uFEFF${csvContent}`], { type: "text/csv;charset=utf-8;" })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement("a")
    const safeName = assignmentName.trim().replace(/[\\/:*?"<>|]+/g, "_").replace(/\s+/g, "_")
    link.href = url
    link.download = `${safeName || "assignment"}_summary.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  }

  /* Data fetching */
  const loadSummary = async () => {
    if (!user?.token) return

    setLoading(true)
    setError(null)

    const [summaryRes, assignmentRes, courseRes] = await Promise.all([
      fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/${assignment_id}/summary`,
        { headers: { Authorization: `Bearer ${user.token}` }, cache: "no-store" }
      ),
      fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}/assignments/${assignment_id}`,
        { headers: { Authorization: `Bearer ${user.token}` }, cache: "no-store" }
      ),
      fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/courses/${course_id}`,
        { headers: { Authorization: `Bearer ${user.token}` }, cache: "no-store" }
      ),
    ])

    if (!summaryRes.ok) {
      setError(summaryRes.status === 409 ? "Course membership data conflicts. A teacher must resolve the enrollment records before reports can load." : "Failed to load assignment data")
      setLoading(false)
      return
    }

    const summaryData = await summaryRes.json()
    setStatistic(summaryData.statistic)
    setSubmissionList(summaryData.submission_list)

    if (assignmentRes.ok) {
      const assignmentData = await assignmentRes.json()
      const assignment = assignmentData.assignment as AssignmentDetail | undefined
      setAssignmentName(assignment?.title || "")
      setAssignmentPoint(assignment?.point ?? null)
    }

    if (courseRes.ok) {
      const courseData = await courseRes.json()
      const course = courseData as CourseDetail | undefined
      setCourseName(course?.name || "")
    }

    setLoading(false)
  }

  useEffect(() => {
    loadSummary()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [course_id, assignment_id, user?.token])

  /* Loading / error states */
  if (loading)
    return (
      <>
        <BreadcrumbNav courseName={courseName} assignmentName={assignmentName} />
        <div className="flex items-center justify-center min-h-[400px] p-6">
          <div className="flex flex-col items-center gap-3">
            <div className="w-8 h-8 rounded-full border-2 border-orange-500 border-t-transparent animate-spin" />
            <p className="text-sm text-muted-foreground">Loading assignment data...</p>
          </div>
        </div>
      </>
    )

  if (error)
    return (
      <>
        <BreadcrumbNav courseName={courseName} assignmentName={assignmentName} />
        <div className="flex items-center justify-center min-h-[400px] p-6">
          <div className="rounded-xl border border-red-200 bg-red-50 dark:bg-red-950/20 dark:border-red-900 px-6 py-5 text-red-600 dark:text-red-400 text-sm font-medium">
            {error}
          </div>
        </div>
      </>
    )

  if (!statistic || !submissionList) return null

  /* Pagination helpers */
  const pageIndex = table.getState().pagination.pageIndex
  const pageSize = table.getState().pagination.pageSize
  const filteredRows = table.getFilteredRowModel().rows.length
  const rowStart = filteredRows === 0 ? 0 : pageIndex * pageSize + 1
  const rowEnd = Math.min((pageIndex + 1) * pageSize, filteredRows)

  return (
    <>
      <BreadcrumbNav courseName={courseName} assignmentName={assignmentName} />
      <div className="p-6 space-y-6">
        {/* ── Page header ── */}
        <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">{courseName}</p>
            <h1 className="text-2xl font-bold tracking-tight">{assignmentName || "Assignment Summary"}</h1>
          </div>
          <div className="flex items-center">
            <div className="space-y-2">
              {assignmentPoint != null && (
                <p className="text-sm text-muted-foreground">
                  Full points: <span className="font-semibold text-foreground">{assignmentPoint}</span>
                </p>
              )}
              <Button variant="outline" size="sm" onClick={handleExportCsv}>
                <Download className="mr-2 h-4 w-4" />
                Export CSV
              </Button>
            </div>
          </div>
        </div>

        {/* ── Statistics ── */}
        <SummaryChart statistic={statistic} maxPoint={assignmentPoint} />

        {/* ── Submission table ── */}
        <div className="space-y-4">
              {/* Filters */}
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <h2 className="text-lg font-semibold">Submission Table</h2>
                <div className="flex items-center gap-3">
                  <Input
                    placeholder="Search by student ID or name..."
                    value={globalFilter ?? ""}
                    onChange={(e) => setGlobalFilter(e.target.value)}
                    className="w-[280px]"
                  />
                  <Select value={statusFilter} onValueChange={setStatusFilter}>
                    <SelectTrigger className="w-[180px]">
                      <SelectValue placeholder="All statuses" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All statuses</SelectItem>
                      <SelectItem value="submitted">Submitted</SelectItem>
                      <SelectItem value="overdue">Overdue</SelectItem>
                      <SelectItem value="incomplete">Incomplete</SelectItem>
                      <SelectItem value="no submitted">Not started</SelectItem>
                      <SelectItem value="timing unavailable">Timing unavailable</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              {/* Data table */}
              <div className="overflow-hidden rounded-md border">
                <Table>
                  <TableHeader>
                    {table.getHeaderGroups().map((headerGroup) => (
                      <TableRow key={headerGroup.id}>
                        {headerGroup.headers.map((header) => (
                          <TableHead
                            key={header.id}
                            className={`
                              ${
                                header.column.getCanSort()
                                  ? "cursor-pointer select-none hover:bg-muted/50"
                                  : ""
                              }
                          
                              ${
                                header.column.id === "submission_status"
                                  ? "text-center"
                                  : ["point", "percentage"].includes(header.column.id)
                                  ? "text-right"
                                  : ""
                              }
                            `}
                            onClick={header.column.getToggleSortingHandler()}
                          >
                            <div
                              className={`
                                flex items-center gap-1
                                ${
                                  header.column.id === "submission_status"
                                    ? "justify-center"
                                    : ["point", "percentage"].includes(header.column.id)
                                    ? "justify-end"
                                    : ""
                                }
                              `}
                            >
                              {header.isPlaceholder
                                ? null
                                : flexRender(header.column.columnDef.header, header.getContext())}
                              {{
                                asc: " \u2191",
                                desc: " \u2193",
                              }[header.column.getIsSorted() as string] ?? null}
                            </div>
                          </TableHead>
                        ))}
                      </TableRow>
                    ))}
                  </TableHeader>
                  <TableBody>
                    {table.getRowModel().rows?.length ? (
                      table.getRowModel().rows.map((row) => (
                        <TableRow
                          key={row.id}
                          data-state={row.getIsSelected() && "selected"}
                        >
                          {row.getVisibleCells().map((cell) => (
                            <TableCell key={cell.id}>
                              {flexRender(cell.column.columnDef.cell, cell.getContext())}
                            </TableCell>
                          ))}
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={columns.length} className="h-24 text-center">
                          No results.
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>

              {/* Pagination */}
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <p>
                  Showing {rowStart}-{rowEnd} of {filteredRows} student{filteredRows !== 1 ? "s" : ""}
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => table.previousPage()}
                    disabled={!table.getCanPreviousPage()}
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  <span className="tabular-nums">
                    Page {pageIndex + 1} of {table.getPageCount()}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => table.nextPage()}
                    disabled={!table.getCanNextPage()}
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
      </div>
      <ExtendDueDateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        userId={selectedUserId}
        courseId={course_id as string}
        assignmentId={assignment_id as string}
      />
    </>
  )
}
