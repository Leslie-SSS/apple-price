import { useState, useEffect } from 'react'
import Header, { SystemStatus } from './components/Header'
import Home from './pages/Home'
import NotificationModal from './components/NotificationModal'

interface StatsResponse {
  total_products: number
  available_products: number
  categories: Record<string, number>
  last_scrape_time: string
  total_subscriptions: number
  scraper_status?: SystemStatus
}

function App() {
  const [filteredCount, setFilteredCount] = useState<number>(0)
  const [isNotificationOpen, setIsNotificationOpen] = useState(false)
  const [categories, setCategories] = useState<string[]>([])
  const [systemStatus, setSystemStatus] = useState<SystemStatus | undefined>()

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const res = await fetch('/api/stats')
        const data: StatsResponse = await res.json()
        if (data.scraper_status) {
          setSystemStatus(data.scraper_status)
        }
      } catch (err) {
        console.error('Failed to fetch stats:', err)
      }
    }

    fetchStatus()
    const interval = setInterval(fetchStatus, 60000) // Every minute
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="min-h-screen bg-[#F5F5F7]">
      <Header
        productCount={filteredCount}
        onOpenNotifications={() => setIsNotificationOpen(true)}
        systemStatus={systemStatus}
      />
      <Home
        onFilteredCountChange={setFilteredCount}
        onCategoriesChange={setCategories}
      />
      <NotificationModal
        isOpen={isNotificationOpen}
        onClose={() => setIsNotificationOpen(false)}
        categories={categories}
      />
    </div>
  )
}

export default App
