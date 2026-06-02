import { Component, OnInit } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.css']
})
export class SettingsComponent implements OnInit {
  isMpConnected: boolean = false;
  loading: boolean = false;
  statusMessage: string = '';
  statusIsError: boolean = false;

  constructor(private http: HttpClient) { }

  ngOnInit(): void {
    // Verificar se voltamos do callback do Mercado Pago
    const params = new URLSearchParams(window.location.hash.split('?')[1] || '');
    const status = params.get('status');
    if (status === 'connected') {
      this.statusMessage = 'Conta Mercado Pago conectada com sucesso! 🎉';
      this.statusIsError = false;
      this.isMpConnected = true;
      // Limpar a URL sem recarregar a página
      history.replaceState(null, '', window.location.pathname + '#/settings');
    } else if (status === 'oauth_error') {
      this.statusMessage = 'Falha ao conectar com o Mercado Pago. Tente novamente.';
      this.statusIsError = true;
      history.replaceState(null, '', window.location.pathname + '#/settings');
    }
    this.loadConfig();
  }

  loadConfig() {
    this.loading = true;
    this.http.get<any>('/api/tenants/me/config').subscribe({
      next: (res) => {
        this.loading = false;
        if (res?.data?.settings?.access_token) {
          this.isMpConnected = true;
        }
      },
      error: () => {
        this.loading = false;
      }
    });
  }

  startOAuth() {
    this.loading = true;
    // Buscar a URL de autorização gerada pelo backend (contém o state anti-CSRF)
    this.http.get<any>('/api/auth/mercadopago/url').subscribe({
      next: (res) => {
        if (res?.data?.url) {
          // Redirecionar o lojista para o site oficial do Mercado Pago
          window.location.href = res.data.url;
        }
      },
      error: () => {
        this.loading = false;
        this.statusMessage = 'Não foi possível iniciar a conexão. Tente novamente.';
        this.statusIsError = true;
      }
    });
  }
}
