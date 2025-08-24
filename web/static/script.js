// Функция для получения заказа по ID
function getOrder() {
    // Получаем ID заказа из поля ввода и убираем лишние пробелы
    const orderId = document.getElementById('orderId').value.trim();
    if (!orderId) {
        showError('Please enter an Order ID');
        return;
    }

    console.log('Fetching order:', orderId);
    hideError();
    showLoading();
    hideOrderInfo();

    // Отправляем запрос к API для получения заказа
    fetch(`/order/${orderId}`)
        .then(response => {
            console.log('Order response status:', response.status);
            if (!response.ok) {
                if (response.status === 404) {
                    throw new Error('Order not found');
                }
                throw new Error('Failed to fetch order');
            }
            return response.json();
        })
        .then(order => {
            console.log('Order received:', order);
            displayOrder(order); // Отображаем заказ на странице
            hideLoading();
            // Обновляем статистику после успешного запроса
            console.log('Refreshing stats after order fetch...');
            refreshStats();
        })
        .catch(error => {
            console.error('Error fetching order:', error);
            hideLoading();
            showError(error.message);
        });
}

// Функция для отображения информации о заказе на странице
function displayOrder(order) {
    // Отображаем основную информацию о заказе
    document.getElementById('orderBasic').innerHTML = `
        <p><strong>Order UID:</strong> ${order.order_uid}</p>
        <p><strong>Track Number:</strong> ${order.track_number}</p>
        <p><strong>Entry:</strong> ${order.entry}</p>
        <p><strong>Locale:</strong> ${order.locale}</p>
        <p><strong>Customer ID:</strong> ${order.customer_id}</p>
        <p><strong>Delivery Service:</strong> ${order.delivery_service}</p>
        <p><strong>Shard Key:</strong> ${order.shardkey}</p>
        <p><strong>SM ID:</strong> ${order.sm_id}</p>
        <p><strong>Date Created:</strong> ${new Date(order.date_created).toLocaleString()}</p>
        <p><strong>OOF Shard:</strong> ${order.oof_shard}</p>
    `;

    // Отображаем информацию о доставке
    document.getElementById('deliveryInfo').innerHTML = `
        <p><strong>Name:</strong> ${order.delivery.name}</p>
        <p><strong>Phone:</strong> ${order.delivery.phone}</p>
        <p><strong>Email:</strong> ${order.delivery.email}</p>
        <p><strong>Address:</strong> ${order.delivery.address}, ${order.delivery.city}</p>
        <p><strong>Region:</strong> ${order.delivery.region}</p>
        <p><strong>ZIP:</strong> ${order.delivery.zip}</p>
    `;

    // Отображаем информацию о платеже (конвертируем timestamp в дату)
    const paymentDate = new Date(order.payment.payment_dt * 1000);
    document.getElementById('paymentInfo').innerHTML = `
        <p><strong>Transaction:</strong> ${order.payment.transaction}</p>
        <p><strong>Request ID:</strong> ${order.payment.request_id || 'N/A'}</p>
        <p><strong>Currency:</strong> ${order.payment.currency}</p>
        <p><strong>Provider:</strong> ${order.payment.provider}</p>
        <p><strong>Amount:</strong> $${(order.payment.amount / 100).toFixed(2)}</p>
        <p><strong>Payment Date:</strong> ${paymentDate.toLocaleString()}</p>
        <p><strong>Bank:</strong> ${order.payment.bank}</p>
        <p><strong>Delivery Cost:</strong> $${(order.payment.delivery_cost / 100).toFixed(2)}</p>
        <p><strong>Goods Total:</strong> $${(order.payment.goods_total / 100).toFixed(2)}</p>
        <p><strong>Custom Fee:</strong> $${(order.payment.custom_fee / 100).toFixed(2)}</p>
    `;

    // Отображаем список товаров в заказе
    const itemsHtml = order.items && order.items.length > 0 
        ? order.items.map(item => `
            <div class="item">
                <p><strong>Name:</strong> ${item.name}</p>
                <p><strong>Brand:</strong> ${item.brand}</p>
                <p><strong>Price:</strong> $${(item.price / 100).toFixed(2)}</p>
                <p><strong>Sale:</strong> ${item.sale}%</p>
                <p><strong>Total Price:</strong> $${(item.total_price / 100).toFixed(2)}</p>
                <p><strong>Size:</strong> ${item.size}</p>
                <p><strong>Status:</strong> ${item.status}</p>
                <p><strong>Track Number:</strong> ${item.track_number}</p>
                <p><strong>CHRT ID:</strong> ${item.chrt_id}</p>
                <p><strong>NM ID:</strong> ${item.nm_id}</p>
            </div>
        `).join('')
        : '<p>No items found</p>';

    document.getElementById('itemsList').innerHTML = itemsHtml;
    document.getElementById('orderInfo').style.display = 'block';
}

// Функция для отображения ошибки на странице
function showError(message) {
    const errorDiv = document.getElementById('error');
    errorDiv.textContent = message;
    errorDiv.style.display = 'block';
}

// Функция для скрытия сообщения об ошибке
function hideError() {
    document.getElementById('error').style.display = 'none';
}

// Функция для отображения индикатора загрузки
function showLoading() {
    document.getElementById('loading').style.display = 'block';
}

// Функция для скрытия индикатора загрузки
function hideLoading() {
    document.getElementById('loading').style.display = 'none';
}

// Функция для скрытия информации о заказе
function hideOrderInfo() {
    document.getElementById('orderInfo').style.display = 'none';
}

// Функция для обновления статистики сервера
function refreshStats() {
    console.log('Refreshing stats...');
    fetch('/stats')
        .then(response => {
            console.log('Stats response status:', response.status);
            return response.json();
        })
        .then(stats => {
            console.log('Received stats:', stats);
            // Обновляем размер кэша
            document.getElementById('cacheSize').textContent = stats.cache_size;
            
            // Обновляем время последнего запроса
            if (stats.last_request_time) {
                const lastRequestTime = new Date(stats.last_request_time);
                document.getElementById('lastRequestTime').textContent = lastRequestTime.toLocaleString();
            } else {
                document.getElementById('lastRequestTime').textContent = 'Never';
            }
            
            // Обновляем длительность последнего запроса
            document.getElementById('lastRequestDuration').textContent = stats.last_request_duration + 'ms';
            console.log('Stats updated successfully');
        })
        .catch(error => {
            console.error('Failed to fetch stats:', error);
        });
}

// Загружаем статистику при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    refreshStats();
});

// Обработчик нажатия Enter в поле ввода ID заказа
document.getElementById('orderId').addEventListener('keypress', function(e) {
    if (e.key === 'Enter') {
        getOrder();
    }
});