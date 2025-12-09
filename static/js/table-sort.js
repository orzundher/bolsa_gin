/**
 * Generic table sorting functionality
 * Provides sorting capabilities for tables with sortable columns
 */
class TableSorter {
    constructor() {
        this.init();
    }

    getCellValue(tr, idx) {
        const td = tr.children[idx];
        if (td) {
            const child = td.querySelector('strong');
            return child ? child.innerText : td.innerText;
        }
        return '';
    }

    parseValue(value) {
        // Parse date format: DD MMM YYYY or DD MMM YYYY HH:MM
        const dateMatch = value.match(/^(\d{2}) (\w{3}) (\d{4})(?:\s+(\d{2}):(\d{2}))?$/);
        if (dateMatch) {
            const day = parseInt(dateMatch[1], 10);
            const monthStr = dateMatch[2];
            const year = parseInt(dateMatch[3], 10);
            const hour = dateMatch[4] ? parseInt(dateMatch[4], 10) : 0;
            const minute = dateMatch[5] ? parseInt(dateMatch[5], 10) : 0;
            const monthMap = { 
                'Jan': 0, 'Feb': 1, 'Mar': 2, 'Apr': 3, 'May': 4, 'Jun': 5, 
                'Jul': 6, 'Aug': 7, 'Sep': 8, 'Oct': 9, 'Nov': 10, 'Dec': 11 
            };
            if (monthStr in monthMap) {
                return new Date(year, monthMap[monthStr], day, hour, minute);
            }
        }

        // Parse numeric values (remove € and , symbols)
        const numValue = parseFloat(value.replace(/€/g, '').replace(/,/g, '.'));
        if (!isNaN(numValue)) {
            return numValue;
        }

        return value;
    }

    comparer(idx, asc) {
        return (a, b) => {
            const v1 = this.parseValue(this.getCellValue(asc ? a : b, idx));
            const v2 = this.parseValue(this.getCellValue(asc ? b : a, idx));
            
            // Handle Date objects
            if (v1 instanceof Date && v2 instanceof Date) {
                return v1.getTime() - v2.getTime();
            }
            
            // Handle numbers
            if (typeof v1 === 'number' && typeof v2 === 'number') {
                return v1 - v2;
            }
            
            // Handle strings
            return v1.toString().localeCompare(v2.toString());
        };
    }

    init() {
        document.addEventListener('DOMContentLoaded', () => {
            document.querySelectorAll('.sortable').forEach(th => {
                th.addEventListener('click', () => {
                    const table = th.closest('table');
                    const tbody = table.querySelector('tbody');
                    const thIndex = Array.from(th.parentNode.children).indexOf(th);
                    const currentIsAsc = th.classList.contains('sort-asc');

                    // Reset all sort indicators
                    document.querySelectorAll('.sortable').forEach(h => {
                        h.classList.remove('sort-asc', 'sort-desc');
                    });

                    // Set current sort direction
                    th.classList.toggle('sort-asc', !currentIsAsc);
                    th.classList.toggle('sort-desc', currentIsAsc);

                    // Sort table rows
                    Array.from(tbody.querySelectorAll('tr'))
                        .sort(this.comparer(thIndex, !currentIsAsc))
                        .forEach(tr => tbody.appendChild(tr));
                });
            });
        });
    }
}

// Initialize table sorter
new TableSorter();