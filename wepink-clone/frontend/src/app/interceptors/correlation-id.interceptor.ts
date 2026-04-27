import { Injectable } from '@angular/core';
import { HttpInterceptor, HttpRequest, HttpHandler, HttpEvent } from '@angular/common/http';
import { Observable } from 'rxjs';
import { v4 as uuidv4 } from 'uuid';

@Injectable()
export class CorrelationIdInterceptor implements HttpInterceptor {
  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    const correlationId = uuidv4();
    const cloned = req.clone({
      headers: req.headers.set('X-Correlation-ID', correlationId)
    });
    console.log(`[HTTP Request] ${req.method} ${req.url} | Correlation-ID: ${correlationId}`);
    return next.handle(cloned);
  }
}
