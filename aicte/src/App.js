import React, { useState } from 'react';
import './App.css';

function App() {
  const [urlInput, setUrlInput] = useState('');
  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [progress, setProgress] = useState({ current: 0, total: 0 });

  // Parse input URLs from textarea
  const parseUrls = (input) => {
    return input
      .split(/[\n,]+/) // Split by newline or comma
      .map(url => url.trim()) // Trim whitespace
      .filter(url => url.length > 0); // Remove empty lines
  };

  // Handle validation
  const handleValidate = async () => {
    setError(null);
    setResults(null);
    
    const urls = parseUrls(urlInput);
    
    // Validate input
    if (urls.length === 0) {
      setError('Please enter at least one URL');
      return;
    }

    if (urls.length > 100) {
      setError('Maximum 100 URLs allowed per request');
      return;
    }

    setLoading(true);
    setProgress({ current: 0, total: urls.length });

    try {
      const response = await fetch('http://localhost:8080/api/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ urls }),
      });

      if (!response.ok) {
        throw new Error(`Server error: ${response.status}`);
      }

      const data = await response.json();
      setResults(data);
      setProgress({ current: data.total, total: data.total });
    } catch (err) {
      setError(err.message || 'Failed to validate links. Make sure the backend server is running.');
    } finally {
      setLoading(false);
    }
  };

  // Copy invalid links to clipboard
  const copyInvalidLinks = () => {
    if (!results || !results.results) return;
    
    const invalidLinks = results.results
      .filter(r => r.status === 'Invalid' || r.status === 'Blocked by Server')
      .map(r => r.url)
      .join('\n');
    
    navigator.clipboard.writeText(invalidLinks);
    alert('Invalid and blocked links copied to clipboard!');
  };

  // Export results as CSV
  const exportAsCSV = () => {
    if (!results || !results.results) return;
    
    const headers = ['Index', 'URL', 'Status'];
    const rows = results.results.map(r => [
      r.index,
      r.url,
      r.status
    ]);
    
    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.map(cell => `"${cell}"`).join(','))
    ].join('\n');
    
    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `pdf-validation-results-${Date.now()}.csv`;
    a.click();
    window.URL.revokeObjectURL(url);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 py-12 px-4">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-3">
            PDF Link Validator
          </h1>
          <p className="text-gray-600">
            Validate multiple PDF links at once
          </p>
        </div>

        {/* Main Card */}
        <div className="bg-white rounded-xl shadow-lg border border-gray-200 p-8 mb-8">
          {/* Input Section */}
          <div className="mb-8">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Enter URLs (one per line or comma-separated)
            </label>
            <textarea
              className="w-full h-40 px-4 py-3 bg-gray-50 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 focus:bg-white resize-none font-mono text-sm transition-all shadow-sm"
              placeholder="https://example.com/document1.pdf&#10;https://example.com/document2.pdf&#10;https://example.com/document3.pdf"
              value={urlInput}
              onChange={(e) => setUrlInput(e.target.value)}
              disabled={loading}
            />
            <p className="text-xs text-gray-500 mt-2">
              You can paste up to 100 URLs at once
            </p>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-3 mb-8">
            <button
              onClick={handleValidate}
              disabled={loading || !urlInput.trim()}
              className={`flex-1 min-w-[200px] ${
                loading 
                  ? 'bg-indigo-400 cursor-wait' 
                  : 'bg-indigo-600 hover:bg-indigo-700'
              } disabled:bg-gray-300 disabled:cursor-not-allowed text-white font-medium py-3 px-6 rounded-lg transition-all shadow-md hover:shadow-lg flex items-center justify-center gap-2`}
            >
              {loading ? (
                <>
                  <svg className="animate-spin h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                  </svg>
                  <span>Validating...</span>
                </>
              ) : (
                <span>Validate Links</span>
              )}
            </button>

            {results && (
              <>
                <button
                  onClick={copyInvalidLinks}
                  className="bg-gray-700 hover:bg-gray-800 disabled:bg-gray-300 disabled:cursor-not-allowed text-white font-medium py-3 px-6 rounded-lg transition-all shadow-md hover:shadow-lg"
                  disabled={results.invalid === 0}
                >
                  Copy Invalid Links
                </button>
                <button
                  onClick={exportAsCSV}
                  className="bg-white border-2 border-gray-300 text-gray-700 hover:bg-gray-50 hover:border-gray-400 font-medium py-3 px-6 rounded-lg transition-all shadow-md hover:shadow-lg"
                >
                  Export CSV
                </button>
              </>
            )}
          </div>

          {/* Progress Indicator */}
          {loading && progress.total > 0 && (
            <div className="mb-8 bg-gradient-to-r from-indigo-50 to-purple-50 rounded-lg p-4 border border-indigo-200">
              <div className="flex justify-between mb-2">
                <span className="text-sm font-medium text-indigo-700">Validating URLs...</span>
                <span className="text-sm font-medium text-indigo-600">{progress.current}/{progress.total}</span>
              </div>
              <div className="w-full bg-indigo-100 rounded-full h-2.5 overflow-hidden shadow-inner">
                <div
                  className="bg-gradient-to-r from-indigo-500 to-purple-500 h-2.5 rounded-full transition-all duration-300 shadow-sm"
                  style={{ width: `${(progress.current / progress.total) * 100}%` }}
                ></div>
              </div>
            </div>
          )}

          {/* Error Display */}
          {error && (
            <div className="bg-red-50 border-l-4 border-red-500 rounded-r-lg p-4 mb-8 shadow-md">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd"/>
                  </svg>
                </div>
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800">Error</h3>
                  <p className="text-sm text-red-700 mt-1">{error}</p>
                </div>
              </div>
            </div>
          )}

          {/* Summary Cards */}
          {results && (
            <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
              <div className="bg-gradient-to-br from-white to-gray-50 rounded-xl p-6 border border-gray-200 shadow-md hover:shadow-lg transition-shadow">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider">Total URLs</div>
                <div className="text-3xl font-semibold text-gray-900 mt-2">{results.total}</div>
              </div>
              <div className="bg-gradient-to-br from-green-50 to-white rounded-xl p-6 border border-green-200 shadow-md hover:shadow-lg transition-shadow">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider">Valid</div>
                <div className="text-3xl font-semibold text-green-600 mt-2">{results.valid}</div>
              </div>
              <div className="bg-gradient-to-br from-red-50 to-white rounded-xl p-6 border border-red-200 shadow-md hover:shadow-lg transition-shadow">
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider">Invalid</div>
                <div className="text-3xl font-semibold text-red-600 mt-2">{results.invalid}</div>
              </div>
              <div 
                className="bg-gradient-to-br from-orange-50 to-white rounded-xl p-6 border border-orange-200 cursor-pointer shadow-md hover:shadow-lg transition-shadow"
                onClick={() => {
                  alert(`Blocked by Server

This link could not be accessed programmatically because the hosting server rejected the request. Some websites use security systems (such as CDN protection, firewall rules, or bot detection) that block automated tools from downloading files directly.

The file may still open normally in a web browser, but it does not allow backend or automated access.

Possible reasons:
	•	Bot protection (e.g., Cloudflare, Akamai)
	•	Access restrictions or firewall rules
	•	IP-based blocking
	•	Anti-scraping security policies

To validate successfully, the link must be publicly accessible as a direct downloadable PDF without server restrictions.`);
                }}
              >
                <div className="text-xs font-medium text-gray-500 uppercase tracking-wider">Blocked</div>
                <div className="text-3xl font-semibold text-orange-600 mt-2">{results.blocked || 0}</div>
              </div>
            </div>
          )}
        </div>

        {/* Results Table */}
        {results && results.results && results.results.length > 0 && (
          <div className="bg-white rounded-xl shadow-lg border border-gray-200 overflow-hidden">
            <div className="px-6 py-5 bg-gradient-to-r from-gray-50 to-white border-b border-gray-200">
              <h2 className="text-lg font-semibold text-gray-900">Validation Results</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50 border-b-2 border-gray-200">
                  <tr>
                    <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      #
                    </th>
                    <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      URL
                    </th>
                    <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {results.results.map((result, idx) => (
                    <tr key={idx} className="hover:bg-gray-50 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        {result.index}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-900">
                        <div className="max-w-2xl truncate" title={result.url}>
                          <a 
                            href={result.url} 
                            target="_blank" 
                            rel="noopener noreferrer"
                            className="text-indigo-600 hover:text-indigo-800 hover:underline transition-colors"
                          >
                            {result.url}
                          </a>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        <span className={`inline-flex items-center px-3 py-1.5 rounded-full text-xs font-medium shadow-sm ${
                          result.status === 'Valid' 
                            ? 'bg-green-100 text-green-800 border border-green-200' 
                            : result.status === 'Blocked by Server'
                            ? 'bg-orange-100 text-orange-800 border border-orange-200'
                            : 'bg-red-100 text-red-800 border border-red-200'
                        }`}>
                          {result.status === 'Valid' && 'Valid'}
                          {result.status === 'Invalid' && 'Invalid'}
                          {result.status === 'Blocked by Server' && 'Blocked'}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="text-center mt-8">
          <p className="text-sm text-gray-500">
            Built with React + Go
          </p>
        </div>
      </div>
    </div>
  );
}

export default App;
