import { Injectable } from '@angular/core';
import {
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpInterceptor,
  HttpErrorResponse
} from '@angular/common/http';
import { Observable, throwError } from 'rxjs';
import { catchError } from 'rxjs/operators';

@Injectable()
export class ErrorInterceptor implements HttpInterceptor {

  intercept(request: HttpRequest<unknown>, next: HttpHandler): Observable<HttpEvent<unknown>> {
    return next.handle(request).pipe(
      catchError((error: HttpErrorResponse) => {
        let errorMsg = 'An unknown error occurred!';
        
        if (error.error instanceof ErrorEvent) {
          // Erro do lado do cliente ou rede
          errorMsg = `Network Error: ${error.error.message}`;
        } else {
          // Erro do Backend
          if (error.error && error.error.error) {
            errorMsg = error.error.error;
          } else {
            errorMsg = `Server Error [${error.status}]: ${error.message}`;
          }

          // Tratamentos específicos por status
          switch (error.status) {
            case 400:
              console.error('Validation Error:', errorMsg);
              break;
            case 401:
              console.error('Unauthorized:', errorMsg);
              break;
            case 404:
              console.error('Not Found:', errorMsg);
              break;
            case 409:
              console.error('Conflict:', errorMsg);
              break;
          }
        }

        return throwError(() => new Error(errorMsg));
      })
    );
  }
}
