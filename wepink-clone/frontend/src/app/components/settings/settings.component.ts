import { Component, OnInit } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { ApiService } from '../../services/api.service';

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

  constructor(private apiService: ApiService, private http: HttpClient) { }

  ngOnInit(): void {
    // Verifica se voltamos do callback do Mercado Pago via query string
    const params = new URLSearchParams(window.location.search);
    const status = params.get('status');
    if (status === 'connected') {
      this.statusMessage = 'Conta Mercado Pago conectada com sucesso! 🎉';
      this.statusIsError = false;
      this.isMpConnected = true;
      history.replaceState(null, '', window.location.pathname);
    } else if (status === 'oauth_error') {
      this.statusMessage = 'Falha ao conectar com o Mercado Pago. Tente novamente.';
      this.statusIsError = true;
      history.replaceState(null, '', window.location.pathname);
    }
    this.loadConfig();
  }

  loadConfig() {
    this.loading = true;
    // Usa HttpClient diretamente pois o AuthInterceptor injetará o JWT automaticamente
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
    // getMpOAuthUrl usa o ApiService que terá o JWT injetado pelo AuthInterceptor
    this.apiService.getMpOAuthUrl().subscribe({
      next: (res) => {
        if (res?.data?.url) {
          window.location.href = res.data.url;
        } else {
          this.loading = false;
          this.statusMessage = 'URL de autorização inválida. Tente novamente.';
          this.statusIsError = true;
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
