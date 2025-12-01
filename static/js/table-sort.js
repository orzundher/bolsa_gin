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
        // Parse date format: DD MMM YYYY
        const dateMatch = value.match(/^(\d{2}) (\w{3}) (\d{4})$/);
        if (dateMatch) {
            const day = parseInt(dateMatch[1], 10);
            const monthStr = dateMatch[2];
            const year = parseInt(dateMatch[3], 10);
            const monthMap = { 
                'Jan': 0, 'Feb': 1, 'Mar': 2, 'Apr': 3, 'May': 4, 'Jun': 5, 
                'Jul': 6, 'Aug': 7, 'Sep': 8, 'Oct': 9, 'Nov': 10, 'Dec': 11 
            };
            if (monthStr in monthMap) {
                return new Date(year, monthMap[monthStr], day);
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
        return (a, b) => ((v1, v2) =>
            v1 !== '' && v2 !== '' && !isNaN(v1) && !isNaN(v2)
                ? v1 - v2
                : v1.toString().localeCompare(v2.toString())
        )(this.parseValue(this.getCellValue(asc ? a : b, idx)), 
          this.parseValue(this.getCellValue(asc ? b : a, idx)));
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