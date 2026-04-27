import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 }, // ramp-up to 20 users
    { duration: '1m', target: 20 },  // stay at 20 users
    { duration: '20s', target: 0 },  // ramp-down to 0 users
  ],
};

const BASE_URL = 'http://localhost:8080';

export default function () {
  // 1. Create Order
  const payload = JSON.stringify({
    items: [
      { product_id: 'p1', quantity: 1, price: 50.0 },
      { product_id: 'p2', quantity: 2, price: 100.0 },
    ],
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Correlation-ID': `k6-test-${__VU}-${__ITER}`,
    },
  };

  const createRes = http.post(`${BASE_URL}/orders`, payload, params);
  check(createRes, {
    'order created status is 201': (r) => r.status === 201,
    'order has id': (r) => JSON.parse(r.body).id !== undefined,
  });

  if (createRes.status !== 201) return;

  const orderId = JSON.parse(createRes.body).id;

  // 2. Process Payment
  sleep(1);
  const payRes = http.post(`${BASE_URL}/payments/${orderId}`, null, params);
  check(payRes, {
    'payment processed status is 200': (r) => r.status === 200,
  });

  // 3. Get Status
  sleep(1);
  const statusRes = http.get(`${BASE_URL}/orders/${orderId}`, params);
  check(statusRes, {
    'get order status is 200': (r) => r.status === 200,
    'order is confirmed or pending': (r) => {
        const status = JSON.parse(r.body).status;
        return status === 'CONFIRMED' || status === 'PENDING';
    },
  });

  sleep(2);
}
