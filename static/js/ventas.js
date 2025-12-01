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