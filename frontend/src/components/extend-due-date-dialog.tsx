"use client"

import { useState } from "react"
import { useAuth } from "@/lib/auth-context"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { toast } from "sonner"
import SmartDatetimePickerByTui from "./smart-datetime-input2"

interface ExtendDueDateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: string
  courseId: string
  assignmentId: string
}

export default function ExtendDueDateDialog({
  open,
  onOpenChange,
  userId,
  courseId,
  assignmentId,
}: ExtendDueDateDialogProps) {
  const { user } = useAuth()
  const [extendedDueDate, setExtendedDueDate] = useState<Date | null>(null)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!user?.token) {
      toast.error("You must be logged in")
      return
    }

    if (!extendedDueDate) {
      toast.error("Please select a due date")
      return
    }

    setIsSubmitting(true)

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL}/courses/${courseId}/assignments/${assignmentId}/override`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${user.token}`,
          },
          body: JSON.stringify({
            student_id: userId,
            extended_due_date: extendedDueDate.toISOString(),
          }),
        }
      )

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.message || "Failed to extend due date")
      }

      toast.success("Due date extended successfully")
      onOpenChange(false)
      setExtendedDueDate(null)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to extend due date")
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      setExtendedDueDate(null)
    }
    onOpenChange(newOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>Extend Due Date</DialogTitle>
          <DialogDescription>
            Set a new extended due date for this student&apos;s assignment.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="py-4 space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Extended Due Date</label>
              <SmartDatetimePickerByTui
                value={extendedDueDate}
                onChange={setExtendedDueDate}
                placeholder="Select due date and time"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Extending..." : "Extend Due Date"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
