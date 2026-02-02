import { useState } from 'react'
import Header from './components/Header'
import Home from './pages/Home'
import NotificationModal from './components/NotificationModal'

function App() {
  const [filteredCount, setFilteredCount] = useState<number>(0)
  const [isNotificationOpen, setIsNotificationOpen] = useState(false)
  const [categories, setCategories] = useState<string[]>([])

  return (
    <div className="min-h-screen bg-[#F5F5F7]">
      <Header productCount={filteredCount} onOpenNotifications={() => setIsNotificationOpen(true)} />
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
