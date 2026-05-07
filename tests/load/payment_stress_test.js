import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
    stages: [
        { duration: '30s', target: 50 },  // Ramp-up: sobe para 50 usuários em 30s
        { duration: '1m', target: 50 },   // Mantém 50 usuários por 1 minuto
        { duration: '30s', target: 100 }, // Ramp-up: sobe para 100 usuários (estresse)
        { duration: '1m', target: 100 },  // Mantém os 100 usuários
        { duration: '30s', target: 0 },   // Ramp-down: volta a 0
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% das requisições devem ser < 500ms
        http_req_failed: ['rate<0.01'],   // Taxa de falha deve ser menor que 1%
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
// Simulação de um ID de pedido e token de lojista já existentes no banco de staging
const ORDER_ID = __ENV.ORDER_ID || 'test-order-id'; 

export default function () {
    const payload = JSON.stringify({
        payment_method: 'pix',
        buyer_email: `loadtest-${uuidv4()}@example.com`
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            // Opcional: Adicionar Headers de Auth se sua API exigir JWT
        },
    };

    // Simulando o request de pagamento
    const res = http.post(`${BASE_URL}/api/orders/${ORDER_ID}/pay`, payload, params);

    check(res, {
        'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
        'has transaction_id': (r) => r.json('data.transaction_id') !== undefined,
    });

    sleep(1); // Simula o tempo do usuário pensando entre requisições
}
