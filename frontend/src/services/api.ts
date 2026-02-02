import axios from 'axios'

const API_BASE = '/api'

export interface Product {
  id: string
  name: string
  category: string
  price: number
  image_url: string
  product_url: string
  specs: string
  specs_detail?: Record<string, string | number | boolean> | string
  description?: string
}

export const api = {
  async getProducts(params?: {
    category?: string
    sort?: string
    order?: string
  }): Promise<{ count: number; products: Product[] }> {
    const response = await axios.get(`${API_BASE}/products`, { params })
    return response.data
  },
}
