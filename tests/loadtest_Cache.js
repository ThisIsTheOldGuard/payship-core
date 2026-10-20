import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
    stages: [
        { duration: '10s', target: 10 },
        { duration: '30s', target: 50 },
        { duration: '10s', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<100'], // Для кэшированных запросов ожидаем < 100мс
        http_req_failed: ['rate<0.01'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// прогрев кэша
export function setup() {
    const orderCount = 20;
    const orderIds = [];

    console.log(`[SETUP] Создаём ${orderCount} заказов для прогрева кэша...`);

    for (let i = 0; i < orderCount; i++) {
        const payload = JSON.stringify({
            customer_name: `CachedUser_${i}`,
            amount: 100 + i,
        });

        const res = http.post(`${BASE_URL}/order`, payload, {
            headers: { 'Content-Type': 'application/json' },
        });

        if (res.status === 201) {
            const id = res.json().id;
            orderIds.push(id);

            http.post(`${BASE_URL}/order/${id}/transitions`,
                JSON.stringify({ status: 'processing' }),
                { headers: { 'Content-Type': 'application/json' } }
            );
        }
    }

    // кжш списка
    http.get(`${BASE_URL}/orders?page=1&limit=10`);

    console.log(`Создано заказов: ${orderIds.length}`);
    return { orderIds };
}

export default function (data) {

    if (!data || !data.orderIds || data.orderIds.length === 0) {
        console.error('Нет сведений по созданным заказам');
        return;
    }

    const randomIndex = randomIntBetween(0, data.orderIds.length - 1);
    const orderId = data.orderIds[randomIndex];

    const getOrderRes = http.get(`${BASE_URL}/order/${orderId}`);
    check(getOrderRes, {
        'get order 200': (r) => r.status === 200,
        'get order fast': (r) => r.timings.duration < 50,
    });

    if (Math.random() < 0.1) {
        const listRes = http.get(`${BASE_URL}/orders?page=1&limit=10`);
        check(listRes, { 'list orders 200': (r) => r.status === 200 });
    }

    if (Math.random() < 0.1) {
        const newPayload = JSON.stringify({
            customer_name: `NewUser_VU${__VU}`,
            amount: 500,
        });

        const createRes = http.post(`${BASE_URL}/order`, newPayload, {
            headers: { 'Content-Type': 'application/json' },
        });
        check(createRes, { 'create order 201': (r) => r.status === 201 });
    }

    sleep(0.1);
}