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

export interface Payment {
  id: string;
  order_id: string;
  amount: number;
  status: 'PENDING' | 'APPROVED' | 'FAILED';
}

export interface ApiResponse<T> {
  status: 'success' | 'error';
  data?: T;
  error?: string;
  correlation_id: string;
}
