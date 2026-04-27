import { Component } from '@angular/core';
import { ApiService } from '../../services/api.service';
import { Order, OrderItem } from '../../models/api.models';

@Component({
  selector: 'app-order',
  templateUrl: './order.component.html',
  styleUrls: ['./order.component.css']
})
export class OrderComponent {
  orderIdInput: string = '';
  currentOrder: Order | null = null;
  errorMessage: string = '';
  loading: boolean = false;

  constructor(private apiService: ApiService) {}

  createSampleOrder() {
    this.loading = true;
    const items: OrderItem[] = [
      { product_id: 'prod-1', quantity: 2, price: 50.0 },
      { product_id: 'prod-2', quantity: 1, price: 100.0 }
    ];

    this.apiService.createOrder(items).subscribe({
      next: (res) => {
        this.currentOrder = res.data!;
        this.orderIdInput = this.currentOrder.id;
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
    this.loading = true;
    this.apiService.processPayment(this.currentOrder.id).subscribe({
      next: (res) => {
        this.lookupOrder(); // Refresh status
      },
      error: (err) => {
        this.errorMessage = err.error?.error || 'Payment failed';
        this.loading = false;
      }
    });
  }
}
