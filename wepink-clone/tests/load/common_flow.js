import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 5,
  duration: '1m',
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const MP_TOKEN = __ENV.MP_TOKEN || 'TEST-4171246039575815-050410-6c9c614c227b60098f98642735d67807-172551460';

export function setup() {
  // Registrar o tenant de teste antes de começar o load
  const payload = JSON.stringify({
    tenant_id: 'k6-test-tenant',
    mp_access_token: MP_TOKEN,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const res = http.post(`${BASE_URL}/tenants`, payload, params);
  console.log(`Setup: Tenant registration status: ${res.status}`);
}

export default function () {
  const correlationId = `k6-${Math.random().toString(36).substring(7)}`;
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Correlation-ID': correlationId,
    },
  };

  // 1. Criar Pedido
  const orderPayload = JSON.stringify({
    tenant_id: 'k6-test-tenant',
    items: [
      { product_id: 'prod-123', quantity: 2, price: 49.90 }
    ],
  });

  const orderRes = http.post(`${BASE_URL}/orders`, orderPayload, params);
  check(orderRes, {
    'order created': (r) => r.status === 201,
  });

  if (orderRes.status !== 201) return;

  const orderId = orderRes.json().id;

  // 2. Processar Pagamento (Fluxo Feliz)
  sleep(1);
  const paymentPayload = JSON.stringify({
    payment_method: 'pix',
    buyer_email: 'test@example.com'
  });

  const payRes = http.post(`${BASE_URL}/payments/${orderId}`, paymentPayload, params);
  check(payRes, {
    'payment request accepted': (r) => r.status === 200,
  });

  // 3. Verificar Status
  sleep(2);
  const getRes = http.get(`${BASE_URL}/orders/${orderId}`, params);
  check(getRes, {
    'order confirmed/pending': (r) => r.json().status === 'CONFIRMED' || r.json().status === 'PENDING',
  });

  sleep(1);
}
