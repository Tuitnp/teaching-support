"use client"

import { useMemo } from "react"
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from "recharts"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"

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

interface SummaryChartProps {
  statistic: SubmissionStatistic
  maxPoint?: number | null
}

function MetricCard({
  label,
  value,
  description,
  accent,
}: {
  label: string
  value: string | number
  description?: string
  accent?: string
}) {
  return (
    <Card size="sm">
      <CardContent className="flex flex-col gap-1">
        <p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
          {label}
        </p>
        <p
          className="text-2xl font-bold tabular-nums tracking-tight"
          style={accent ? { color: accent } : undefined}
        >
          {value}
        </p>
        {description && (
          <p className="text-[11px] text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  )
}

/* ---------- Submission status donut chart ---------- */

const STATUS_COLORS: Record<string, string> = {
  Submitted: "var(--color-submitted)",
  Incomplete: "var(--color-incomplete)",
  "Not started": "var(--color-not_started)",
}

const STATUS_CHART_CONFIG = {
  count: { label: "Students" },
  submitted: { label: "Submitted", color: "hsl(142, 71%, 45%)" },
  incomplete: { label: "Incomplete", color: "hsl(38, 92%, 50%)" },
  not_started: { label: "Not started", color: "hsl(0, 84%, 60%)" },
} satisfies ChartConfig

/* ---------- Distribution bar chart ---------- */

const DISTRIBUTION_CHART_CONFIG = {
  count: { label: "Students", color: "hsl(24, 95%, 53%)" },
} satisfies ChartConfig

export default function SummaryChart({
  statistic,
  maxPoint,
}: SummaryChartProps) {
  const toNum = (v: unknown) => {
    if (v === null || v === undefined || v === "") return 0
    const n = Number(v)
    return Number.isFinite(n) ? n : 0
  }

  const avgScore = toNum(statistic.avg_score)
  const medianScore = toNum(statistic.median_score)
  const highestScore = toNum(statistic.highest_score)
  const lowestScore = toNum(statistic.lowest_score)

  const submissionPieData = useMemo(
    () => [
      { status: "Submitted", count: statistic.submitted ?? 0, fill: "var(--color-submitted)" },
      { status: "Incomplete", count: statistic.incomplete ?? 0, fill: "var(--color-incomplete)" },
      { status: "Not started", count: statistic.not_started ?? 0, fill: "var(--color-not_started)" },
    ],
    [statistic.submitted, statistic.incomplete, statistic.not_started]
  )

  const distributionBarData = useMemo(
    () =>
      (statistic.distribution ?? []).map((d) => ({
        range: `${d.range_start}\u2013${d.range_end}`,
        count: d.count,
      })),
    [statistic.distribution]
  )

  const maxCount = useMemo(
    () => Math.max(...(statistic.distribution ?? []).map((d) => d.count), 0),
    [statistic.distribution]
  )

  const pointLabel = maxPoint != null ? ` (/${maxPoint})` : ""

  return (
    <div className="space-y-6">
      {/* ── Summary metric cards ── */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        <MetricCard
          label="Total Students"
          value={statistic.students ?? 0}
          accent="hsl(220, 70%, 55%)"
        />
        <MetricCard
          label="Submitted"
          value={statistic.submitted ?? 0}
          description={`${(statistic.submission_rate ?? 0).toFixed(1)}% rate`}
          accent="hsl(142, 71%, 45%)"
        />
        <MetricCard
          label="Incomplete"
          value={statistic.incomplete ?? 0}
          description="No file attached"
          accent="hsl(38, 92%, 50%)"
        />
        <MetricCard
          label="Not Started"
          value={statistic.not_started ?? 0}
          description="No submission yet"
          accent="hsl(0, 84%, 60%)"
        />
        <MetricCard
          label="Graded"
          value={statistic.graded ?? 0}
          description={`of ${statistic.submitted ?? 0} submitted`}
          accent="hsl(142, 71%, 45%)"
        />
        <MetricCard
          label="Ungraded"
          value={statistic.ungrade ?? 0}
          accent="hsl(38, 92%, 50%)"
        />
      </div>

      {/* ── Score stats row ── */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <MetricCard
          label={`Average${pointLabel}`}
          value={avgScore.toFixed(2)}
          accent="hsl(24, 95%, 53%)"
        />
        <MetricCard
          label={`Highest${pointLabel}`}
          value={highestScore}
          description="Top performer"
          accent="hsl(142, 71%, 45%)"
        />
        <MetricCard
          label={`Lowest${pointLabel}`}
          value={lowestScore}
          description="Needs attention"
          accent="hsl(0, 84%, 60%)"
        />
        <MetricCard
          label={`Median${pointLabel}`}
          value={medianScore}
          accent="hsl(262, 83%, 58%)"
        />
      </div>

      {/* ── Charts row ── */}
      <div className="grid grid-cols-1 lg:grid-cols-5 gap-4">
        {/* Submission status donut */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Submission Status</CardTitle>
            <CardDescription>
              {statistic.submitted ?? 0} of {statistic.students ?? 0} students
              submitted
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col items-center gap-4">
            <ChartContainer
              config={STATUS_CHART_CONFIG}
              className="mx-auto aspect-square w-full max-w-[260px]"
            >
              <PieChart>
                <ChartTooltip
                  cursor={false}
                  content={<ChartTooltipContent hideLabel nameKey="status" />}
                />
                <Pie
                  data={submissionPieData}
                  dataKey="count"
                  nameKey="status"
                  innerRadius="55%"
                  outerRadius="85%"
                  strokeWidth={2}
                  stroke="var(--background)"
                >
                  {submissionPieData.map((entry) => (
                    <Cell key={entry.status} fill={STATUS_COLORS[entry.status]} />
                  ))}
                </Pie>
              </PieChart>
            </ChartContainer>
            {/* Legend */}
            <div className="flex flex-wrap justify-center gap-x-5 gap-y-1 text-xs text-muted-foreground">
              {submissionPieData.map((item) => (
                <span key={item.status} className="flex items-center gap-1.5">
                  <span
                    className="inline-block h-2 w-2 rounded-full"
                    style={{ backgroundColor: STATUS_COLORS[item.status] }}
                  />
                  {item.status}: <span className="font-semibold text-foreground">{item.count}</span>
                </span>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Score distribution bar chart */}
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle>Score Distribution</CardTitle>
            <CardDescription>
              Number of students per score range
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ChartContainer
              config={DISTRIBUTION_CHART_CONFIG}
              className="aspect-auto h-[260px] w-full"
            >
              <BarChart
                data={distributionBarData}
                margin={{ top: 4, right: 4, left: -10, bottom: 4 }}
              >
                <CartesianGrid vertical={false} />
                <XAxis
                  dataKey="range"
                  tickLine={false}
                  axisLine={false}
                  tickMargin={8}
                  fontSize={12}
                />
                <YAxis
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  fontSize={12}
                />
                <ChartTooltip
                  cursor={false}
                  content={<ChartTooltipContent hideLabel />}
                />
                <Bar dataKey="count" radius={[6, 6, 0, 0]} maxBarSize={56}>
                  {distributionBarData.map((entry) => (
                    <Cell
                      key={entry.range}
                      fill={
                        entry.count === maxCount
                          ? "hsl(24, 95%, 53%)"
                          : "hsl(24, 90%, 88%)"
                      }
                    />
                  ))}
                </Bar>
              </BarChart>
            </ChartContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
