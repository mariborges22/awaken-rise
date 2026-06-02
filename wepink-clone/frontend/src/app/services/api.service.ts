import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Order, Payment, ApiResponse, OrderItem, Product } from '../models/api.models';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private baseUrl = '/api';

  constructor(private http: HttpClient) {}

  // O X-Tenant-ID é injetado automaticamente pelo TenantInterceptor.
  // O JWT de lojista é injetado automaticamente pelo AuthInterceptor.
  // Não precisamos passar headers manualmente aqui.

  getProducts(): Observable<ApiResponse<Product[]>> {
    return this.http.get<ApiResponse<Product[]>>(`${this.baseUrl}/products`);
  }

  createOrder(items: OrderItem[]): Observable<ApiResponse<Order>> {
    return this.http.post<ApiResponse<Order>>(`${this.baseUrl}/orders`, { items });
  }

  getOrder(id: string): Observable<ApiResponse<Order>> {
    return this.http.get<ApiResponse<Order>>(`${this.baseUrl}/orders/${id}`);
  }

  processPayment(orderId: string, payload: { payment_method: string, buyer_email: string }): Observable<ApiResponse<Payment>> {
    return this.http.post<ApiResponse<Payment>>(`${this.baseUrl}/payments/${orderId}`, payload);
  }

  // Métodos Admin (JWT injetado pelo AuthInterceptor)
  getMpOAuthUrl(): Observable<ApiResponse<{ url: string }>> {
    return this.http.get<ApiResponse<{ url: string }>>(`${this.baseUrl}/auth/mercadopago/url`);
  }
}
