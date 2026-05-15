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
    // Aqui buscaríamos o status atual do banco
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
