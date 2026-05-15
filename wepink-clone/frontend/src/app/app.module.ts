import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { FormsModule } from '@angular/forms';

import { AppComponent } from './app.component';
import { OrderComponent } from './components/order/order.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { OnboardingComponent } from './components/onboarding/onboarding.component';
import { CorrelationIdInterceptor } from './interceptors/correlation-id.interceptor';

@NgModule({
  declarations: [
    AppComponent,
    OrderComponent,
    DashboardComponent,
    OnboardingComponent
  ],
  imports: [
    BrowserModule,
    HttpClientModule,
    FormsModule
  ],
  providers: [
    { provide: HTTP_INTERCEPTORS, useClass: CorrelationIdInterceptor, multi: true }
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }
