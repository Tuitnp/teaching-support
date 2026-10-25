"use client"

import { useState, useEffect } from "react"
import { serializeDateTime } from "@/lib/datetime"
import { useAuth } from "@/lib/auth-context"
import type { AICredential, AIConfig } from "@/lib/ai-types"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import { Spinner } from "@/components/ui/spinner"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import { X, Paperclip, Upload, FolderOpen, FileText, File as FileIcon, Search, Pencil, ChevronDown, ChevronRight, Bot, ArrowUpRight } from "lucide-react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { TipTapTextEditor } from "./ui/tiptap"
import { toast } from "sonner"
import SmartDatetimePickerByTui from "./smart-datetime-input2"

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
  ai_config_id: string
  prompt: string
  visible: boolean
}

interface EditAssignmentDialogProps {
  assignment: Assignment
  courseId: string
  onUpdated: (assignment: Assignment) => void
}

type AssignmentForm = {
  title: string
  description: string
  point: number
  start_date: Date | null
  due_date: Date | null
  close_date: Date | null
  ai_config_id: string
  prompt: string
  visible: boolean
}

const DEFAULT_FORM: AssignmentForm = {
  title: "",
  description: "",
  point: 10,
  start_date: null,
  due_date: null,
  close_date: null,
  ai_config_id: "",
  prompt: "",
  visible: true,
}

