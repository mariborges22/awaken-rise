import { Component } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { TenantService } from '../../core/services/tenant.service';

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
    plan: 'starter'
  };

  isCnpjValid: boolean = false;
  loading: boolean = false;
  successMessage: boolean = false;

  constructor(
    private http: HttpClient,
    private tenantService: TenantService,
    private router: Router
  ) {}

  validateCNPJ() {
    // Remove caracteres não numéricos
    const cleanCnpj = this.merchantData.cnpj.replace(/[^\d]/g, '');
    
    // Validação básica de tamanho (14 dígitos)
    this.isCnpjValid = cleanCnpj.length === 14;
    
    // Formatação visual automática (opcional, mas melhora UX)
    if (cleanCnpj.length === 14) {
      this.merchantData.cnpj = cleanCnpj.replace(/^(\d{2})(\d{3})(\d{3})(\d{4})(\d{2})$/, "$1.$2.$3/$4-$5");
    }
  }

  submit() {
    if (!this.isCnpjValid) return;

    this.loading = true;
    const tenantId = 'tnt_' + Math.random().toString(36).substring(7);
    const payload = {
      tenant_id: tenantId,
      legal_name: this.merchantData.legalName,
      cnpj: this.merchantData.cnpj.replace(/[^\d]/g, ''),
      contact_email: this.merchantData.contactEmail,
      plan: this.merchantData.plan
    };

    // Chamada real para o Backend
    this.http.post('/api/tenants', payload).subscribe({
      next: (res) => {
        this.loading = false;
        this.tenantService.setTenant(tenantId); // Grava o Tenant ID no estado e localStorage
        this.successMessage = true;
      },
      error: (err) => {
        this.loading = false;
        alert('Erro ao registrar sistema. Verifique os dados ou a conexão.');
      }
    });
  }

  goToDashboard() {
    this.router.navigate(['/saas/dashboard']);
  }
}
