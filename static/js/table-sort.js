/**
 * Generic table sorting functionality
 * Provides sorting capabilities for tables with sortable columns
 */
console.log('[TableSorter] Script table-sort.js cargado e iniciando...');
class TableSorter {
    constructor() {
        console.log('[TableSorter] Instanciando TableSorter...');
        this.tables = [];
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

        const numValue = parseFloat(value.replace(/€/g, '').replace(/,/g, '.').replace(/%/g, ''));
        if (!isNaN(numValue)) {
            return numValue;
        }

        return value;
    }

    comparer(idx, asc) {
        return (a, b) => {
            const v1 = this.parseValue(this.getCellValue(asc ? a : b, idx));
            const v2 = this.parseValue(this.getCellValue(asc ? b : a, idx));

            if (v1 instanceof Date && v2 instanceof Date) {
                return v1.getTime() - v2.getTime();
            }
            if (typeof v1 === 'number' && typeof v2 === 'number') {
                return v1 - v2;
            }
            return v1.toString().localeCompare(v2.toString());
        };
    }

    sortTable(table, index, asc) {
        const tbody = table.querySelector('tbody');
        if (!tbody) return;

        const headers = table.querySelectorAll('.sortable');
        const th = Array.from(table.querySelectorAll('th')).find(el =>
            Array.from(el.parentNode.children).indexOf(el) === index
        );

        headers.forEach(h => h.classList.remove('sort-asc', 'sort-desc'));

        if (th && th.classList.contains('sortable')) {
            th.classList.toggle('sort-asc', asc);
            th.classList.toggle('sort-desc', !asc);
        }

        Array.from(tbody.querySelectorAll('tr'))
            .sort(this.comparer(index, asc))
            .forEach(tr => tbody.appendChild(tr));

        console.log(`[TableSorter] Tabla "${table.id}" actualizada/renderizada. Estado de ordenamiento:`, { index, asc });
    }

    restoreSortForTable(table) {
        if (!table || !table.id) return;
        const storageKey = `sort_${window.location.pathname}_${table.id}`;
        const raw = localStorage.getItem(storageKey);
        console.log(`[TableSorter] restoreSortForTable: key="${storageKey}" raw="${raw}"`);
        const savedSort = JSON.parse(raw);
        if (savedSort !== null) {
            this.sortTable(table, savedSort.index, savedSort.asc);
        }
    }

    restoreAll() {
        console.log(`[TableSorter] restoreAll: ${this.tables.length} tabla(s) registradas`);
        this.tables.forEach(({ table, storageKey }) => {
            const raw = localStorage.getItem(storageKey);
            console.log(`[TableSorter] restoreAll: key="${storageKey}" valor="${raw}"`);
            const savedSort = JSON.parse(raw);
            if (savedSort !== null) {
                this.sortTable(table, savedSort.index, savedSort.asc);
            }
        });
    }

    init() {
        document.addEventListener('DOMContentLoaded', () => {
            const path = window.location.pathname;
            console.log(`[TableSorter] DOMContentLoaded: path="${path}"`);

            document.querySelectorAll('table').forEach((table, tableIndex) => {
                const tableId = table.id || `table-${tableIndex}`;
                const storageKey = `sort_${path}_${tableId}`;

                const sortables = table.querySelectorAll('.sortable');
                if (sortables.length === 0) return;

                console.log(`[TableSorter] Tabla registrada: id="${tableId}" storageKey="${storageKey}"`);
                this.tables.push({ table, storageKey });

                sortables.forEach((th) => {
                    th.addEventListener('click', () => {
                        const thRealIndex = Array.from(th.parentNode.children).indexOf(th);
                        const currentIsAsc = th.classList.contains('sort-asc');
                        const newAsc = !currentIsAsc;

                        this.sortTable(table, thRealIndex, newAsc);

                        const data = JSON.stringify({ index: thRealIndex, asc: newAsc });
                        localStorage.setItem(storageKey, data);
                        console.log(`[TableSorter] Sort guardado: key="${storageKey}" valor="${data}"`);
                    });
                });
            });
        });

        window.addEventListener('load', () => {
            console.log(`[TableSorter] window.load: llamando restoreAll()`);
            this.restoreAll();
        });
    }
}

const tableSorter = new TableSorter();