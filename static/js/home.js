/**
 * Home page specific functionality
 * Handles summary table auto-sorting by utility column
 */
document.addEventListener('DOMContentLoaded', function() {
    const summaryTable = document.getElementById('summaryTable');
    if (summaryTable) {
        // Auto-sort by utility column (6th column) in descending order
        const utilityHeader = summaryTable.querySelector('thead th:nth-child(6)');
        if (utilityHeader) {
            // Click twice to get descending order (first click = asc, second = desc)
            utilityHeader.click();
            utilityHeader.click();
        }
    }
});