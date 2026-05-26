import { Component } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { TenantService } from '../../core/services/tenant.service';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-onboarding',
  templateUrl: './onboarding.component.html',
  styleUrls: ['./onboarding.component.css']
})
export class OnboardingComponent {
  merchantData = {
    legalName: '',
    cnpj: '',
    contactEmail: '',
    plan: 'starter',
    adminName: '',
    adminPassword: ''
  };

  isCnpjValid: boolean = false;
  loading: boolean = false;
  successMessage: boolean = false;

  constructor(
    private http: HttpClient,
    private tenantService: TenantService,
    private authService: AuthService,
    private router: Router
  ) {}

  validateCNPJ() {
    const cleanCnpj = this.merchantData.cnpj.replace(/[^\d]/g, '');
    this.isCnpjValid = cleanCnpj.length === 14;
    
    if (cleanCnpj.length === 14) {
      this.merchantData.cnpj = cleanCnpj.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, "$1.$2.$3/$4-$5");
    }
  }

  submit() {
    if (!this.isCnpjValid || !this.merchantData.adminName || !this.merchantData.adminPassword) {
      alert('Preencha todos os campos corretamente.');
      return;
    }

    this.loading = true;
    const tenantId = 'tnt_' + Math.random().toString(36).substring(7);
    const tenantPayload = {
      tenant_id: tenantId,
      legal_name: this.merchantData.legalName,
      cnpj: this.merchantData.cnpj.replace(/[^\d]/g, ''),
      contact_email: this.merchantData.contactEmail,
      plan: this.merchantData.plan
    };

    // 1. Create Tenant
    this.http.post('/api/tenants', tenantPayload).subscribe({
      next: (res) => {
        this.tenantService.setTenant(tenantId);
        
        // 2. Register Admin User
        this.authService.register(
          tenantId,
          this.merchantData.adminName,
          this.merchantData.contactEmail,
          this.merchantData.adminPassword,
          'admin'
        ).subscribe({
          next: () => {
            // 3. Auto Login
            this.authService.login(this.merchantData.contactEmail, this.merchantData.adminPassword).subscribe({
              next: () => {
                this.loading = false;
                this.successMessage = true;
              },
              error: () => {
                this.loading = false;
                alert('Conta criada, mas falha no login automático. Faça login manualmente.');
                this.router.navigate(['/saas/login']);
              }
            });
          },
          error: (err) => {
            this.loading = false;
            alert('Loja criada, mas falha ao criar conta de administrador.');
          }
        });
      },
      error: (err) => {
        this.loading = false;
        alert('Erro ao registrar a loja. Verifique os dados ou a conexão.');
      }
    });
  }

  goToDashboard() {
    this.router.navigate(['/saas/dashboard']);
  }
}
