import { useState, useEffect } from 'react'
import { api, Product } from '@/services/api'

interface UseProductsOptions {
  category?: string
  sort?: string
  order?: string
}

export function useProducts(options: UseProductsOptions = {}) {
  const [products, setProducts] = useState<Product[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchProducts = async () => {
      setLoading(true)
      setError(null)

      try {
        const response = await api.getProducts(options)
        setProducts(response.products)
      } catch (err) {
        setError('Failed to fetch products')
        console.error(err)
      } finally {
        setLoading(false)
      }
    }

    fetchProducts()
  }, [options.category, options.sort, options.order])

  return { products, loading, error }
}
