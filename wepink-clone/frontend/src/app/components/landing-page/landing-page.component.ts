import { Component } from '@angular/core';

@Component({
  selector: 'app-landing-page',
  templateUrl: './landing-page.component.html',
  styleUrls: ['./landing-page.component.scss']
})
export class LandingPageComponent {
  // Número de exemplo, o lojista poderá trocar pelo dele
  whatsappNumber = '5511999999999'; 
  
  openWhatsApp() {
    const text = encodeURIComponent('Olá! Gostaria de saber mais sobre a plataforma Awaken Rise para o meu e-commerce.');
    window.open(`https://wa.me/${this.whatsappNumber}?text=${text}`, '_blank');
  }
}
