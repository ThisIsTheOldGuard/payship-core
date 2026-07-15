import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '20s', target: 20 },
        { duration: '40s', target: 30 },
        { duration: '10s', target: 10 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'],
        http_req_failed: ['rate<0.05'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'

export default function () {
    const health = http.get(`${BASE_URL}/health`);
    check(health, {'health 200': (r) => r.status === 200});

    const payload = JSON.stringify({
        customer_name: `User_VU${__VU}_IT${__ITER}`,
        amount: Math.floor(Math.random() * 500) + 10,
    });

    const create = http.post(`${BASE_URL}/order`, payload, {
        headers: { 'Content-Type': 'application/json' },
    });
    check(create, { 'create 201': (r) => r.status === 201 });

    const orderID = create.json().id;
    //console.log(`Создан заказ с ID: ${orderID}`);

    const valid = http.post(`${BASE_URL}/order/${orderID}/transitions`,
        JSON.stringify({status: 'processing'}),
        { headers: { 'Content-Type': 'application/json' } });
    check(valid, { 'OK valid transition 204': (r) => r.status === 204 });

    // if (valid.status !== 204) {
    //     console.log(`Ждал 204, получил ${valid.status}. Ответ: ${valid.body}`);
    //     console.log(`Отправленный JSON ${transit}`);
    // }

    const invalid = http.post(`${BASE_URL}/order/${orderID}/transitions`,
        JSON.stringify({status: 'pending'}),
        {
            headers: { 'Content-Type': 'application/json' },
            responseCallback: http.expectedStatuses(400)
        }
    );
    check(invalid, { 'BAD invalid transition 400': (r) => r.status === 400 });

    const oneOrder = http.get(`${BASE_URL}/order/${orderID}`);
    check(oneOrder, {'get order 200': (r) => r.status === 200});

    const listOrders = http.get(`${BASE_URL}/orders?page=1&limit=10`);
    check(listOrders, {'list 200': (r) => r.status === 200});

    sleep(1);
}