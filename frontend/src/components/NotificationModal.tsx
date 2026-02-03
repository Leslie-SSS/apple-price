import { useState, useEffect } from 'react'
import { CloseIcon, InfoIcon } from './icons'

interface NewArrivalSubscription {
  id: string
  name: string
  categories: string[]
  max_price: number
  min_price: number
  keywords: string[]
  bark_key: string
  enabled: boolean
  created_at: string
}

interface NotificationModalProps {
  isOpen: boolean
  onClose: () => void
  categories: string[]
}

export default function NotificationModal({ isOpen, onClose, categories }: NotificationModalProps) {
  const [subscriptions, setSubscriptions] = useState<NewArrivalSubscription[]>([])
  const [loading, setLoading] = useState(false)
  const [showBarkHelp, setShowBarkHelp] = useState(false)

  // Form state
  const [name, setName] = useState('')
  const [selectedCategories, setSelectedCategories] = useState<string[]>([])
  const [minPrice, setMinPrice] = useState('')
  const [maxPrice, setMaxPrice] = useState('')
  const [keywords, setKeywords] = useState('')
  const [barkKey, setBarkKey] = useState('')

  useEffect(() => {
    if (isOpen) {
      fetchSubscriptions()
    }
  }, [isOpen])

  const fetchSubscriptions = async () => {
    try {
      const res = await fetch('/api/new-arrival-subscriptions')
      const data = await res.json()
      setSubscriptions(data.subscriptions || [])
    } catch (error) {
      console.error('Failed to fetch subscriptions:', error)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const payload: any = {
        name,
        categories: selectedCategories,
        keywords: keywords ? keywords.split(',').map(k => k.trim()).filter(k => k) : [],
        bark_key: barkKey,
        enabled: true,
      }

      if (minPrice) payload.min_price = parseFloat(minPrice)
      if (maxPrice) payload.max_price = parseFloat(maxPrice)

      const res = await fetch('/api/new-arrival-subscriptions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      if (res.ok) {
        // Reset form
        setName('')
        setSelectedCategories([])
        setMinPrice('')
        setMaxPrice('')
        setKeywords('')
        setBarkKey('')
        fetchSubscriptions()
      }
    } catch (error) {
      console.error('Failed to create subscription:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await fetch(`/api/new-arrival-subscriptions/${id}`, { method: 'DELETE' })
      fetchSubscriptions()
    } catch (error) {
      console.error('Failed to delete subscription:', error)
    }
  }

  const toggleCategory = (cat: string) => {
    setSelectedCategories(prev =>
      prev.includes(cat) ? prev.filter(c => c !== cat) : [...prev, cat]
    )
  }

  if (!isOpen) return null

  return (
    <>
      <div className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50" onClick={onClose} />
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div
          className="bg-white rounded-3xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-hidden animate-slideUp"
          onClick={e => e.stopPropagation()}
        >
          {/* Header */}
          <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-sm border-b border-gray-100 px-6 py-4 flex justify-between items-center">
            <h2 className="text-xl font-semibold text-[#1D1D1F]">上新通知设置</h2>
            <button
              onClick={onClose}
              className="w-9 h-9 flex items-center justify-center rounded-full hover:bg-gray-100 text-gray-500"
            >
              <CloseIcon className="w-5 h-5" />
            </button>
          </div>

          {/* Content */}
          <div className="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
            {/* Bark Info Box */}
            <div className="mb-6 p-4 bg-blue-50 border border-blue-100 rounded-2xl">
              <div className="flex items-start gap-3">
                <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center flex-shrink-0">
                  <span className="text-white text-sm">🔔</span>
                </div>
                <div className="flex-1">
                  <h4 className="text-sm font-semibold text-[#1D1D1F] mb-1">使用 Bark 接收推送通知</h4>
                  <p className="text-xs text-gray-600 mb-2">
                    Bark 是一款开源的 iOS 推送通知工具。在 App Store 搜索下载 "Bark"，获取推送 Key 后即可接收通知。
                  </p>
                  <button
                    type="button"
                    onClick={() => setShowBarkHelp(!showBarkHelp)}
                    className="text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
                  >
                    <InfoIcon className="w-3 h-3" />
                    {showBarkHelp ? '收起使用指南' : '查看使用指南'}
                  </button>
                  {showBarkHelp && (
                    <div className="mt-3 p-3 bg-white rounded-lg text-xs text-gray-600 space-y-2">
                      <p><strong>步骤 1:</strong> 在 App Store 搜索并下载 "Bark" 应用</p>
                      <p><strong>步骤 2:</strong> 打开 Bark 应用，首页会显示你的推送 Key</p>
                      <p><strong>步骤 3:</strong> 复制推送 Key，粘贴到下方输入框中</p>
                      <p><strong>步骤 4:</strong> 设置订阅条件，保存后即可接收通知</p>
                      <p className="text-gray-400 pt-2 border-t border-gray-100">
                        开源项目: <a href="https://github.com/Finb/Bark" target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">github.com/Finb/Bark</a>
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Add New Subscription Form */}
            <form onSubmit={handleSubmit} className="mb-6 p-4 bg-gray-50 rounded-2xl">
              <h3 className="text-sm font-semibold text-[#1D1D1F] mb-3">添加新通知</h3>

              {/* Name */}
              <div className="mb-3">
                <label className="block text-xs text-gray-500 mb-1">名称</label>
                <input
                  type="text"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="例如: MacBook Pro 通知"
                  className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  required
                />
              </div>

              {/* Categories */}
              <div className="mb-3">
                <label className="block text-xs text-gray-500 mb-1">产品类型 (可多选)</label>
                <div className="flex flex-wrap gap-2">
                  {categories.map(cat => (
                    <button
                      key={cat}
                      type="button"
                      onClick={() => toggleCategory(cat)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        selectedCategories.includes(cat)
                          ? 'bg-[#0071E3] text-white'
                          : 'bg-white border border-gray-200 text-[#1D1D1F] hover:bg-gray-50'
                      }`}
                    >
                      {cat}
                    </button>
                  ))}
                </div>
              </div>

              {/* Price Range */}
              <div className="mb-3 flex gap-3">
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">最低价格 (可选)</label>
                  <input
                    type="number"
                    value={minPrice}
                    onChange={e => setMinPrice(e.target.value)}
                    placeholder="0"
                    className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">最高价格 (可选)</label>
                  <input
                    type="number"
                    value={maxPrice}
                    onChange={e => setMaxPrice(e.target.value)}
                    placeholder="无限制"
                    className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  />
                </div>
              </div>

              {/* Keywords */}
              <div className="mb-3">
                <label className="block text-xs text-gray-500 mb-1">关键词 (可选, 逗号分隔)</label>
                <input
                  type="text"
                  value={keywords}
                  onChange={e => setKeywords(e.target.value)}
                  placeholder="例如: M3 Pro, 16GB"
                  className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                />
              </div>

              {/* Bark Key */}
              <div className="mb-3">
                <label className="block text-xs text-gray-500 mb-1">
                  Bark 推送 Key <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={barkKey}
                  onChange={e => setBarkKey(e.target.value)}
                  placeholder="输入 Bark 推送 Key"
                  className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  required
                />
              </div>

              <button
                type="submit"
                disabled={loading}
                className="w-full py-2.5 bg-[#0071E3] text-white rounded-xl font-semibold hover:bg-[#0077ED] transition-colors disabled:opacity-50"
              >
                {loading ? '保存中...' : '保存通知设置'}
              </button>
            </form>

            {/* Existing Subscriptions */}
            <div>
              <h3 className="text-sm font-semibold text-[#1D1D1F] mb-3">已设置的通知</h3>
              {subscriptions.length === 0 ? (
                <p className="text-sm text-gray-400 text-center py-4">暂无通知设置</p>
              ) : (
                <div className="space-y-2">
                  {subscriptions.map(sub => (
                    <div
                      key={sub.id}
                      className="p-3 bg-gray-50 rounded-xl flex items-center justify-between"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-[#1D1D1F]">{sub.name}</span>
                          <span className={`px-1.5 py-0.5 rounded text-[10px] ${sub.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-200 text-gray-500'}`}>
                            {sub.enabled ? '启用' : '禁用'}
                          </span>
                        </div>
                        <div className="text-xs text-gray-500 mt-1 truncate">
                          {sub.categories.length > 0 ? sub.categories.join(', ') : '全部分类'}
                          {sub.min_price > 0 && ` · ¥${sub.min_price}+`}
                          {sub.max_price > 0 && ` · ¥${sub.max_price}-`}
                          {sub.keywords.length > 0 && ` · 关键词: ${sub.keywords.join(', ')}`}
                        </div>
                      </div>
                      <button
                        onClick={() => handleDelete(sub.id)}
                        className="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                      >
                        ×
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
