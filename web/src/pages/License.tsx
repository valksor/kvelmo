import { useState, useEffect, useMemo } from 'react'
import { Loader2, Scale, ChevronDown, ChevronRight, FileText } from 'lucide-react'

interface LicenseEntry {
  path: string
  license: string
  unknown: boolean
}

interface LicensesData {
  licenses: LicenseEntry[]
  count?: number
}

export default function License() {
  const [data, setData] = useState<LicensesData | null>(null)
  const [projectLicense, setProjectLicense] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expandedLicenses, setExpandedLicenses] = useState<Set<string>>(new Set())

  useEffect(() => {
    // Fetch project license (plain text)
    fetch('/api/v1/license')
      .then((res) => res.text())
      .then((text) => setProjectLicense(text))
      .catch(() => {}) // Ignore errors for project license

    // Fetch dependency licenses (JSON)
    fetch('/licenses.json')
      .then((res) => {
        if (!res.ok) throw new Error('Failed to load licenses')
        return res.json()
      })
      .then((json) => {
        setData(json)
        setLoading(false)
      })
      .catch((err) => {
        setError(err.message)
        setLoading(false)
      })
  }, [])

  // Group packages by license type
  const groupedByLicense = useMemo(() => {
    if (!data?.licenses) return new Map<string, LicenseEntry[]>()

    const groups = new Map<string, LicenseEntry[]>()
    for (const pkg of data.licenses) {
      const license = pkg.license || 'Unknown'
      if (!groups.has(license)) {
        groups.set(license, [])
      }
      groups.get(license)!.push(pkg)
    }

    // Sort groups by size (most common first)
    return new Map([...groups.entries()].sort((a, b) => b[1].length - a[1].length))
  }, [data])

  const toggleLicense = (license: string) => {
    setExpandedLicenses((prev) => {
      const next = new Set(prev)
      if (next.has(license)) {
        next.delete(license)
      } else {
        next.add(license)
      }
      return next
    })
  }

  const expandAll = () => {
    setExpandedLicenses(new Set(groupedByLicense.keys()))
  }

  const collapseAll = () => {
    setExpandedLicenses(new Set())
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Open Source Licenses</h1>
        <div className="alert alert-error">
          <span>{error}</span>
        </div>
      </div>
    )
  }

  const totalPackages = data?.licenses?.length || 0

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Open Source Licenses</h1>
          <p className="text-base-content/60 mt-1">
            This software includes {totalPackages} open source packages
          </p>
        </div>
        <div className="flex gap-2">
          <button onClick={expandAll} className="btn btn-ghost btn-sm">
            Expand All
          </button>
          <button onClick={collapseAll} className="btn btn-ghost btn-sm">
            Collapse All
          </button>
        </div>
      </div>

      {/* Project License */}
      {projectLicense && (
        <div className="card bg-base-100 shadow-sm">
          <div className="card-body">
            <div className="flex items-center gap-2">
              <FileText size={20} className="text-primary" />
              <h2 className="card-title">Mehrhof License</h2>
            </div>
            <pre className="bg-base-200 p-4 rounded-lg text-sm overflow-x-auto whitespace-pre-wrap font-mono text-base-content/80 max-h-96 overflow-y-auto">
              {projectLicense}
            </pre>
          </div>
        </div>
      )}

      {/* License Summary */}
      <div className="stats shadow w-full">
        <div className="stat">
          <div className="stat-title">Total Packages</div>
          <div className="stat-value text-primary">{totalPackages}</div>
        </div>
        <div className="stat">
          <div className="stat-title">License Types</div>
          <div className="stat-value">{groupedByLicense.size}</div>
        </div>
        <div className="stat">
          <div className="stat-title">Most Common</div>
          <div className="stat-value text-sm">
            {groupedByLicense.size > 0 ? [...groupedByLicense.keys()][0] : 'N/A'}
          </div>
        </div>
      </div>

      {/* License Groups */}
      <div className="space-y-3">
        {[...groupedByLicense.entries()].map(([license, packages]) => {
          const isExpanded = expandedLicenses.has(license)
          return (
            <div key={license} className="card bg-base-100 shadow-sm">
              <div
                className="card-body py-3 px-4 cursor-pointer"
                onClick={() => toggleLicense(license)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    {isExpanded ? (
                      <ChevronDown size={20} className="text-base-content/50" />
                    ) : (
                      <ChevronRight size={20} className="text-base-content/50" />
                    )}
                    <Scale size={18} className="text-primary" />
                    <span className="font-medium">{license}</span>
                    <span className="badge badge-sm badge-primary">{packages.length}</span>
                  </div>
                </div>

                {isExpanded && (
                  <div className="mt-4 ml-9 space-y-2">
                    {packages.map((pkg, idx) => (
                      <div
                        key={`${pkg.path}-${idx}`}
                        className="flex items-center justify-between py-2 px-3 bg-base-200 rounded-lg"
                      >
                        <span className="font-mono text-sm">{pkg.path}</span>
                        {pkg.unknown && (
                          <span className="badge badge-warning badge-sm">Unknown</span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* Footer */}
      <div className="text-center text-sm text-base-content/50 py-4">
        <p>
          This list is automatically generated from project dependencies.
          <br />
          For full license texts, please refer to each package's repository.
        </p>
      </div>
    </div>
  )
}
