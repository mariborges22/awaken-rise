import { Component, OnInit } from '@angular/core';
import { ApiService } from '../../services/api.service';
import { Order, OrderItem, Product } from '../../models/api.models';

@Component({
  selector: 'app-order',
  templateUrl: './order.component.html',
  styleUrls: ['./order.component.css']
})
export class OrderComponent implements OnInit {
  orderIdInput: string = '';
  currentOrder: Order | null = null;
  errorMessage: string = '';
  loading: boolean = false;
  
  products: Product[] = [];
  cart: { [productId: string]: number } = {};

  buyerEmail: string = '';
  pixQRCodeBase64: string = '';
  pixCopyPaste: string = '';

  constructor(private apiService: ApiService) {}

  ngOnInit() {
    this.loadProducts();
  }

  loadProducts() {
    this.loading = true;
    this.apiService.getProducts().subscribe({
      next: (res) => {
        this.products = res.data || [];
        this.loading = false;
      },
      error: (err) => {
        this.errorMessage = 'Failed to load products';
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
      this.errorMessage = 'Cart is empty';
      return;
    }

    this.loading = true;
    this.apiService.createOrder(items).subscribe({
      next: (res) => {
        this.currentOrder = res.data!;
        this.orderIdInput = this.currentOrder.id;
        this.cart = {}; // Clear cart
        this.loading = false;
      },
      error: (err) => {
        this.errorMessage = err.error?.error || 'Failed to create order';
        this.loading = false;
      }
    });
  }

  lookupOrder() {
    if (!this.orderIdInput) return;
    this.loading = true;
    this.apiService.getOrder(this.orderIdInput).subscribe({
      next: (res) => {
        this.currentOrder = res.data!;
        this.loading = false;
      },
      error: (err) => {
        this.errorMessage = 'Order not found';
        this.loading = false;
      }
    });
  }

  pay() {
    if (!this.currentOrder) return;
    if (!this.buyerEmail) {
      this.errorMessage = 'Por favor, informe um email válido para gerar o Pix.';
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
        this.lookupOrder(); // Refresh status
      },
      error: (err) => {
        this.errorMessage = err.error?.error || 'Payment failed';
        this.loading = false;
      }
    });
  }
}
