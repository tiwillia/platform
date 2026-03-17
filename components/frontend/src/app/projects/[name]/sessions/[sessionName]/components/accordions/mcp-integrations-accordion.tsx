'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'
import { Plug, Link2, CheckCircle2, XCircle, AlertCircle, AlertTriangle, Info, Check, X, Plus, Trash2, Loader2 } from 'lucide-react'
import {
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from 'sonner'
import { useMcpStatus } from '@/services/queries/use-mcp'
import { useIntegrationsStatus } from '@/services/queries/use-integrations'
import { useUpdateSessionMcpServers } from '@/services/queries/use-sessions'
import type { McpServer, McpTool } from '@/services/api/sessions'
import type { McpServerConfig } from '@/types/agentic-session'

// ─── MCP Servers Accordion ───────────────────────────────────────────────────

type McpServersAccordionProps = {
  projectName: string
  sessionName: string
  sessionPhase?: string
  specMcpServers?: McpServerConfig[]
}

export function McpServersAccordion({
  projectName,
  sessionName,
  sessionPhase,
  specMcpServers = [],
}: McpServersAccordionProps) {
  const [placeholderTimedOut, setPlaceholderTimedOut] = useState(false)
  const [showAddForm, setShowAddForm] = useState(false)
  const [newServer, setNewServer] = useState<McpServerConfig>({ name: '', type: 'http' })
  const updateMcpServersMutation = useUpdateSessionMcpServers()

  // Only fetch MCP status once the session is actually running (runner pod ready)
  const isRunning = sessionPhase === 'Running'
  const isStopped = sessionPhase === 'Stopped' || sessionPhase === 'Completed' || sessionPhase === 'Failed'
  const { data: mcpStatus, isPending: mcpPending } = useMcpStatus(projectName, sessionName, isRunning)
  const mcpServers = mcpStatus?.servers || []

  const showPlaceholders =
    !isRunning || mcpPending || (mcpServers.length === 0 && !placeholderTimedOut)

  const userServerNames = new Set(specMcpServers.map((s) => s.name))

  const handleAddServer = () => {
    if (!newServer.name.trim()) return
    const updated = [...specMcpServers, newServer]
    updateMcpServersMutation.mutate(
      { projectName, sessionName, mcpServers: updated },
      {
        onSuccess: () => {
          toast.success(`Added MCP server "${newServer.name}"`)
          setNewServer({ name: '', type: 'http' })
          setShowAddForm(false)
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to add MCP server')
        },
      }
    )
  }

  const handleRemoveServer = (serverName: string) => {
    const updated = specMcpServers.filter((s) => s.name !== serverName)
    updateMcpServersMutation.mutate(
      { projectName, sessionName, mcpServers: updated },
      {
        onSuccess: () => {
          toast.success(`Removed MCP server "${serverName}"`)
        },
        onError: (error) => {
          toast.error(error instanceof Error ? error.message : 'Failed to remove MCP server')
        },
      }
    )
  }

  useEffect(() => {
    if (mcpServers.length > 0) {
      setPlaceholderTimedOut(false)
      return
    }
    if (!isRunning || !mcpStatus) return
    const t = setTimeout(() => setPlaceholderTimedOut(true), 15 * 1000)
    return () => clearTimeout(t)
  }, [mcpStatus, mcpServers.length, isRunning])

  const getStatusIcon = (server: McpServer) => {
    switch (server.status) {
      case 'configured':
      case 'connected':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      case 'disconnected':
      default:
        return <AlertCircle className="h-4 w-4 text-muted-foreground" />
    }
  }

  const getStatusBadge = (server: McpServer) => {
    switch (server.status) {
      case 'configured':
        return (
          <Badge variant="outline" className="text-xs bg-blue-50 text-blue-700 border-blue-200">
            Configured
          </Badge>
        )
      case 'connected':
        return (
          <Badge variant="outline" className="text-xs bg-green-50 text-green-700 border-green-200">
            Connected
          </Badge>
        )
      case 'error':
        return (
          <Badge variant="outline" className="text-xs bg-red-50 text-red-700 border-red-200">
            Error
          </Badge>
        )
      case 'disconnected':
      default:
        return (
          <Badge variant="outline" className="text-xs bg-muted text-muted-foreground border-border">
            Disconnected
          </Badge>
        )
    }
  }

  const renderCardSkeleton = () => (
    <div
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      aria-hidden
    >
      <div className="flex-1 min-w-0 space-y-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full flex-shrink-0" />
          <Skeleton className="h-4 w-20" />
        </div>
        <Skeleton className="h-3 w-full max-w-[240px]" />
      </div>
    </div>
  )

  const renderAnnotationBadge = (key: string, value: boolean) => (
    <Badge
      key={key}
      variant="outline"
      className={`text-[10px] px-1.5 py-0 font-normal gap-0.5 ${
        value
          ? 'bg-green-50 text-green-700 border-green-200 dark:bg-green-950/30 dark:text-green-400 dark:border-green-800'
          : 'bg-red-50 text-red-700 border-red-200 dark:bg-red-950/30 dark:text-red-400 dark:border-red-800'
      }`}
    >
      {value ? <Check className="h-2.5 w-2.5" /> : <X className="h-2.5 w-2.5" />}
      {key}
    </Badge>
  )

  const renderToolRow = (tool: McpTool) => {
    const annotations = Object.entries(tool.annotations).filter(
      ([, v]) => typeof v === 'boolean'
    )
    return (
      <div key={tool.name} className="flex items-center justify-between gap-3 px-3 py-2">
        <code className="text-xs truncate">{tool.name}</code>
        {annotations.length > 0 && (
          <div className="flex items-center gap-1 flex-shrink-0">
            {annotations.map(([k, v]) => renderAnnotationBadge(k, v as boolean))}
          </div>
        )}
      </div>
    )
  }

  const renderServerCard = (server: McpServer) => {
    const tools = server.tools ?? []
    const toolCount = tools.length
    const isUserDefined = userServerNames.has(server.name)

    return (
      <div
        key={server.name}
        className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <div className="flex-shrink-0">
              {getStatusIcon(server)}
            </div>
            <h4 className="font-medium text-sm">{server.displayName}</h4>
            {isUserDefined && (
              <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                Custom
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-2 mt-1 ml-6">
            {server.version && (
              <span className="text-[10px] text-muted-foreground">v{server.version}</span>
            )}
            {toolCount > 0 && (
              <Popover>
                <PopoverTrigger asChild>
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
                  >
                    <Info className="h-3 w-3" />
                    <span>{toolCount} {toolCount === 1 ? 'tool' : 'tools'}</span>
                  </button>
                </PopoverTrigger>
                <PopoverContent
                  align="start"
                  className="w-80 p-0"
                >
                  <div className="px-3 py-2.5 border-b bg-muted/30">
                    <p className="text-xs font-medium">
                      {server.displayName} — {toolCount} {toolCount === 1 ? 'tool' : 'tools'}
                    </p>
                  </div>
                  <div className="max-h-72 overflow-y-auto">
                    {tools.map((tool) => renderToolRow(tool))}
                  </div>
                </PopoverContent>
              </Popover>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          {getStatusBadge(server)}
          {isUserDefined && isStopped && (
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => handleRemoveServer(server.name)}
              disabled={updateMcpServersMutation.isPending}
            >
              <Trash2 className="h-3 w-3 text-muted-foreground" />
            </Button>
          )}
        </div>
      </div>
    )
  }

  return (
    <AccordionItem value="mcp-servers" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Plug className="h-4 w-4" />
          <span>MCP Servers</span>
          {!showPlaceholders && mcpServers.length > 0 && (
            <Badge variant="outline" className="text-[10px] px-2 py-0.5">
              {mcpServers.length}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-2">
          {showPlaceholders ? (
            <>
              {renderCardSkeleton()}
              {renderCardSkeleton()}
            </>
          ) : mcpServers.length > 0 ? (
            mcpServers.map((server) => renderServerCard(server))
          ) : (
            <p className="text-xs text-muted-foreground py-2">
              No MCP servers available for this session.
            </p>
          )}

          {/* User-defined servers (shown when not running, from spec) */}
          {isStopped && !isRunning && specMcpServers.length > 0 && mcpServers.length === 0 && (
            specMcpServers.map((s) => (
              <div
                key={s.name}
                className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <AlertCircle className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                    <h4 className="font-medium text-sm">{s.name}</h4>
                    <Badge variant="outline" className="text-[10px] px-1.5 py-0">Custom</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground mt-0.5 ml-6">
                    {s.type === 'stdio' ? `${s.command} ${(s.args ?? []).join(' ')}` : s.url ?? 'HTTP server'}
                  </p>
                </div>
                <div className="flex items-center gap-2 flex-shrink-0">
                  <Badge variant="outline" className="text-xs bg-muted text-muted-foreground border-border">
                    Pending
                  </Badge>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6"
                    onClick={() => handleRemoveServer(s.name)}
                    disabled={updateMcpServersMutation.isPending}
                  >
                    <Trash2 className="h-3 w-3 text-muted-foreground" />
                  </Button>
                </div>
              </div>
            ))
          )}

          {/* Add MCP Server form (stopped sessions only) */}
          {isStopped && showAddForm && (
            <div className="space-y-2 p-3 border rounded-lg bg-background/50">
              <div className="flex items-end gap-2">
                <div className="flex-1 space-y-1">
                  <Label className="text-xs">Server Name</Label>
                  <Input
                    value={newServer.name}
                    onChange={(e) => setNewServer({ ...newServer, name: e.target.value })}
                    placeholder="my-server"
                    disabled={updateMcpServersMutation.isPending}
                  />
                </div>
                <div className="w-24 space-y-1">
                  <Label className="text-xs">Type</Label>
                  <Select
                    value={newServer.type ?? 'http'}
                    onValueChange={(v) => setNewServer({ ...newServer, type: v as 'http' | 'stdio' })}
                    disabled={updateMcpServersMutation.isPending}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">HTTP</SelectItem>
                      <SelectItem value="stdio">Stdio</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              {(newServer.type ?? 'http') === 'http' && (
                <div className="space-y-1">
                  <Label className="text-xs">URL</Label>
                  <Input
                    value={newServer.url ?? ''}
                    onChange={(e) => setNewServer({ ...newServer, url: e.target.value })}
                    placeholder="https://my-mcp-server.example.com/mcp"
                    disabled={updateMcpServersMutation.isPending}
                  />
                </div>
              )}
              {newServer.type === 'stdio' && (
                <>
                  <div className="space-y-1">
                    <Label className="text-xs">Command</Label>
                    <Input
                      value={newServer.command ?? ''}
                      onChange={(e) => setNewServer({ ...newServer, command: e.target.value })}
                      placeholder="uvx"
                      disabled={updateMcpServersMutation.isPending}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">Arguments</Label>
                    <Input
                      value={(newServer.args ?? []).join(' ')}
                      onChange={(e) => setNewServer({
                        ...newServer,
                        args: e.target.value.split(' ').filter((a) => a.length > 0),
                      })}
                      placeholder="my-mcp-package --flag"
                      disabled={updateMcpServersMutation.isPending}
                    />
                  </div>
                </>
              )}
              <div className="flex gap-2 pt-1">
                <Button
                  size="sm"
                  onClick={handleAddServer}
                  disabled={!newServer.name.trim() || updateMcpServersMutation.isPending}
                >
                  {updateMcpServersMutation.isPending ? (
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  ) : null}
                  Add
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowAddForm(false)
                    setNewServer({ name: '', type: 'http' })
                  }}
                  disabled={updateMcpServersMutation.isPending}
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}

          {isStopped && !showAddForm && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAddForm(true)}
              className="w-full"
            >
              <Plus className="h-3 w-3 mr-1" />
              Add MCP Server
            </Button>
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

// ─── Integrations Accordion ──────────────────────────────────────────────────

export function IntegrationsAccordion() {
  const { data: integrationsStatus, isPending } = useIntegrationsStatus()

  const githubConfigured = integrationsStatus?.github?.active != null
  const gitlabConfigured = integrationsStatus?.gitlab?.connected ?? false
  const jiraConfigured = integrationsStatus?.jira?.connected ?? false
  const googleConfigured = integrationsStatus?.google?.connected ?? false

  const integrations = [
    {
      key: 'github',
      name: 'GitHub',
      configured: githubConfigured,
      configuredMessage: 'Authenticated. Git push and repository access enabled.',
    },
    {
      key: 'gitlab',
      name: 'GitLab',
      configured: gitlabConfigured,
      configuredMessage: 'Authenticated. Git push and repository access enabled.',
    },
    {
      key: 'google',
      name: 'Google Workspace',
      configured: googleConfigured,
      configuredMessage: 'Authenticated. Drive, Calendar, and Gmail access enabled.',
    },
    {
      key: 'jira',
      name: 'Jira',
      configured: jiraConfigured,
      configuredMessage: 'Authenticated. Issue and project access enabled.',
    },
  ].sort((a, b) => a.name.localeCompare(b.name))

  const configuredCount = integrations.filter((i) => i.configured).length

  const renderCardSkeleton = () => (
    <div
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
      aria-hidden
    >
      <div className="flex-1 min-w-0 space-y-2">
        <div className="flex items-center gap-2">
          <Skeleton className="h-4 w-4 rounded-full flex-shrink-0" />
          <Skeleton className="h-4 w-20" />
        </div>
        <Skeleton className="h-3 w-full max-w-[240px]" />
      </div>
    </div>
  )

  const renderIntegrationCard = (integration: (typeof integrations)[number]) => (
    <div
      key={integration.key}
      className="flex items-start justify-between gap-3 p-3 border rounded-lg bg-background/50"
    >
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <div className="flex-shrink-0">
            {integration.configured ? (
              <CheckCircle2 className="h-4 w-4 text-green-600" />
            ) : (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <AlertTriangle className="h-4 w-4 text-amber-500" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>Not configured</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
          <h4 className="font-medium text-sm">{integration.name}</h4>
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">
          {integration.configured ? (
            integration.configuredMessage
          ) : (
            <>
              Not connected.{' '}
              <Link href="/integrations" className="text-primary hover:underline">
                Set up
              </Link>{' '}
              to enable {integration.name} access.
            </>
          )}
        </p>
      </div>
    </div>
  )

  return (
    <AccordionItem value="integrations" className="border rounded-lg px-3 bg-card">
      <AccordionTrigger className="text-base font-semibold hover:no-underline py-3">
        <div className="flex items-center gap-2">
          <Link2 className="h-4 w-4" />
          <span>Integrations</span>
          {!isPending && (
            <Badge variant="outline" className="text-[10px] px-2 py-0.5">
              {configuredCount}/{integrations.length}
            </Badge>
          )}
        </div>
      </AccordionTrigger>
      <AccordionContent className="px-1 pb-3">
        <div className="space-y-2">
          {isPending ? (
            <>
              {renderCardSkeleton()}
              {renderCardSkeleton()}
              {renderCardSkeleton()}
            </>
          ) : (
            integrations.map((integration) => renderIntegrationCard(integration))
          )}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

// ─── Legacy export (renders both) ────────────────────────────────────────────

type McpIntegrationsAccordionProps = {
  projectName: string
  sessionName: string
}

/** @deprecated Use McpServersAccordion + IntegrationsAccordion separately */
export function McpIntegrationsAccordion({
  projectName,
  sessionName,
}: McpIntegrationsAccordionProps) {
  return (
    <>
      <McpServersAccordion projectName={projectName} sessionName={sessionName} />
      <IntegrationsAccordion />
    </>
  )
}
