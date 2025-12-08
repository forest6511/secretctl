import { useState, useEffect, useCallback } from 'react'
import {
  RefreshCw, Download, CheckCircle, XCircle, AlertCircle,
  ChevronLeft, ChevronRight, Filter, Calendar, Search
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  SearchAuditLogs, VerifyAuditLogs, GetAuditLogStats
} from '../../wailsjs/go/main/App'
import { main } from '../../wailsjs/go/models'

interface AuditPageProps {
  onNavigateBack: () => void
}

const ACTION_OPTIONS = [
  { value: '', label: 'All Actions' },
  { value: 'secret.get', label: 'Get Secret' },
  { value: 'secret.set', label: 'Set Secret' },
  { value: 'secret.delete', label: 'Delete Secret' },
  { value: 'secret.list', label: 'List Secrets' },
  { value: 'auth.unlock', label: 'Unlock' },
  { value: 'auth.lock', label: 'Lock' },
  { value: 'vault.init', label: 'Vault Init' },
]

const SOURCE_OPTIONS = [
  { value: '', label: 'All Sources' },
  { value: 'cli', label: 'CLI' },
  { value: 'mcp', label: 'MCP' },
  { value: 'ui', label: 'Desktop' },
]

const PAGE_SIZE = 20

export function AuditPage({ onNavigateBack }: AuditPageProps) {
  const [logs, setLogs] = useState<main.AuditLogEntry[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [currentPage, setCurrentPage] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [verificationStatus, setVerificationStatus] = useState<boolean | null>(null)
  const [isVerifying, setIsVerifying] = useState(false)
  const [stats, setStats] = useState<Record<string, number> | null>(null)
  const [selectedLog, setSelectedLog] = useState<main.AuditLogEntry | null>(null)

  // Filter state
  const [filterAction, setFilterAction] = useState('')
  const [filterSource, setFilterSource] = useState('')
  const [filterKey, setFilterKey] = useState('')
  const [filterStartDate, setFilterStartDate] = useState('')
  const [filterEndDate, setFilterEndDate] = useState('')

  const loadLogs = useCallback(async () => {
    setIsLoading(true)
    try {
      const filter: main.AuditLogFilter = {
        action: filterAction,
        source: filterSource,
        key: filterKey,
        startTime: filterStartDate ? new Date(filterStartDate).toISOString() : '',
        endTime: filterEndDate ? new Date(filterEndDate + 'T23:59:59').toISOString() : '',
      }
      const result = await SearchAuditLogs(filter, PAGE_SIZE, currentPage * PAGE_SIZE)
      setLogs(result.entries || [])
      setTotalCount(result.total)
    } catch (err) {
      console.error('Failed to load audit logs:', err)
    } finally {
      setIsLoading(false)
    }
  }, [filterAction, filterSource, filterKey, filterStartDate, filterEndDate, currentPage])

  const loadStats = async () => {
    try {
      const s = await GetAuditLogStats()
      setStats(s)
    } catch (err) {
      console.error('Failed to load stats:', err)
    }
  }

  const verifyChain = async () => {
    setIsVerifying(true)
    try {
      const valid = await VerifyAuditLogs()
      setVerificationStatus(valid)
    } catch (err) {
      console.error('Failed to verify:', err)
      setVerificationStatus(false)
    } finally {
      setIsVerifying(false)
    }
  }

  useEffect(() => {
    loadLogs()
  }, [loadLogs])

  useEffect(() => {
    loadStats()
    verifyChain()
  }, [])

  const handleFilter = () => {
    setCurrentPage(0)
    loadLogs()
  }

  const handleClearFilters = () => {
    setFilterAction('')
    setFilterSource('')
    setFilterKey('')
    setFilterStartDate('')
    setFilterEndDate('')
    setCurrentPage(0)
  }

  const exportToCSV = () => {
    const headers = ['Timestamp', 'Action', 'Source', 'Key', 'Status', 'Error']
    const rows = logs.map(log => [
      log.timestamp,
      log.action,
      log.source,
      log.key || '',
      log.success ? 'Success' : 'Failure',
      log.error || ''
    ])
    const csv = [headers.join(','), ...rows.map(r => r.map(c => `"${c}"`).join(','))].join('\n')
    downloadFile(csv, 'audit-logs.csv', 'text/csv')
  }

  const exportToJSON = () => {
    const json = JSON.stringify(logs, null, 2)
    downloadFile(json, 'audit-logs.json', 'application/json')
  }

  const downloadFile = (content: string, filename: string, mimeType: string) => {
    const blob = new Blob([content], { type: mimeType })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  const formatTimestamp = (ts: string) => {
    const date = new Date(ts)
    return date.toLocaleString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  }

  const totalPages = Math.ceil(totalCount / PAGE_SIZE)

  return (
    <div className="flex flex-col h-screen macos-titlebar-padding" data-testid="audit-page">
      {/* Header */}
      <div className="border-b border-border p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="icon" onClick={onNavigateBack} data-testid="back-button">
              <ChevronLeft className="w-5 h-5" />
            </Button>
            <h1 className="text-xl font-semibold">Audit Log</h1>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={verifyChain}
              disabled={isVerifying}
              data-testid="verify-chain-button"
            >
              <RefreshCw className={`w-4 h-4 mr-2 ${isVerifying ? 'animate-spin' : ''}`} />
              Verify Chain
            </Button>
            <Button variant="outline" size="sm" onClick={exportToCSV} data-testid="export-csv-button">
              <Download className="w-4 h-4 mr-2" />
              CSV
            </Button>
            <Button variant="outline" size="sm" onClick={exportToJSON} data-testid="export-json-button">
              <Download className="w-4 h-4 mr-2" />
              JSON
            </Button>
          </div>
        </div>

        {/* Verification Status */}
        <div className="mt-3 flex items-center gap-4">
          <div className="flex items-center gap-2" data-testid="chain-status">
            {verificationStatus === null ? (
              <>
                <AlertCircle className="w-4 h-4 text-muted-foreground" />
                <span className="text-sm text-muted-foreground">Checking chain integrity...</span>
              </>
            ) : verificationStatus ? (
              <>
                <CheckCircle className="w-4 h-4 text-green-500" />
                <span className="text-sm text-green-500">Chain Verified</span>
              </>
            ) : (
              <>
                <XCircle className="w-4 h-4 text-destructive" />
                <span className="text-sm text-destructive">Chain Invalid</span>
              </>
            )}
          </div>
          {stats && (
            <div className="text-sm text-muted-foreground">
              Total: {stats.total} | Success: {stats.success} | Failure: {stats.failure}
            </div>
          )}
        </div>
      </div>

      {/* Filters */}
      <div className="border-b border-border p-4 bg-muted/30">
        <div className="flex items-center gap-3 flex-wrap">
          <Filter className="w-4 h-4 text-muted-foreground" />

          <select
            value={filterAction}
            onChange={(e) => setFilterAction(e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-3 text-sm"
            data-testid="filter-action"
          >
            {ACTION_OPTIONS.map(opt => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>

          <select
            value={filterSource}
            onChange={(e) => setFilterSource(e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-3 text-sm"
            data-testid="filter-source"
          >
            {SOURCE_OPTIONS.map(opt => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>

          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Key..."
              value={filterKey}
              onChange={(e) => setFilterKey(e.target.value)}
              className="pl-9 w-40"
              data-testid="filter-key"
            />
          </div>

          <div className="flex items-center gap-1">
            <Calendar className="w-4 h-4 text-muted-foreground" />
            <Input
              type="date"
              value={filterStartDate}
              onChange={(e) => setFilterStartDate(e.target.value)}
              className="w-36"
              data-testid="filter-start-date"
            />
            <span className="text-muted-foreground">-</span>
            <Input
              type="date"
              value={filterEndDate}
              onChange={(e) => setFilterEndDate(e.target.value)}
              className="w-36"
              data-testid="filter-end-date"
            />
          </div>

          <Button size="sm" onClick={handleFilter} data-testid="apply-filter-button">
            Apply
          </Button>
          <Button size="sm" variant="ghost" onClick={handleClearFilters} data-testid="clear-filter-button">
            Clear
          </Button>
        </div>
      </div>

      {/* Log Table */}
      <div className="flex-1 overflow-auto p-4">
        <div className="border rounded-lg overflow-hidden" data-testid="audit-log-table">
          <table className="w-full text-sm">
            <thead className="bg-muted/50">
              <tr>
                <th className="text-left p-3 font-medium">Timestamp</th>
                <th className="text-left p-3 font-medium">Action</th>
                <th className="text-left p-3 font-medium">Source</th>
                <th className="text-left p-3 font-medium">Key</th>
                <th className="text-left p-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={5} className="p-8 text-center text-muted-foreground">
                    Loading...
                  </td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-8 text-center text-muted-foreground">
                    No audit logs found
                  </td>
                </tr>
              ) : (
                logs.map((log, idx) => (
                  <tr
                    key={`${log.timestamp}-${idx}`}
                    className="border-t border-border hover:bg-muted/30 cursor-pointer"
                    onClick={() => setSelectedLog(log)}
                    data-testid={`audit-log-row-${idx}`}
                  >
                    <td className="p-3 font-mono text-xs">{formatTimestamp(log.timestamp)}</td>
                    <td className="p-3">{log.action}</td>
                    <td className="p-3">
                      <span className={`px-2 py-0.5 rounded text-xs ${
                        log.source === 'cli' ? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300' :
                        log.source === 'mcp' ? 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300' :
                        'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                      }`}>
                        {log.source}
                      </span>
                    </td>
                    <td className="p-3 font-mono text-xs truncate max-w-xs">{log.key || '-'}</td>
                    <td className="p-3">
                      {log.success ? (
                        <CheckCircle className="w-4 h-4 text-green-500" />
                      ) : (
                        <XCircle className="w-4 h-4 text-destructive" />
                      )}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pagination */}
      <div className="border-t border-border p-4 flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          Showing {currentPage * PAGE_SIZE + 1}-{Math.min((currentPage + 1) * PAGE_SIZE, totalCount)} of {totalCount}
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setCurrentPage(p => Math.max(0, p - 1))}
            disabled={currentPage === 0}
            data-testid="prev-page-button"
          >
            <ChevronLeft className="w-4 h-4" />
            Prev
          </Button>
          <span className="text-sm">
            Page {currentPage + 1} of {totalPages || 1}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setCurrentPage(p => p + 1)}
            disabled={currentPage >= totalPages - 1}
            data-testid="next-page-button"
          >
            Next
            <ChevronRight className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {/* Detail Modal */}
      {selectedLog && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
          onClick={() => setSelectedLog(null)}
          data-testid="audit-detail-modal"
        >
          <Card className="w-full max-w-lg mx-4" onClick={e => e.stopPropagation()}>
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span>Audit Log Detail</span>
                <Button variant="ghost" size="sm" onClick={() => setSelectedLog(null)}>
                  &times;
                </Button>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <label className="text-sm font-medium text-muted-foreground">Timestamp</label>
                <p className="font-mono text-sm">{formatTimestamp(selectedLog.timestamp)}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">Action</label>
                <p>{selectedLog.action}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">Source</label>
                <p>{selectedLog.source}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">Key</label>
                <p className="font-mono text-sm">{selectedLog.key || '-'}</p>
              </div>
              <div>
                <label className="text-sm font-medium text-muted-foreground">Status</label>
                <div className="flex items-center gap-2">
                  {selectedLog.success ? (
                    <>
                      <CheckCircle className="w-4 h-4 text-green-500" />
                      <span className="text-green-500">Success</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="w-4 h-4 text-destructive" />
                      <span className="text-destructive">Failure</span>
                    </>
                  )}
                </div>
              </div>
              {selectedLog.error && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Error</label>
                  <p className="text-sm text-destructive">{selectedLog.error}</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  )
}
