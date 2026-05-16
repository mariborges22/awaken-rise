import { Injectable } from '@angular/core';
import {
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpInterceptor
} from '@angular/common/http';
import { Observable } from 'rxjs';
import { TenantService } from '../core/services/tenant.service';

@Injectable()
export class TenantInterceptor implements HttpInterceptor {
  constructor(private tenantService: TenantService) {}

  intercept(request: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    const currentTenant = this.tenantService.getCurrentTenant();

    if (currentTenant) {
      const clonedReq = request.clone({
        headers: request.headers.set('X-Tenant-ID', currentTenant)
      });
      return next.handle(clonedReq);
    }

    return next.handle(request);
  }
}
