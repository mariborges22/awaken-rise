import { Component, OnInit } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.css']
})
export class SettingsComponent implements OnInit {
  selectedProvider: string = 'mercadopago';
  isMpConnected: boolean = false;
  tempToken: string = '';
  loading: boolean = false;
  saveSuccess: boolean = false;

  constructor(private http: HttpClient) { }

  ngOnInit(): void {
    this.loadConfig();
  }

  loadConfig() {
    this.loading = true;
    this.http.get<any>('/api/tenants/me/config').subscribe({
      next: (res) => {
        this.loading = false;
        if (res && res.data) {
          const config = res.data;
          this.selectedProvider = config.provider || 'mercadopago';
          if (config.settings && config.settings.access_token) {
            this.isMpConnected = true;
            this.tempToken = config.settings.access_token;
          } else {
            this.isMpConnected = false;
          }
        }
      },
      error: (err) => {
        this.loading = false;
        console.error('Falha ao carregar configurações do inquilino:', err);
      }
    });
  }

  selectProvider(id: string) {
    this.selectedProvider = id;
    this.saveSuccess = false;
  }

  startOAuth() {
    alert('Redirecionando para o Mercado Pago (Fluxo OAuth)...');
    // Em produção, aqui abriria o link de autorização do MP
  }

  saveConfig() {
    if (!this.tempToken) return;

    this.loading = true;
    const payload = {
      provider: this.selectedProvider,
      settings: {
        access_token: this.tempToken
      }
    };

    // Chamada para o novo endpoint seguro
    this.http.put('/api/tenants/me/config', payload).subscribe({
      next: () => {
        this.loading = false;
        this.saveSuccess = true;
        this.isMpConnected = true;
        this.tempToken = '';
        setTimeout(() => this.saveSuccess = false, 5000);
      },
      error: () => {
        this.loading = false;
        alert('Falha ao salvar configuração. Verifique as chaves.');
      }
    });
  }
}
