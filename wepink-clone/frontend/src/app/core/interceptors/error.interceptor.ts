import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { catchError, throwError } from 'rxjs';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  return next(req).pipe(
    catchError((error: HttpErrorResponse) => {
      let errorMsg = 'An unknown error occurred!';
      
      if (error.error instanceof ErrorEvent) {
        // Erro do lado do cliente ou rede
        errorMsg = `Network Error: ${error.error.message}`;
      } else {
        // Erro do Backend (Nosso HandleError mapeado)
        // O backend retorna: { "status": "error", "error": "...", "correlation_id": "..." }
        if (error.error && error.error.error) {
          errorMsg = error.error.error;
        } else {
          errorMsg = `Server Error [${error.status}]: ${error.message}`;
        }

        // Tratamentos específicos por status
        switch (error.status) {
          case 400:
            console.error('Validation Error:', errorMsg);
            // Aqui poderíamos injetar um ToastService para mostrar na tela
            break;
          case 401:
            console.error('Unauthorized:', errorMsg);
            // Redirecionar para login?
            break;
          case 404:
            console.error('Not Found:', errorMsg);
            break;
          case 409:
            console.error('Conflict:', errorMsg);
            break;
        }
      }

      // Re-throw para o componente lidar se precisar (ex: marcar input como inválido)
      return throwError(() => new Error(errorMsg));
    })
  );
};
