"use client"

import { useEffect, useState } from "react"
import { useAuth } from "@/lib/auth-context"
import type { AICredential } from "@/lib/ai-types"
import { Button } from "@/components/ui/button"
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { Textarea } from "@/components/ui/textarea"
import { ArrowUpRight, Bot, ChevronDown, ChevronRight } from "lucide-react"
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { toast } from "sonner"

type Assignment = {
    id: string
    title: string
    ai_config_id?: string
    prompt?: string
    assignment_prompt?: string
    visible: boolean
}

type AIConfig = {
    id: string
    credential_id: string
    config_name: string
    credential_name: string
    model: string
}

interface AssignmentAISetupDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    courseId: string
    assignment: Assignment
    onSaved: (assignment: { ai_config_id?: string; assignment_prompt: string }) => void
    onStartGrading: () => Promise<void> | void
}

export function AssignmentAISetupDialog({
    open,
    onOpenChange,
    courseId,
    assignment,
    onSaved,
    onStartGrading,
}: AssignmentAISetupDialogProps) {
    const { user } = useAuth()
    const [credentials, setCredentials] = useState<AICredential[]>([])
    const [configs, setConfigs] = useState<AIConfig[]>([])
    const [isLoadingCredentials, setIsLoadingCredentials] = useState(false)
    const [isLoadingConfigs, setIsLoadingConfigs] = useState(false)
    const [selectedCredentialId, setSelectedCredentialId] = useState("")
    const [selectedConfigId, setSelectedConfigId] = useState("")
    const [promptSectionOpen, setPromptSectionOpen] = useState(true)
    const [prompt, setPrompt] = useState("")
    const [isSaving, setIsSaving] = useState(false)

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
            const currentConfigId = (assignment.ai_config_id ?? "").trim()
            if (currentConfigId) {
                await fetchConfigDetail(currentConfigId, next)
            } else {
                setSelectedCredentialId((cur) => (next.length === 0 ? "" : next.some((c) => c.id === cur) ? cur : next[0].id))
            }
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

    const fetchConfigDetail = async (configId: string, credentialsList?: AICredential[]) => {
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
            const promptValue = (assignment.prompt ?? assignment.assignment_prompt ?? "").trim()
            setPrompt(promptValue)
            if (data?.credential_id && Array.isArray(credentialsList) && credentialsList.length > 0) {
                await fetchConfigs(data.credential_id)
            }
        } catch {
            const currentPrompt = (assignment.prompt ?? assignment.assignment_prompt ?? "").trim()
            setPrompt(currentPrompt)
        }
    }

    useEffect(() => {
        if (open) {
            setPrompt((assignment.prompt ?? assignment.assignment_prompt ?? "").trim())
            setPromptSectionOpen(true)
            void fetchCredentials()
        } else {
            setCredentials([])
            setConfigs([])
            setSelectedCredentialId("")
            setSelectedConfigId("")
            setPrompt((assignment.prompt ?? assignment.assignment_prompt ?? "").trim())
            setPromptSectionOpen(true)
            setIsSaving(false)
        }
        // Intentionally react only to dialog open/close and assignment identity.
    }, [open, assignment.id])

    useEffect(() => {
        if (!selectedCredentialId) {
            setConfigs([])
            setSelectedConfigId("")
            return
        }
        void fetchConfigs(selectedCredentialId)
    }, [selectedCredentialId])

    useEffect(() => {
        const currentConfigId = (assignment.ai_config_id ?? "").trim()
        if (open && currentConfigId) {
            void fetchConfigDetail(currentConfigId, credentials)
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open, assignment.ai_config_id])

    const handleStartGrading = async () => {
        if (!user?.token) return
        if (!selectedConfigId) {
            toast.error("Select an AI config first", {
                description: "Choose a credential and model profile before starting AI grading.",
            })
            return
        }

        setIsSaving(true)
        try {
            const formData = new FormData()
            formData.append("ai_config_id", selectedConfigId)
            formData.append("visible", String(assignment.visible))
            if (prompt.trim()) {
                formData.append("prompt", prompt.trim())
            }

            const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/courses/${courseId}/assignments/${assignment.id}`, {
                method: "PUT",
                headers: {
                    Authorization: `Bearer ${user.token}`,
                },
                body: formData,
            })

            if (!res.ok) {
                toast.error("Failed to save AI setup", {
                    description: "Please try again.",
                })
                return
            }

            const data = await res.json()
            onSaved({
                ai_config_id: data?.ai_config_id,
                assignment_prompt: data?.prompt ?? prompt.trim(),
            })
            onOpenChange(false)
            await onStartGrading()
        } catch {
            toast.error("Something went wrong", {
                description: "Unable to save AI setup.",
            })
        } finally {
            setIsSaving(false)
        }
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="max-h-[90vh] sm:max-w-2xl">
                <DialogHeader>
                    <DialogTitle>AI setup</DialogTitle>
                    <DialogDescription>
                        Review or update the credential and model profile before starting AI grading.
                    </DialogDescription>
                </DialogHeader>

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
                                    <p className="text-sm text-muted-foreground">
                                        No AI credential profiles found. Open AI settings to create a credential and config first.
                                    </p>
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
                                            <p className="text-xs text-muted-foreground">
                                                Provider: {credentials.find((c) => c.id === selectedCredentialId)?.provider}
                                            </p>
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
                                                <p className="text-sm text-muted-foreground">
                                                    No configs found for this credential. Open AI settings to create one.
                                                </p>
                                            </div>
                                        ) : (
                                            <Select
                                                value={selectedConfigId}
                                                onValueChange={(value) => setSelectedConfigId(value)}
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
                                                Using {configs.find((c) => c.id === selectedConfigId)?.model} from {configs.find((c) => c.id === selectedConfigId)?.credential_name}.
                                            </p>
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
                                                value={prompt}
                                                onChange={(event) => setPrompt(event.target.value)}
                                            />
                                        </div>
                                    )}
                                </div>
                            )}
                        </>
                    )}
                </div>

                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isSaving}>
                        Cancel
                    </Button>
                    <Button onClick={handleStartGrading} disabled={isSaving || !selectedConfigId}>
                        {isSaving ? <Spinner className="mr-2 h-4 w-4" /> : null}
                        Save & Start AI grading
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}
