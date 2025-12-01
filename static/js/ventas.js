/**
 * Ventas (Sales) page specific functionality
 * Handles sales table auto-sorting by date column
 */
document.addEventListener('DOMContentLoaded', function() {
    const salesTable = document.getElementById('salesTable');
    if (salesTable) {
        // Auto-sort by sale date column (2nd column) in descending order
        const saleDateHeader = salesTable.querySelector('thead th:nth-child(2)');
        if (saleDateHeader) {
            // Click twice to get descending order (first click = asc, second = desc)
            saleDateHeader.click();
            saleDateHeader.click();
        }
    }
});