import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Métrica customizada para trackear taxa de erro
export let errorRate = new Rate('errors');

// Configuração de Stress (Ramp-up, Sustentação e Ramp-down)
export let options = {
    stages: [
        { duration: '15s', target: 50 },  // Ramp-up rápido para 50 usuários
        { duration: '30s', target: 200 }, // Acelera para 200 VUs simultâneos (Pico de Black Friday)
        { duration: '1m', target: 200 },  // Mantém 200 VUs por 1 minuto
        { duration: '15s', target: 0 },   // Desce para 0 (Ramp-down)
    ],
    thresholds: {
        // Tolerância de Performance
        http_req_duration: ['p(95)<500'], // 95% das requisições devem demorar menos de 500ms
        errors: ['rate<0.01'],            // Taxa de erro deve ser menor que 1%
    },
};

const BASE_URL = 'http://localhost:8080';

// Headers comuns (simulando a Vitrine)
const HEADERS = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': 'tenant-loadtest-01', // O inquilino chumbado que configuramos
};

export default function () {
    // 1. Cenário de Navegação (Listagem de Produtos - Cache Redis)
    let resProducts = http.get(`${BASE_URL}/products`, { headers: HEADERS });
    
    let productsCheck = check(resProducts, {
        'GET Products is 200': (r) => r.status === 200,
        'has products': (r) => r.json('data') !== null,
    });
    errorRate.add(!productsCheck);

    // Pausa simulando o usuário lendo a vitrine
    sleep(Math.random() * 2);

    // 2. Cenário de Compra (Se a listagem falhou, nem tenta comprar)
    if (productsCheck) {
        let products = resProducts.json('data');
        if (products && products.length > 0) {
            // Pega um produto aleatório
            let randomProduct = products[Math.floor(Math.random() * products.length)];
            
            let orderPayload = JSON.stringify({
                items: [
                    {
                        product_id: randomProduct.id,
                        quantity: 1
                    }
                ]
            });

            // Cria o pedido
            let resOrder = http.post(`${BASE_URL}/orders`, orderPayload, { headers: HEADERS });
            
            let orderCheck = check(resOrder, {
                'POST Order is 201': (r) => r.status === 201,
            });
            errorRate.add(!orderCheck);
            
            // Simula o tempo do usuário preenchendo o e-mail do PIX (apenas 30% das pessoas vão pro pagamento)
            if (orderCheck && Math.random() < 0.3) {
                sleep(1);
                let orderId = resOrder.json('data.id');
                
                let paymentPayload = JSON.stringify({
                    payment_method: 'pix',
                    buyer_email: `loadtest_${__VU}_${__ITER}@teste.com`
                });

                // Tenta gerar o PIX chamando o Gateway do Mercado Pago
                let resPayment = http.post(`${BASE_URL}/payments/${orderId}`, paymentPayload, { headers: HEADERS });
                
                let paymentCheck = check(resPayment, {
                    'POST Payment is 200': (r) => r.status === 200,
                });
                errorRate.add(!paymentCheck);
            }
        }
    }

    // Tempo de reflexão final do usuário antes de dar refresh
    sleep(Math.random() * 1);
}
