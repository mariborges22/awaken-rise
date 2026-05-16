import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { OnboardingComponent } from './components/onboarding/onboarding.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { SettingsComponent } from './components/settings/settings.component';
import { OrderComponent } from './components/order/order.component';
import { LandingPageComponent } from './components/landing-page/landing-page.component';

const routes: Routes = [
  // Public Storefront (Vitrine e Checkout)
  { path: 'store/:tenantId', component: OrderComponent }, // Checkout para uma loja
  
  // SaaS Dashboard (Lojistas)
  { path: 'saas/onboarding', component: OnboardingComponent },
  { path: 'saas/dashboard', component: DashboardComponent },
  { path: 'saas/settings', component: SettingsComponent },

  // Default Redirect
  { path: '', component: LandingPageComponent, pathMatch: 'full' },
  { path: '**', redirectTo: '' }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
