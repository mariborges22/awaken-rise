import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { OnboardingComponent } from './components/onboarding/onboarding.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { SettingsComponent } from './components/settings/settings.component';
import { OrderComponent } from './components/order/order.component';
import { LandingPageComponent } from './components/landing-page/landing-page.component';
import { LoginComponent } from './components/login/login.component';
import { AuthGuard } from './core/guards/auth.guard';

const routes: Routes = [
  // ─────────────────────────────────────────────
  // Vitrine Pública (Sem login)
  // O :tenantId é lido pelo OrderComponent e registrado no TenantService.
  // O TenantInterceptor injeta X-Tenant-ID automaticamente em todas as requisições.
  // ─────────────────────────────────────────────
  { path: 'loja/:tenantId', component: OrderComponent },

  // ─────────────────────────────────────────────
  // Autenticação (Acesso Público)
  // ─────────────────────────────────────────────
  { path: 'login', component: LoginComponent },

  // ─────────────────────────────────────────────
  // Painel Admin do SaaS (Protegido pelo AuthGuard)
  // ─────────────────────────────────────────────
  { path: 'saas/onboarding', component: OnboardingComponent },
  {
    path: 'saas/dashboard',
    component: DashboardComponent,
    canActivate: [AuthGuard]
  },
  {
    path: 'saas/settings',
    component: SettingsComponent,
    canActivate: [AuthGuard]
  },

  // ─────────────────────────────────────────────
  // Defaults
  // ─────────────────────────────────────────────
  { path: '', component: LandingPageComponent, pathMatch: 'full' },
  { path: '**', redirectTo: '' }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
