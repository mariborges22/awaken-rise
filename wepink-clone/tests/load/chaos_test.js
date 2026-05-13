import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  iterations: 10,
  vus: 2,
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  // CENÁRIO 1: Tenant Inexistente (Deve falhar com 500 ou 404)
  const badOrderPayload = JSON.stringify({
    tenant_id: 'ghost-tenant-' + Math.random(),
    items: [{ product_id: 'p1', quantity: 1, price: 10.0 }],
  });

  const badRes = http.post(`${BASE_URL}/orders`, badOrderPayload, params);
  check(badRes, {
    'ghost tenant order fails': (r) => r.status >= 400,
  });

  // CENÁRIO 2: Teste de Idempotência (Pagar 2x o mesmo pedido)
  const okOrderPayload = JSON.stringify({
    tenant_id: 'k6-test-tenant',
    items: [{ product_id: 'p1', quantity: 1, price: 10.0 }],
  });

  const orderRes = http.post(`${BASE_URL}/orders`, okOrderPayload, params);
  const orderId = orderRes.json().id;
  const idempotencyKey = `chaos-key-${orderId}`;

  const payParams = {
    headers: {
      'Content-Type': 'application/json',
      'X-Idempotency-Key': idempotencyKey,
    },
  };

  // Primeira tentativa
  const pay1 = http.post(`${BASE_URL}/payments/${orderId}`, JSON.stringify({}), payParams);
  
  // Segunda tentativa imediata (concorrência/duplicidade)
  const pay2 = http.post(`${BASE_URL}/payments/${orderId}`, JSON.stringify({}), payParams);

  check(pay1, { 'first pay ok': (r) => r.status === 200 });
  check(pay2, { 
    'second pay handled (idempotent)': (r) => r.status === 200 || r.status === 409 
  });

  sleep(1);
}
