import React, { useState } from 'react';
import * as XLSX from 'xlsx';
import './App.css';

// Get API URL from environment variable
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function App() {
  const [file, setFile] = useState(null);
  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [progress, setProgress] = useState({ current: 0, total: 0 });
  const [activeTab, setActiveTab] = useState('all'); // 'all', 'valid', 'invalid', 'blocked'

  // Handle file upload
  const handleFileUpload = (e) => {
    const uploadedFile = e.target.files[0];
    if (uploadedFile) {
      const fileType = uploadedFile.name.split('.').pop().toLowerCase();
      if (fileType === 'xlsx' || fileType === 'xls') {
        setFile(uploadedFile);
        setError(null);
      } else {
        setError('Please upload a valid Excel file (.xlsx or .xls)');
        setFile(null);
      }
    }
  };

  // Parse Excel file
  const parseExcelFile = (file) => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      
      reader.onload = (e) => {
        try {
          const data = new Uint8Array(e.target.result);
          const workbook = XLSX.read(data, { type: 'array' });
          const firstSheet = workbook.Sheets[workbook.SheetNames[0]];
          const jsonData = XLSX.utils.sheet_to_json(firstSheet);
          
          // Validate required columns
          if (jsonData.length === 0) {
            reject(new Error('Excel file is empty'));
            return;
          }
          
          const firstRow = jsonData[0];
          const hasCollegeColumn = 'college' in firstRow || 'College' in firstRow;
          const hasLinksColumn = 'links' in firstRow || 'Links' in firstRow || 'link' in firstRow || 'Link' in firstRow;
          
          if (!hasCollegeColumn || !hasLinksColumn) {
            reject(new Error('Excel file must have "college" and "links" columns'));
            return;
          }
          
          resolve(jsonData);
        } catch (err) {
          reject(new Error('Failed to parse Excel file: ' + err.message));
        }
      };
      
      reader.onerror = () => {
        reject(new Error('Failed to read file'));
      };
      
      reader.readAsArrayBuffer(file);
    });
  };

  // Handle validation
  const handleValidate = async () => {
    if (!file) {
      setError('Please upload an Excel file');
      return;
    }

    setError(null);
    setResults(null);
    setLoading(true);

    try {
      // Parse Excel file
      const jsonData = await parseExcelFile(file);
      
      // Process data to extract college and links
      const dataToValidate = [];
      jsonData.forEach((row) => {
        const college = row.college || row.College || '';
        const linksStr = row.links || row.Links || row.link || row.Link || '';
        
        if (!college || !linksStr) {
          return;
        }
        
        // Split links by comma, newline, or semicolon
        const links = linksStr.toString()
          .split(/[,;\n]+/)
          .map(link => link.trim())
          .filter(link => link.length > 0);
        
        links.forEach(link => {
          dataToValidate.push({
            college: college.trim(),
            url: link
          });
        });
      });

      if (dataToValidate.length === 0) {
        setError('No valid links found in the Excel file');
        setLoading(false);
        return;
      }

      if (dataToValidate.length > 500) {
        setError('Too many links. Maximum 500 links allowed per file');
        setLoading(false);
        return;
      }

      setProgress({ current: 0, total: dataToValidate.length });

      // Send to backend
      const response = await fetch(`${API_URL}/api/validate`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ data: dataToValidate }),
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
      .map(r => `${r.college}: ${r.url}`)
      .join('\n');
    
    navigator.clipboard.writeText(invalidLinks);
    alert('Invalid and blocked links copied to clipboard!');
  };

  // Export results as CSV
  const exportAsCSV = () => {
    if (!results || !results.results) return;
    
    const headers = ['Index', 'College', 'URL', 'Status'];
    const rows = results.results.map(r => [
      r.index,
      r.college,
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

  // Download template Excel file
  const downloadTemplate = () => {
    // Create template data
    const templateData = [
      {
        college: 'Example College Name 1',
        links: 'https://example.com/document1.pdf, https://example.com/document2.pdf'
      },
      {
        college: 'Example College Name 2',
        links: 'https://example.com/document3.pdf'
      }
    ];

    // Create workbook and worksheet
    const worksheet = XLSX.utils.json_to_sheet(templateData);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'Template');

    // Set column widths for better readability
    worksheet['!cols'] = [
      { wch: 30 }, // college column width
      { wch: 80 }  // links column width
    ];

    // Generate Excel file and trigger download
    XLSX.writeFile(workbook, 'mandatory_disclosure_links.xlsx');
  };

  // Group results by college
  const getResultsByCollege = () => {
    if (!results || !results.results) return {};
    
    // Filter results based on active tab
    let filteredResults = results.results;
    if (activeTab === 'valid') {
      filteredResults = results.results.filter(r => r.status === 'Valid');
    } else if (activeTab === 'invalid') {
      filteredResults = results.results.filter(r => r.status === 'Invalid');
    } else if (activeTab === 'blocked') {
      filteredResults = results.results.filter(r => r.status === 'Blocked by Server');
    }
    
    const grouped = {};
    filteredResults.forEach(result => {
      if (!grouped[result.college]) {
        grouped[result.college] = {
          valid: 0,
          invalid: 0,
          blocked: 0,
          links: []
        };
      }
      grouped[result.college].links.push(result);
      
      if (result.status === 'Valid') {
        grouped[result.college].valid++;
      } else if (result.status === 'Blocked by Server') {
        grouped[result.college].blocked++;
      } else {
        grouped[result.college].invalid++;
      }
    });
    
    return grouped;
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
          {/* File Upload Section */}
          <div className="mb-8">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Upload Excel File (with "college" and "links" columns)
            </label>
            <div className="flex items-center justify-center w-full">
              <label className="flex flex-col items-center justify-center w-full h-40 border-2 border-gray-300 border-dashed rounded-lg cursor-pointer bg-gray-50 hover:bg-gray-100 transition-colors">
                <div className="flex flex-col items-center justify-center pt-5 pb-6">
                  <svg className="w-10 h-10 mb-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path>
                  </svg>
                  <p className="mb-2 text-sm text-gray-500">
                    {file ? (
                      <span className="font-semibold text-indigo-600">{file.name}</span>
                    ) : (
                      <>
                        <span className="font-semibold">Click to upload</span> or drag and drop
                      </>
                    )}
                  </p>
                  <p className="text-xs text-gray-500">Excel files (.xlsx, .xls)</p>
                </div>
                <input
                  type="file"
                  className="hidden"
                  accept=".xlsx,.xls"
                  onChange={handleFileUpload}
                  disabled={loading}
                />
              </label>
            </div>
            <div className="flex items-center justify-between mt-2">
              <p className="text-xs text-gray-500">
                Excel file should have two columns: "college" and "links". Links can be comma-separated in a single cell.
              </p>
              <button
                onClick={downloadTemplate}
                className="text-xs text-indigo-600 hover:text-indigo-800 font-medium underline flex items-center gap-1"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                </svg>
                Download Template
              </button>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap gap-3 mb-8">
            <button
              onClick={handleValidate}
              disabled={loading || !file}
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
              <h2 className="text-lg font-semibold text-gray-900 mb-4">Validation Results by College</h2>
              
              {/* Tab Buttons */}
              <div className="flex flex-wrap gap-2">
                <button
                  onClick={() => setActiveTab('all')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    activeTab === 'all'
                      ? 'bg-indigo-600 text-white shadow-md'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  All ({results.total})
                </button>
                <button
                  onClick={() => setActiveTab('valid')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    activeTab === 'valid'
                      ? 'bg-green-600 text-white shadow-md'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  Valid ({results.valid})
                </button>
                <button
                  onClick={() => setActiveTab('invalid')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    activeTab === 'invalid'
                      ? 'bg-red-600 text-white shadow-md'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  Invalid ({results.invalid})
                </button>
                <button
                  onClick={() => setActiveTab('blocked')}
                  className={`px-4 py-2 rounded-lg font-medium transition-all ${
                    activeTab === 'blocked'
                      ? 'bg-orange-600 text-white shadow-md'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  Blocked ({results.blocked || 0})
                </button>
              </div>
            </div>
            
            {/* Grouped by College */}
            {Object.entries(getResultsByCollege()).length === 0 ? (
              <div className="px-6 py-12 text-center">
                <div className="text-gray-400 mb-2">
                  <svg className="w-16 h-16 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
                <p className="text-gray-500 text-lg font-medium">No {activeTab === 'all' ? '' : activeTab} results found</p>
              </div>
            ) : (
              Object.entries(getResultsByCollege()).map(([college, data], idx) => (
              <div key={idx} className="border-b border-gray-200 last:border-b-0">
                <div className="bg-gradient-to-r from-indigo-50 to-purple-50 px-6 py-4">
                  <div className="flex items-center justify-between flex-wrap gap-2">
                    <h3 className="text-base font-semibold text-gray-900">{college}</h3>
                    <div className="flex gap-4 text-sm">
                      <span className="text-green-700">
                        <span className="font-semibold">{data.valid}</span> Valid
                      </span>
                      <span className="text-red-700">
                        <span className="font-semibold">{data.invalid}</span> Invalid
                      </span>
                      {data.blocked > 0 && (
                        <span className="text-orange-700">
                          <span className="font-semibold">{data.blocked}</span> Blocked
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead className="bg-gray-50 border-b border-gray-200">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          #
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          URL
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Status
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-100">
                      {data.links.map((result, linkIdx) => (
                        <tr key={linkIdx} className="hover:bg-gray-50 transition-colors">
                          <td className="px-6 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
                            {result.index}
                          </td>
                          <td className="px-6 py-3 text-sm text-gray-900">
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
                          <td className="px-6 py-3 whitespace-nowrap text-sm">
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
            ))
            )}
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
