/**
 * Ventas (Sales) page specific functionality
 * Handles sales table auto-sorting by date column
 */
document.addEventListener('DOMContentLoaded', function () {
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

/**
 * Submit edit form via AJAX
 */
function submitEditForm() {
    const form = document.getElementById('editSaleForm');
    
    // Validate form
    if (!form.checkValidity()) {
        form.reportValidity();
        return;
    }
    
    const saleId = document.getElementById('edit_sale_id').value;

    // Get form values
    const tickerId = parseInt(document.getElementById('edit_ticker_id').value);
    const saleDateStr = document.getElementById('edit_sale_date').value; // Format: YYYY-MM-DDTHH:MM
    const shares = parseFloat(document.getElementById('edit_shares').value);
    const salePrice = parseFloat(document.getElementById('edit_sale_price').value);
    const operationCost = parseFloat(document.getElementById('edit_operation_cost').value) || 0;
    const withheldTax = parseFloat(document.getElementById('edit_withheld_tax').value) || 0;

    const data = {
        ticker_id: tickerId,
        sale_date: saleDateStr, // Already in YYYY-MM-DDTHH:MM format from datetime-local input
        shares: shares,
        sale_price: salePrice,
        operation_cost: operationCost,
        withheld_tax: withheldTax
    };

    fetch(`/api/sale/${saleId}`, {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
    })
        .then(response => response.json())
        .then(result => {
            if (result.error) {
                alert('Error: ' + result.error);
                return;
            }

            // Update the table row
            updateTableRow(result);

            // Close the modal
            closeEditModal();
        })
        .catch(error => {
            console.error('Error updating sale:', error);
            alert('Error al actualizar la venta');
        });
}

/**
 * Update table row with new data
 */
function updateTableRow(data) {
    const row = document.querySelector(`tr[data-id="${data.id}"]`);
    if (!row) return;

    // Update ticker
    const tickerCell = row.querySelector('[data-field="ticker"]');
    if (tickerCell) {
        tickerCell.textContent = data.ticker;
        tickerCell.setAttribute('data-ticker-id', data.ticker_id);
    }

    // Update sale date
    const saleDateCell = row.querySelector('[data-field="sale_date"]');
    if (saleDateCell) {
        saleDateCell.textContent = data.sale_date;
    }

    // Update shares
    const sharesCell = row.querySelector('[data-field="shares"]');
    if (sharesCell) {
        sharesCell.textContent = data.shares.toFixed(6);
    }

    // Update sale price
    const salePriceCell = row.querySelector('[data-field="sale_price"]');
    if (salePriceCell) {
        salePriceCell.textContent = data.sale_price.toFixed(4) + '€';
    }

    // Update operation cost
    const operationCostCell = row.querySelector('[data-field="operation_cost"]');
    if (operationCostCell) {
        operationCostCell.textContent = data.operation_cost.toFixed(2) + '€';
    }

    // Update withheld tax
    const withheldTaxCell = row.querySelector('[data-field="withheld_tax"]');
    if (withheldTaxCell) {
        withheldTaxCell.textContent = data.withheld_tax.toFixed(2) + '€';
    }

    // Update total sale value
    const totalSaleValueCell = row.querySelector('[data-field="total_sale_value"]');
    if (totalSaleValueCell) {
        totalSaleValueCell.textContent = data.total_sale_value.toFixed(2) + '€';
    }

    // Update performance with color
    const performanceCell = row.querySelector('[data-field="performance"]');
    if (performanceCell) {
        performanceCell.textContent = data.performance.toFixed(2) + '%';
        performanceCell.className = 'px-6 py-4 font-medium ' + (data.performance >= 0
            ? 'text-green-600 dark:text-green-400'
            : 'text-red-600 dark:text-red-400');
    }

    // Update profit with color
    const profitCell = row.querySelector('[data-field="profit"]');
    if (profitCell) {
        profitCell.textContent = data.profit.toFixed(2) + '€';
        profitCell.className = 'px-6 py-4 font-medium ' + (data.profit >= 0
            ? 'text-green-600 dark:text-green-400'
            : 'text-red-600 dark:text-red-400');
    }

    // Update the edit button onclick with new values
    const editButton = row.querySelector('button[onclick^="openEditModal"]');
    if (editButton) {
        editButton.setAttribute('onclick', `openEditModal(${data.id}, '${data.ticker}', ${data.ticker_id}, '${data.sale_date}', ${data.shares}, ${data.sale_price}, ${data.operation_cost}, ${data.withheld_tax})`);
    }
}

/**
 * Opens the edit sale modal and populates it with the sale data
 */
function openEditModal(id, ticker, tickerId, saleDate, shares, salePrice, operationCost, withheldTax) {
    // Set form action
    const form = document.getElementById('editSaleForm');
    form.action = `/update-sale/${id}`;

    // Populate form fields
    document.getElementById('edit_sale_id').value = id;
    document.getElementById('edit_ticker_id').value = tickerId;

    // Convert date format from "02 Jan 2006 15:04" to "YYYY-MM-DDTHH:MM" for datetime-local input
    const dateInput = document.getElementById('edit_sale_date');
    if (saleDate) {
        // Parse format: "02 Jan 2006 15:04"
        const dateMatch = saleDate.match(/^(\d{2}) (\w{3}) (\d{4})(?: (\d{2}):(\d{2}))?$/);
        if (dateMatch) {
            const day = dateMatch[1];
            const monthStr = dateMatch[2];
            const year = dateMatch[3];
            const hour = dateMatch[4] || '00';
            const minute = dateMatch[5] || '00';
            
            const monthMap = { 
                'Jan': '01', 'Feb': '02', 'Mar': '03', 'Apr': '04', 'May': '05', 'Jun': '06', 
                'Jul': '07', 'Aug': '08', 'Sep': '09', 'Oct': '10', 'Nov': '11', 'Dec': '12' 
            };
            const month = monthMap[monthStr] || '01';
            
            dateInput.value = `${year}-${month}-${day}T${hour}:${minute}`;
        }
    }

    document.getElementById('edit_shares').value = shares;
    document.getElementById('edit_sale_price').value = salePrice;
    document.getElementById('edit_operation_cost').value = operationCost;
    document.getElementById('edit_withheld_tax').value = withheldTax;

    // Show modal
    const modal = document.getElementById('editSaleModal');
    modal.classList.remove('hidden');
    modal.classList.add('flex');
}

/**
 * Closes the edit sale modal
 */
function closeEditModal() {
    const modal = document.getElementById('editSaleModal');
    modal.classList.add('hidden');
    modal.classList.remove('flex');
}

// Close modal when clicking outside of it
document.addEventListener('click', function (event) {
    const editModal = document.getElementById('editSaleModal');
    if (editModal && event.target === editModal) {
        closeEditModal();
    }
});

