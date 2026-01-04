"""Parse gosec report and display issues in a readable format."""

import json
import os

def main():
    # Try to read the most recent report
    report_files = [
        'reports/go/gosec-report-after-fix',
        'reports/go/gosec-report'
    ]
    
    report_file = None
    for f in report_files:
        if os.path.exists(f):
            report_file = f
            break
    
    if not report_file:
        print("No gosec report found!")
        return
        
    with open(report_file, 'r') as f:
        data = json.load(f)
    
    issues = data.get('Issues', [])
    
    print(f"=== GOSEC SECURITY REPORT ===")
    print(f"Total issues found: {len(issues)}\n")
    
    for idx, issue in enumerate(issues, 1):
        severity = issue.get('severity', 'UNKNOWN')
        confidence = issue.get('confidence', 'UNKNOWN')
        rule_id = issue.get('rule_id', 'N/A')
        file_path = issue.get('file', 'N/A')
        line = issue.get('line', 'N/A')
        details = issue.get('details', 'No details')
        cwe_id = issue.get('cwe', {}).get('id', 'N/A')
        
        print(f"--- Issue #{idx} ---")
        print(f"  Severity: {severity} | Confidence: {confidence}")
        print(f"  Rule: {rule_id} | CWE-{cwe_id}")
        print(f"  File: {file_path}:{line}")
        print(f"  Details: {details}")
        print()

if __name__ == "__main__":
    main()
