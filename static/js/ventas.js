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

let fpAddSale = null;
let fpEditSale = null;

// Configuración de Flatpickr
const flatpickrConfig = {
    enableTime: true,
    dateFormat: "Y-m-d\\TH:i",
    altInput: true,
    altFormat: "d.m.Y H:i",
    time_24hr: true,
    locale: "de"
};

document.addEventListener('DOMContentLoaded', function() {
    // Inicializar Flatpickr si existen los elementos
    const addInput = document.getElementById('add_sale_date');
    if (addInput) {
        fpAddSale = flatpickr(addInput, flatpickrConfig);
    }
    
    const editInput = document.getElementById('edit_sale_date');
    if (editInput) {
        fpEditSale = flatpickr(editInput, flatpickrConfig);
    }

    // Event delegation for edit buttons
    document.addEventListener('click', function(event) {
        const editBtn = event.target.closest('.edit-sale-btn');
        if (editBtn) {
            const id = editBtn.dataset.saleId;
            const ticker = editBtn.dataset.ticker;
            const tickerId = editBtn.dataset.tickerId;
            const saleDate = editBtn.dataset.saleDate;
            const shares = editBtn.dataset.shares;
            const salePrice = editBtn.dataset.salePrice;
            const operationCost = editBtn.dataset.operationCost;
            const withheldTax = editBtn.dataset.withheldTax;
            
            openEditModal(id, ticker, tickerId, saleDate, shares, salePrice, operationCost, withheldTax);
        }
    });
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

    // Update the edit button data attributes with new values
    const editButton = row.querySelector('.edit-sale-btn');
    if (editButton) {
        editButton.dataset.saleId = data.id;
        editButton.dataset.ticker = data.ticker;
        editButton.dataset.tickerId = data.ticker_id;
        editButton.dataset.saleDate = data.sale_date;
        editButton.dataset.shares = data.shares;
        editButton.dataset.salePrice = data.sale_price;
        editButton.dataset.operationCost = data.operation_cost;
        editButton.dataset.withheldTax = data.withheld_tax;
    }
}

/**
 * Opens the edit sale modal and populates it with the sale data
 */
function openEditModal(id, ticker, tickerId, saleDate, shares, salePrice, operationCost, withheldTax) {
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

            const isoDate = `${year}-${month}-${day}T${hour}:${minute}`;
            if (fpEditSale) {
                fpEditSale.setDate(isoDate);
            } else {
                dateInput.value = isoDate;
            }
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

/**
 * Opens the add sale modal
 */
function openAddSaleModal() {
    const modal = document.getElementById('addSaleModal');

    // Reset form
    const form = document.getElementById('addSaleForm');
    form.reset();

    // Set current date and time as default
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');

    const isoDate = `${year}-${month}-${day}T${hours}:${minutes}`;
    
    if (fpAddSale) {
        fpAddSale.setDate(isoDate);
    } else {
        document.getElementById('add_sale_date').value = isoDate;
    }

    // Show modal
    modal.classList.remove('hidden');
    modal.classList.add('flex');
}

/**
 * Closes the add sale modal
 */
function closeAddSaleModal() {
    const modal = document.getElementById('addSaleModal');
    modal.classList.add('hidden');
    modal.classList.remove('flex');
}

// Close add modal when clicking outside of it
document.addEventListener('click', function (event) {
    const addModal = document.getElementById('addSaleModal');
    if (addModal && event.target === addModal) {
        closeAddSaleModal();
    }
});
