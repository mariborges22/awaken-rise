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

  // Para o Teste de Carga e dev local, estamos chumbando o ID do lojista.
  // No lançamento oficial, isso virá da URL (ex: meusaas.com/loja/:tenantId)
  private getHeaders() {
    return {
      headers: {
        'X-Tenant-ID': 'tenant-loadtest-01'
      }
    };
  }

  getProducts(): Observable<ApiResponse<Product[]>> {
    return this.http.get<ApiResponse<Product[]>>(`${this.baseUrl}/products`, this.getHeaders());
  }

  createOrder(items: OrderItem[]): Observable<ApiResponse<Order>> {
    return this.http.post<ApiResponse<Order>>(`${this.baseUrl}/orders`, { items }, this.getHeaders());
  }

  getOrder(id: string): Observable<ApiResponse<Order>> {
    return this.http.get<ApiResponse<Order>>(`${this.baseUrl}/orders/${id}`, this.getHeaders());
  }

  processPayment(orderId: string, payload: { payment_method: string, buyer_email: string }): Observable<ApiResponse<Payment>> {
    return this.http.post<ApiResponse<Payment>>(`${this.baseUrl}/payments/${orderId}`, payload, this.getHeaders());
  }
}
