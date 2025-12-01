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
    const saleDateStr = document.getElementById('edit_sale_date').value;
    const shares = parseFloat(document.getElementById('edit_shares').value);
    const salePrice = parseFloat(document.getElementById('edit_sale_price').value);
    const operationCost = parseFloat(document.getElementById('edit_operation_cost').value) || 0;
    const withheldTax = parseFloat(document.getElementById('edit_withheld_tax').value) || 0;

    // Convert date from DD/MM/YYYY to YYYY-MM-DD
    let formattedDate = saleDateStr;
    const dateParts = saleDateStr.split('/');
    if (dateParts.length === 3) {
        formattedDate = `${dateParts[2]}-${dateParts[1]}-${dateParts[0]}`;
    }

    const data = {
        ticker_id: tickerId,
        sale_date: formattedDate,
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

    // Convert date format from "02 Jan 2006" to "DD/MM/YYYY"
    const dateInput = document.getElementById('edit_sale_date');
    if (saleDate) {
        // Parse the date string and convert to DD/MM/YYYY format
        const parsedDate = new Date(saleDate);
        if (!isNaN(parsedDate.getTime())) {
            const year = parsedDate.getFullYear();
            const month = String(parsedDate.getMonth() + 1).padStart(2, '0');
            const day = String(parsedDate.getDate()).padStart(2, '0');
            dateInput.value = `${day}/${month}/${year}`;
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

    const calcModal = document.getElementById('calculationModal');
    if (calcModal && event.target === calcModal) {
        closeCalculationModal();
    }
});

/**
 * Opens the calculation detail modal and fetches data
 */
function openCalculationModal(id) {
    // Show loading state or clear previous data
    document.getElementById('calc_purchases_list').innerHTML = '<tr><td colspan="4" class="px-4 py-2 text-center">Cargando...</td></tr>';

    // Show modal
    const modal = document.getElementById('calculationModal');
    modal.classList.remove('hidden');
    modal.classList.add('flex');

    // Fetch data
    fetch(`/sale-calculation/${id}`)
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                alert('Error: ' + data.error);
                closeCalculationModal();
                return;
            }

            // Populate Sale Info
            document.getElementById('calc_ticker').textContent = data.ticker;
            document.getElementById('calc_sale_date').textContent = data.sale_date;
            document.getElementById('calc_shares').textContent = data.shares.toFixed(6);
            document.getElementById('calc_sale_price').textContent = data.sale_price.toFixed(4) + '€';

            // Populate Purchases List
            const tbody = document.getElementById('calc_purchases_list');
            tbody.innerHTML = '';

            if (data.purchases && data.purchases.length > 0) {
                data.purchases.forEach(p => {
                    const tr = document.createElement('tr');
                    tr.className = 'bg-white border-b dark:bg-gray-800 dark:border-gray-700';
                    tr.innerHTML = `
                        <td class="px-4 py-2">${p.date}</td>
                        <td class="px-4 py-2">${p.shares.toFixed(6)}</td>
                        <td class="px-4 py-2">${p.price.toFixed(4)}€</td>
                        <td class="px-4 py-2">${p.total.toFixed(2)}€</td>
                    `;
                    tbody.appendChild(tr);
                });
            } else {
                tbody.innerHTML = '<tr><td colspan="4" class="px-4 py-2 text-center">No hay compras registradas antes de esta fecha</td></tr>';
            }

            // Populate WAC Calculation
            document.getElementById('calc_total_capital').textContent = data.total_capital.toFixed(4) + '€';
            document.getElementById('calc_total_shares').textContent = data.total_shares.toFixed(6);
            document.getElementById('calc_wac').textContent = data.wac.toFixed(4) + '€';

            // Populate Profit Calculation
            document.getElementById('calc_profit_sale_price').textContent = data.sale_price.toFixed(4) + '€';
            document.getElementById('calc_profit_wac').textContent = data.wac.toFixed(4) + '€';

            const diffPerShare = data.sale_price - data.wac;
            const diffElement = document.getElementById('calc_diff_per_share');
            diffElement.textContent = diffPerShare.toFixed(4) + '€';
            diffElement.className = diffPerShare >= 0 ? 'font-medium text-green-600 dark:text-green-400' : 'font-medium text-red-600 dark:text-red-400';

            document.getElementById('calc_profit_shares').textContent = data.shares.toFixed(6);

            const profitElement = document.getElementById('calc_total_profit');
            profitElement.textContent = data.profit.toFixed(4) + '€';
            profitElement.className = data.profit >= 0 ? 'font-bold text-green-600 dark:text-green-400' : 'font-bold text-red-600 dark:text-red-400';
        })
        .catch(error => {
            console.error('Error fetching calculation details:', error);
            alert('Error al cargar los detalles del cálculo');
            closeCalculationModal();
        });
}

/**
 * Closes the calculation detail modal
 */
function closeCalculationModal() {
    const modal = document.getElementById('calculationModal');
    modal.classList.add('hidden');
    modal.classList.remove('flex');
}