import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class TenantService {
  // Em um caso real, o tenant inicial poderia vir da URL (ex: dominio.com/:tenantId)
  private tenantIdSubject = new BehaviorSubject<string | null>(null);
  tenantId$ = this.tenantIdSubject.asObservable();

  setTenant(tenantId: string) {
    this.tenantIdSubject.next(tenantId);
    // Também poderíamos salvar no localStorage se fosse persistente entre abas
    localStorage.setItem('current_tenant', tenantId);
  }

  getCurrentTenant(): string | null {
    // Tenta pegar do state, se não, pega do localStorage
    const stateVal = this.tenantIdSubject.getValue();
    if (stateVal) return stateVal;
    
    return localStorage.getItem('current_tenant');
  }

  clearTenant() {
    this.tenantIdSubject.next(null);
    localStorage.removeItem('current_tenant');
  }
}