export function EditAssignmentDialog({
  assignment,
  courseId,
  onUpdated
}: EditAssignmentDialogProps) {
  const { user } = useAuth()
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<AssignmentForm>(DEFAULT_FORM)
  const [tagInput, setTagInput] = useState("")
  const [tags, setTags] = useState<string[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  // AI setup
  const [credentials, setCredentials] = useState<AICredential[]>([])
  const [configs, setConfigs] = useState<AIConfig[]>([])
  const [isLoadingCredentials, setIsLoadingCredentials] = useState(false)
  const [isLoadingConfigs, setIsLoadingConfigs] = useState(false)
  const [selectedCredentialId, setSelectedCredentialId] = useState("")
  const [selectedConfigId, setSelectedConfigId] = useState("")
  const [promptSectionOpen, setPromptSectionOpen] = useState(false)

  // Attachments
  const [selectedAttachments, setSelectedAttachments] = useState<Attachment[]>([])
  const [uploadedFiles, setUploadedFiles] = useState<File[]>([])
  const [existingAttachments, setExistingAttachments] = useState<Attachment[]>([])
  const [isLoadingAttachments, setIsLoadingAttachments] = useState(false)
  const [attachmentTabOpen, setAttachmentTabOpen] = useState(false)
  const [attachmentSearch, setAttachmentSearch] = useState("")

  const MAX_FILES = 5
  const totalAttachments = selectedAttachments.length + uploadedFiles.length

  const set = <K extends keyof AssignmentForm>(
    key: K,
    value: AssignmentForm[K]
  ) => {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  const getFileIcon = (mime: string) => {
    if (mime.includes("pdf") || mime.includes("word")) return FileText
    return FileIcon
  }

  const fetchCredentials = async () => {
    if (!user?.token) return
    setIsLoadingCredentials(true)
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/ai`, {
        headers: { Authorization: `Bearer ${user.token}` },
      })
      if (!res.ok) throw new Error("Failed to load AI credentials")
      const data = await res.json()
      const next: AICredential[] = Array.isArray(data) ? data : []
      setCredentials(next)
      setSelectedCredentialId((cur) => (next.length === 0 ? "" : next.some((c) => c.id === cur) ? cur : next[0].id))
    } catch {
      setCredentials([])
      setSelectedCredentialId("")
    } finally {
      setIsLoadingCredentials(false)
    }
  }

  const fetchConfigs = async (credentialId: string) => {
    if (!user?.token || !credentialId) {
      setConfigs([])
      return
    }
    setIsLoadingConfigs(true)
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/ai/configs/${credentialId}`, {
        headers: { Authorization: `Bearer ${user.token}` },
      })
      if (!res.ok) throw new Error("Failed to load AI configs")
      const data = await res.json()
      setConfigs(Array.isArray(data) ? data : [])
    } catch {
      setConfigs([])
    } finally {
      setIsLoadingConfigs(false)
    }
  }

  const fetchConfigDetail = async (configId: string) => {
    if (!user?.token || !configId) return
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/ai/configs/detail/${configId}`, {
        headers: { Authorization: `Bearer ${user.token}` },
      })
      if (!res.ok) throw new Error("Failed to load AI config")
      const data = await res.json()
      if (data?.credential_id) {
        setSelectedCredentialId(data.credential_id)
      }
      if (data?.id) {
        setSelectedConfigId(data.id)
      }
      if (typeof data?.id === "string") {
        setForm((prev) => ({
          ...prev,
          ai_config_id: data.id,
          prompt: assignment.prompt ?? prev.prompt,
        }))
      }
      if (data?.id) {
        setPromptSectionOpen(true)
      }
    } catch {
      // Keep the assignment prompt in the form even if config detail lookup fails.
    }
  }

  useEffect(() => {
    if (open) void fetchCredentials()
  }, [open])

  useEffect(() => {
    if (open && assignment.ai_config_id) {
      void fetchConfigDetail(assignment.ai_config_id)
    }
  }, [open, assignment])

  useEffect(() => {
    if (!selectedCredentialId) {
      setConfigs([])
      setForm((prev) => ({ ...prev, ai_config_id: "", prompt: "" }))
      return
    }
    void fetchConfigs(selectedCredentialId)
  }, [selectedCredentialId])

  useEffect(() => {
    if (!selectedConfigId) return
    setPromptSectionOpen(true)
  }, [selectedConfigId])

  useEffect(() => {
    if (!selectedConfigId || isLoadingConfigs) return
    const exists = configs.some((c) => c.id === selectedConfigId)
    if (!exists) {
      setSelectedConfigId("")
      setForm((prev) => ({ ...prev, ai_config_id: "", prompt: "" }))
    }
  }, [configs, selectedConfigId, isLoadingConfigs])

  const fetchExistingAttachments = async () => {
    if (existingAttachments.length > 0) return
    setIsLoadingAttachments(true)
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/attachments`, {
        headers: { Authorization: `Bearer ${user?.token}` },
      })
      const data = await res.json()
      setExistingAttachments(data ?? [])
    } finally {
      setIsLoadingAttachments(false)
    }
  }

  const handleUploadFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? [])
    setUploadedFiles((prev) => {
      const combined = [...prev, ...files]
      const remaining = MAX_FILES - selectedAttachments.length
      return combined.slice(0, remaining)
    })
    e.target.value = ""
  }

  const removeUploadedFile = (index: number) =>
    setUploadedFiles((prev) => prev.filter((_, i) => i !== index))

  const toggleExistingAttachment = (att: Attachment) => {
    setSelectedAttachments((prev) => {
      const exists = prev.find((a) => a.id === att.id)
      if (exists) return prev.filter((a) => a.id !== att.id)
      if (prev.length + uploadedFiles.length >= MAX_FILES) return prev
      return [...prev, att]
    })
  }

  const addTag = () => {
    const t = tagInput.trim().toLowerCase()
    if (t && !tags.includes(t)) setTags((prev) => [...prev, t])
    setTagInput("")
  }

  const removeTag = (tag: string) => setTags((prev) => prev.filter((t) => t !== tag))

  const validate = () => {
    const e: Record<string, string> = {}
    if (!form.title.trim()) e.title = "Title is required"
    if (!form.description.trim()) e.description = "Description is required"
    if (form.point <= 0) e.point = "Points must be greater than 0"
    if (!form.start_date) e.start_date = "Start date is required"
    if (!form.due_date) e.due_date = "Due date is required"
    if (!form.close_date) e.close_date = "Close date is required"
    if (form.start_date && form.due_date && form.due_date <= form.start_date)
      e.due_date = "Due date must be after start date"
    if (form.due_date && form.close_date && form.close_date < form.due_date)
      e.close_date = "Close date must be on or after due date"
    if ((form.prompt ?? "").trim() && !form.ai_config_id) {
      e.ai_config_id = "Select an AI config before adding a prompt"
    }
    setErrors(e)
    return Object.keys(e).length === 0
  }

  const handleSubmit = async () => {
    if (!validate()) return
    setIsSubmitting(true)
    try {
      const formData = new FormData();

      if (!form.start_date || !form.due_date || !form.close_date) return;

      formData.append("title", form.title)
      formData.append("description", form.description)
      formData.append("point", String(form.point))
      formData.append("start", serializeDateTime(form.start_date, assignment.start_date))
      formData.append("due", serializeDateTime(form.due_date, assignment.due_date))
      formData.append("close", serializeDateTime(form.close_date, assignment.close_date))
      if (form.ai_config_id) {
        formData.append("ai_config_id", form.ai_config_id)
      }
      if ((form.prompt ?? "").trim()) {
        formData.append("prompt", form.prompt)
      }
      formData.append("visible", String(form.visible))

      if (tags) {
        tags.forEach(tag => formData.append("tags", tag));
      }

      if (uploadedFiles && Array.isArray(uploadedFiles) && uploadedFiles.length > 0) {
        uploadedFiles.forEach((file) => {
          if (file instanceof File) {
            formData.append("files", file);
          }
        });
      } else if (selectedAttachments && selectedAttachments.length > 0) {
        selectedAttachments.forEach((attachment) => {
          formData.append("attachments", attachment.id);
        });
      }

      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/courses/${courseId}/assignments/${assignment.id}`, {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${user?.token}`,
        },
        body: formData,
      })

      if (!res.ok) {
        toast.error("Failed to save Assignment", {
          description: "Something went wrong. Please try again.",
        })
        return;
      }

      const data = await res.json()
      onUpdated(data)
      setOpen(false)
      setErrors({})
      toast.success("Assignment updated", {
        description: "The assignment was updated successfully.",
      })
    } catch (err) {
      setErrors({ submit: err instanceof Error ? err.message : "Something went wrong" })
      toast.error("Failed to save Assignment", {
        description: "Something went wrong. Please try again.",
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleOpenChange = (val: boolean) => {
    setOpen(val)
    if (val) {
      // Initialize form with assignment data when opening
      setForm({
        title: assignment.title,
        description: assignment.description,
        point: assignment.point,
        start_date: assignment.start_date ? new Date(assignment.start_date) : null,
        due_date: assignment.due_date ? new Date(assignment.due_date) : null,
        close_date: assignment.close_date ? new Date(assignment.close_date) : null,
        ai_config_id: assignment.ai_config_id ?? "",
        prompt: assignment.prompt ?? "",
        visible: assignment.visible,
      })
      setTags(assignment.tags || [])
      setSelectedAttachments(assignment.attachments || [])
      setUploadedFiles([])
      setTagInput("")
      setErrors({})
      setAttachmentTabOpen(false)
      setPromptSectionOpen(Boolean(assignment.ai_config_id || assignment.prompt))
    } else {
      setErrors({})
      setSelectedAttachments([])
      setUploadedFiles([])
      setAttachmentTabOpen(false)
      setPromptSectionOpen(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className="w-full justify-start gap-2"
        >
          <Pencil className="h-4 w-4" />
          Edit
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90vh] sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Edit Assignment</DialogTitle>
          <DialogDescription>Edit this assignment&apos;s details.</DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[calc(90vh-12rem)] pr-4">
          <div className="grid gap-5 py-2">
            {/* Title */}
            <div className="grid gap-1.5">
              <Label htmlFor="title">Title <span className="text-destructive">*</span></Label>
              <Input
                id="title"
                placeholder="e.g. HTML Basics Quiz"
                value={form.title}
                onChange={(e) => set("title", e.target.value)}
              />
              {errors.title && <p className="text-xs text-destructive">{errors.title}</p>}
            </div>

            {/* Description */}
            <div className="grid gap-1.5">
              <Label htmlFor="description">Description <span className="text-destructive">*</span></Label>
              <TipTapTextEditor
                value={form.description}
                onChange={(e) => set("description", e)}
                editable={true}
              />
              {errors.description && <p className="text-xs text-destructive">{errors.description}</p>}
            </div>

            {/* Points */}
            <div className="grid gap-1.5">
              <Label htmlFor="point">Points <span className="text-destructive">*</span></Label>
              <Input
                id="point"
                type="number"
                min={1}
                placeholder="100"
                value={form.point}
                onChange={(e) => set("point", Number(e.target.value))}
              />
              {errors.point && <p className="text-xs text-destructive">{errors.point}</p>}
            </div>

            {/* Tags */}
            <div className="grid gap-1.5">
              <Label>Tags</Label>
              <div className="flex gap-2">
                <Input
                  placeholder="Add a tag and press Enter"
                  value={tagInput}
                  onChange={(e) => setTagInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault()
                      addTag()
                    }
                  }}
                />
                <Button type="button" variant="outline" onClick={addTag}>
                  Add
                </Button>
              </div>
              {tags.length > 0 && (
                <div className="flex flex-wrap gap-2 pt-1">
                  {tags.map((tag) => (
                    <Badge key={tag} variant="secondary" className="gap-1 pr-1">
                      {tag}
                      <button
                        onClick={() => removeTag(tag)}
                        className="ml-0.5 rounded-sm opacity-60 hover:opacity-100"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}
            </div>

            {/* Attachments */}
            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label>
                  Attachments
                  <span className="ml-2 text-xs font-normal text-muted-foreground">
                    (optional, up to {MAX_FILES})
                  </span>
                </Label>
                <span className="text-xs text-muted-foreground">
                  {totalAttachments}/{MAX_FILES}
                </span>
              </div>

              {/* Selected summary chips */}
              {totalAttachments > 0 && (
                <div className="flex flex-wrap gap-1.5">
                  {uploadedFiles.map((f, i) => (
                    <Badge key={`upload-${i}`} variant="secondary" className="gap-1 pr-1 text-xs">
                      <Upload className="h-3 w-3 shrink-0" />
                      {f.name}
                      <button onClick={() => removeUploadedFile(i)} className="ml-0.5 rounded-sm opacity-60 hover:opacity-100">
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                  {selectedAttachments.map((a) => (
                    <Badge key={a.id} variant="secondary" className="gap-1 pr-1 text-xs">
                      <Paperclip className="h-3 w-3 shrink-0" />
                      {a.file_name}
                      <button onClick={() => toggleExistingAttachment(a)} className="ml-0.5 rounded-sm opacity-60 hover:opacity-100">
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}

              {/* Add attachment panel */}
              {totalAttachments < MAX_FILES && (
                <div className="rounded-lg border">
                  <button
                    type="button"
                    onClick={() => {
                      setAttachmentTabOpen((v) => !v)
                      fetchExistingAttachments()
                    }}
                    className="flex w-full items-center justify-between px-4 py-3 text-sm text-muted-foreground hover:text-foreground transition-colors"
                  >
                    <span className="flex items-center gap-2">
                      <Paperclip className="h-4 w-4" />
                      Add attachment
                    </span>
                    <span className="text-xs">{attachmentTabOpen ? "Close" : "Open"}</span>
                  </button>

                  {attachmentTabOpen && (
                    <div className="border-t">
                      <Tabs defaultValue="upload" className="p-3">
                        <TabsList className="mb-3 w-full">
                          <TabsTrigger value="upload" className="flex-1 gap-1.5">
                            <Upload className="h-3.5 w-3.5" />
                            Upload New
                          </TabsTrigger>
                          <TabsTrigger value="existing" className="flex-1 gap-1.5">
                            <FolderOpen className="h-3.5 w-3.5" />
                            Select Existing
                          </TabsTrigger>
                        </TabsList>

                        <TabsContent value="upload" className="mt-0">
                          <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-muted-foreground/25 p-6 text-center transition-colors hover:border-muted-foreground/50 hover:bg-muted/30">
                            <Upload className="h-8 w-8 text-muted-foreground" />
                            <div>
                              <p className="text-sm font-medium">Click to upload files</p>
                              <p className="text-xs text-muted-foreground mt-0.5">
                                Up to {MAX_FILES - totalAttachments} more file{MAX_FILES - totalAttachments !== 1 ? "s" : ""}
                              </p>
                            </div>
                            <input
                              type="file"
                              multiple
                              className="sr-only"
                              onChange={handleUploadFiles}
                            />
                          </label>
                        </TabsContent>

                        <TabsContent value="existing" className="mt-0">
                          {isLoadingAttachments ? (
                            <div className="flex items-center justify-center py-8">
                              <Spinner className="h-5 w-5" />
                            </div>
                          ) : existingAttachments.length === 0 ? (
                            <p className="py-6 text-center text-sm text-muted-foreground">No uploaded files found</p>
                          ) : (
                            <>
                              <div className="relative mb-2">
                                <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                                <Input
                                  placeholder="Search files..."
                                  value={attachmentSearch}
                                  onChange={(e) => setAttachmentSearch(e.target.value)}
                                  className="h-8 pl-8 text-sm"
                                />
                              </div>
                              <ScrollArea className="h-44">
                                <div className="space-y-1 pr-2">
                                  {existingAttachments
                                    .filter((a) =>
                                      a.file_name.toLowerCase().includes(attachmentSearch.toLowerCase())
                                    )
                                    .map((att) => {
                                      const Icon = getFileIcon(att.file_type)
                                      const isSelected = selectedAttachments.some((a) => a.id === att.id)
                                      const isDisabled = !isSelected && totalAttachments >= MAX_FILES
                                      return (
                                        <button
                                          key={att.id}
                                          type="button"
                                          disabled={isDisabled}
                                          onClick={() => toggleExistingAttachment(att)}
                                          className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors ${isSelected
                                            ? "bg-primary/10 text-primary ring-1 ring-primary/30"
                                            : isDisabled
                                              ? "cursor-not-allowed opacity-40"
                                              : "hover:bg-muted"
                                            }`}
                                        >
                                          <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                                          <div className="min-w-0 flex-1">
                                            <p className="truncate font-medium">{att.file_name}</p>
                                            <p className="text-xs text-muted-foreground">{formatSize(att.size)}</p>
                                          </div>
                                          {isSelected && (
                                            <span className="shrink-0 text-xs font-medium text-primary">Selected</span>
                                          )}
                                        </button>
                                      )
                                    })}
                                  {existingAttachments.filter((a) =>
                                    a.file_name.toLowerCase().includes(attachmentSearch.toLowerCase())
                                  ).length === 0 && (
                                      <p className="py-4 text-center text-xs text-muted-foreground">
                                        No files match &ldquo;{attachmentSearch}&rdquo;
                                      </p>
                                    )}
                                </div>
                              </ScrollArea>
                            </>
                          )}
                        </TabsContent>
                      </Tabs>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Dates */}
            <div className="flex flex-col gap-2">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="start_date">Start Date <span className="text-destructive">*</span></Label>
                <SmartDatetimePickerByTui
                  value={form.start_date}
                  onChange={(e) => set("start_date", e)}
                  placeholder='E.g. "tomorrow at 5pm" or "in 2 hours"'
                />
                {errors.start_date && <p className="text-xs text-destructive">{errors.start_date}</p>}
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="due_date">Due Date <span className="text-destructive">*</span></Label>
                <SmartDatetimePickerByTui
                  value={form.due_date}
                  onChange={(e) => set("due_date", e)}
                  placeholder='E.g. "tomorrow at 5pm" or "in 2 hours"'
                />
                {errors.due_date && <p className="text-xs text-destructive">{errors.due_date}</p>}
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="close_date">Close Date <span className="text-destructive">*</span></Label>
                <SmartDatetimePickerByTui
                  value={form.close_date}
                  onChange={(e) => set("close_date", e)}
                  placeholder='E.g. "tomorrow at 5pm" or "in 2 hours"'
                />
                {errors.close_date && <p className="text-xs text-destructive">{errors.close_date}</p>}
              </div>
            </div>

            <div className="grid gap-3 rounded-lg border p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-0.5">
                  <div className="flex items-center gap-2">
                    <Bot className="h-4 w-4 text-muted-foreground" />
                    <Label className="text-base">AI setup</Label>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    Pick a credential first, then a config, then write the prompt.
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Button asChild variant="outline" size="sm" className="shrink-0 gap-2">
                    <a href="/ai/settings" target="_blank" rel="noreferrer">
                      AI settings
                      <ArrowUpRight className="h-4 w-4" />
                    </a>
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="shrink-0 gap-2"
                    onClick={() => setPromptSectionOpen((value) => !value)}
                  >
                    {promptSectionOpen ? "Hide" : "Show"}
                    {promptSectionOpen ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>

              {promptSectionOpen && (
                <>
                  {isLoadingCredentials ? (
                    <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                      <Spinner className="h-4 w-4" />
                      Loading AI credentials...
                    </div>
                  ) : credentials.length === 0 ? (
                    <div className="rounded-md border p-3">
                      <p className="text-sm text-muted-foreground">No AI credential profiles found. Open AI settings in a new tab to create a credential and config first.</p>
                    </div>
                  ) : (
                    <div className="grid gap-3">
                      <div className="grid gap-1.5">
                        <Label htmlFor="ai_credential_id">AI credential</Label>
                        <Select
                          value={selectedCredentialId}
                          onValueChange={(value) => {
                            setSelectedCredentialId(value)
                            setSelectedConfigId("")
                            setForm((prev) => ({ ...prev, ai_config_id: "", prompt: "" }))
                          }}
                        >
                          <SelectTrigger id="ai_credential_id" className="w-full justify-between">
                            <SelectValue placeholder="Select an AI credential" />
                          </SelectTrigger>
                          <SelectContent>
                            {credentials.map((credential) => (
                              <SelectItem key={credential.id} value={credential.id}>
                                {credential.name}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        {selectedCredentialId && (
                          <p className="text-xs text-muted-foreground">Provider: {credentials.find(c => c.id === selectedCredentialId)?.provider}</p>
                        )}
                      </div>

                      <div className="grid gap-1.5">
                        <Label htmlFor="ai_config_id">AI config</Label>
                        {isLoadingConfigs ? (
                          <div className="flex items-center gap-2 rounded-md border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                            <Spinner className="h-4 w-4" />
                            Loading AI configs...
                          </div>
                        ) : configs.length === 0 ? (
                          <div className="rounded-md border p-3">
                            <p className="text-sm text-muted-foreground">No configs found for this credential. Open AI settings to create one.</p>
                          </div>
                        ) : (
                          <Select
                            value={selectedConfigId}
                            onValueChange={(value) => {
                              setSelectedConfigId(value)
                              setForm((prev) => ({ ...prev, ai_config_id: value, prompt: "" }))
                            }}
                          >
                            <SelectTrigger id="ai_config_id" className="w-full justify-between">
                              <SelectValue placeholder="Select an AI config" />
                            </SelectTrigger>
                            <SelectContent>
                              {configs.map((config) => (
                                <SelectItem key={config.id} value={config.id}>
                                  {config.config_name} · {config.model}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        )}
                        {selectedConfigId && (
                          <p className="text-xs text-muted-foreground">
                            Using {configs.find(c => c.id === selectedConfigId)?.model} from {configs.find(c => c.id === selectedConfigId)?.credential_name}.
                          </p>
                        )}
                        {errors.ai_config_id && (
                          <p className="text-xs text-destructive">{errors.ai_config_id}</p>
                        )}
                      </div>

                      {selectedConfigId && (
                        <div className="grid gap-3 rounded-md border bg-muted/20 p-3">
                          <div>
                            <Label className="text-sm">Prompt</Label>
                            <p className="text-xs text-muted-foreground">
                              This prompt is sent with the selected config.
                            </p>
                          </div>

                          <Textarea
                            id="prompt"
                            rows={4}
                            value={form.prompt}
                            onChange={(e) => set("prompt", e.target.value)}
                          />
                        </div>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>

            {errors.submit && (
              <p className="text-sm text-destructive">{errors.submit}</p>
            )}
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? <Spinner className="mr-2 h-4 w-4" /> : null}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
