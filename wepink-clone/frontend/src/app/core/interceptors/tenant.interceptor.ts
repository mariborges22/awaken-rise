import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { TenantService } from '../services/tenant.service';

export const tenantInterceptor: HttpInterceptorFn = (req, next) => {
  const tenantService = inject(TenantService);
  const currentTenant = tenantService.getCurrentTenant();

  if (currentTenant) {
    const clonedReq = req.clone({
      headers: req.headers.set('X-Tenant-ID', currentTenant)
    });
    return next(clonedReq);
  }

  return next(req);
};
