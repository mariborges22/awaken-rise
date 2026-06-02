import { Component, OnInit } from '@angular/core';
import { ApiService } from '../../services/api.service';
import { Product } from '../../models/api.models';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {
  isVerified: boolean = true;
  todayRevenue: number = 1250.50;
  todayOrders: number = 24;
  conversionRate: number = 3.8;
  
  usageCount: number = 65;
  planLimit: number = 100;
  planType: string = 'starter';

  // Product Management
  products: Product[] = [];
  newProduct = { name: '', price: 0, stock: 0 };
  loadingProducts: boolean = false;
  creatingProduct: boolean = false;

  constructor(private apiService: ApiService, private http: HttpClient) { }

  ngOnInit(): void {
    this.loadProducts();
  }

  loadProducts() {
    this.loadingProducts = true;
    this.apiService.getProducts().subscribe({
      next: (res) => {
        this.products = res.data || [];
        this.loadingProducts = false;
      },
      error: () => {
        this.loadingProducts = false;
      }
    });
  }

  createProduct() {
    if (!this.newProduct.name || this.newProduct.price <= 0) return;
    this.creatingProduct = true;
    
    this.http.post('/api/products', this.newProduct).subscribe({
      next: () => {
        this.newProduct = { name: '', price: 0, stock: 0 };
        this.creatingProduct = false;
        this.loadProducts();
      },
      error: () => {
        alert('Erro ao criar produto');
        this.creatingProduct = false;
      }
    });
  }

  upgradePlan() {
    // Redireciona para o link de assinatura master do SaaS
    window.location.href = 'https://www.mercadopago.com.br/subscriptions/checkout?preapproval_plan_id=fake_plan_id';
  }
}
