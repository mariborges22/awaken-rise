import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription, interval } from 'rxjs';
import { switchMap, takeWhile } from 'rxjs/operators';
import { ApiService } from '../../services/api.service';
import { TenantService } from '../../core/services/tenant.service';
import { Order, OrderItem, Product } from '../../models/api.models';

@Component({
  selector: 'app-order',
  templateUrl: './order.component.html',
  styleUrls: ['./order.component.css']
})
export class OrderComponent implements OnInit, OnDestroy {
  currentOrder: Order | null = null;
  errorMessage: string = '';
  loading: boolean = false;
  
  products: Product[] = [];
  cart: { [productId: string]: number } = {};

  buyerEmail: string = '';
  pixQRCodeBase64: string = '';
  pixCopyPaste: string = '';
  paymentConfirmed: boolean = false;

  private pollingSub: Subscription | null = null;

  constructor(
    private apiService: ApiService,
    private tenantService: TenantService,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    // Lê o :tenantId da URL (ex: /loja/loja-da-maria) e registra no TenantService.
    // O TenantInterceptor vai injetar automaticamente esse ID no header X-Tenant-ID
    // em todas as requisições HTTP subsequentes.
    const tenantId = this.route.snapshot.paramMap.get('tenantId');
    if (tenantId) {
      this.tenantService.setTenant(tenantId);
    }
    this.loadProducts();
  }

  ngOnDestroy() {
    this.stopPolling();
  }

  loadProducts() {
    this.loading = true;
    this.apiService.getProducts().subscribe({
      next: (res) => {
        this.products = res.data || [];
        this.loading = false;
      },
      error: () => {
        this.errorMessage = 'Não foi possível carregar os produtos. Tente novamente.';
        this.loading = false;
      }
    });
  }

  addToCart(product: Product) {
    if (!this.cart[product.id]) {
      this.cart[product.id] = 0;
    }
    if (this.cart[product.id] < product.stock) {
      this.cart[product.id]++;
    }
  }

  removeFromCart(productId: string) {
    if (this.cart[productId] > 0) {
      this.cart[productId]--;
    }
  }

  getCartTotal(): number {
    return this.getCartItems().reduce((total, item) => {
      const product = this.products.find(p => p.id === item.product_id);
      return total + (product ? product.price * item.quantity : 0);
    }, 0);
  }

  getCartItems(): OrderItem[] {
    return Object.keys(this.cart)
      .filter(id => this.cart[id] > 0)
      .map(id => ({
        product_id: id,
        quantity: this.cart[id]
      }));
  }

  createOrder() {
    const items = this.getCartItems();
    if (items.length === 0) {
      this.errorMessage = 'Seu carrinho está vazio.';
      return;
    }

    this.loading = true;
    this.errorMessage = '';
    this.apiService.createOrder(items).subscribe({
      next: (res) => {
        this.currentOrder = res.data!;
        this.cart = {};
        this.loading = false;
      },
      error: (err) => {
        this.errorMessage = err.error?.error || 'Falha ao criar pedido.';
        this.loading = false;
      }
    });
  }

  pay() {
    if (!this.currentOrder) return;
    if (!this.buyerEmail) {
      this.errorMessage = 'Por favor, informe seu e-mail para gerar o PIX.';
      return;
    }

    this.loading = true;
    this.errorMessage = '';
    
    this.apiService.processPayment(this.currentOrder.id, {
      payment_method: 'pix',
      buyer_email: this.buyerEmail
    }).subscribe({
      next: (res) => {
        this.loading = false;
        if (res.data?.gateway_response) {
          this.pixQRCodeBase64 = res.data.gateway_response.PixQRCodeBase64;
          this.pixCopyPaste = res.data.gateway_response.PixCopyPaste;
        }
        // Inicia o polling para detectar quando o PIX for pago
        this.startPolling();
      },
      error: (err) => {
        this.errorMessage = err.error?.error || 'Falha ao processar pagamento.';
        this.loading = false;
      }
    });
  }

  // Polling: verifica o status do pedido a cada 3 segundos até ser confirmado
  private startPolling() {
    if (!this.currentOrder) return;
    const orderId = this.currentOrder.id;
    
    this.pollingSub = interval(3000).pipe(
      switchMap(() => this.apiService.getOrder(orderId)),
      takeWhile(res => res.data?.status !== 'CONFIRMED', true)
    ).subscribe({
      next: (res) => {
        this.currentOrder = res.data!;
        if (res.data?.status === 'CONFIRMED') {
          this.paymentConfirmed = true;
          this.stopPolling();
        }
      }
    });
  }

  private stopPolling() {
    if (this.pollingSub) {
      this.pollingSub.unsubscribe();
      this.pollingSub = null;
    }
  }
}
