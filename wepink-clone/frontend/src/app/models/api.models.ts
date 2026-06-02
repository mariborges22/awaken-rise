export interface Product {
  id: string;
  name: string;
  price: number;
  stock: number;
}

export interface OrderItem {
  product_id: string;
  quantity: number;
  price?: number;
}

export interface Order {
  id: string;
  items: OrderItem[];
  total: number;
  status: 'PENDING' | 'CONFIRMED' | 'CANCELLED';
}

export interface GatewayResponse {
  Success: boolean;
  TransactionID: string;
  Status: string;
  PaymentURL: string;
  PixQRCodeBase64: string;
  PixCopyPaste: string;
  ErrorMessage: string;
}

export interface Payment {
  payment_id: string;
  order_id: string;
  status: 'PENDING' | 'APPROVED' | 'FAILED';
  gateway_response?: GatewayResponse;
}

export interface ApiResponse<T> {
  status: 'success' | 'error';
  data?: T;
  error?: string;
  correlation_id: string;
}
