import { Component, OnInit } from '@angular/core';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css']
})
export class DashboardComponent implements OnInit {
  // Mock data - Em breve virá do Backend
  isVerified: boolean = true;
  todayRevenue: number = 1250.50;
  todayOrders: number = 24;
  conversionRate: number = 3.8;
  
  usageCount: number = 65;
  planLimit: number = 100;
  planType: string = 'starter';

  constructor() { }

  ngOnInit(): void {
    // Aqui faremos o fetch dos dados do Tenant
  }
}
