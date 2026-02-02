import { useState, useMemo, useEffect } from 'react'
import { useProducts } from '@/hooks/useProducts'
import ProductCard from '@/components/ProductCard'
import { SearchIcon, CloseIcon, FilterIcon } from '@/components/icons'

interface HomeProps {
  onFilteredCountChange?: (count: number) => void
  onCategoriesChange?: (categories: string[]) => void
}

const CHIP_OPTIONS = ['全部', 'M1', 'M2', 'M3', 'M3 Pro', 'M3 Max', 'M4', 'M4 Pro', 'M4 Max']

const STORAGE_OPTIONS = ['全部', '256GB', '512GB', '1TB', '2TB', '4TB', '8TB']

const MEMORY_OPTIONS = ['全部', '8GB', '16GB', '24GB', '32GB', '64GB', '128GB', '256GB']

const PRICE_PRESETS = [
  { label: '全部', min: 0, max: Infinity },
  { label: '¥5000以下', min: 0, max: 5000 },
  { label: '¥5000-10000', min: 5000, max: 10000 },
  { label: '¥10000-15000', min: 10000, max: 15000 },
  { label: '¥15000-20000', min: 15000, max: 20000 },
  { label: '¥20000以上', min: 20000, max: Infinity },
]

const SORT_OPTIONS = [
  { label: '默认', value: 'default' },
  { label: '价格低到高', value: 'price_asc' },
  { label: '价格高到低', value: 'price_desc' },
]

export default function Home({ onFilteredCountChange, onCategoriesChange }: HomeProps) {
  const [categoryFilter, setCategoryFilter] = useState<string>('全部')
  const [chipFilter, setChipFilter] = useState<string>('全部')
  const [pricePreset, setPricePreset] = useState<number>(0)
  const [storageFilter, setStorageFilter] = useState<string>('全部')
  const [memoryFilter, setMemoryFilter] = useState<string>('全部')
  const [searchQuery, setSearchQuery] = useState<string>('')
  const [sortBy, setSortBy] = useState<string>('default')
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false)

  const { products, loading } = useProducts({
    category: '',
    sort: 'score',
    order: 'desc',
  })

  const categories = useMemo(() => {
    if (!products) return ['全部']
    const cats = new Set(products.map(p => p.category))
    return ['全部', ...Array.from(cats).sort()]
  }, [products])

  const filteredProducts = useMemo(() => {
    if (!products) return []
    let result = [...products]

    if (categoryFilter !== '全部') {
      result = result.filter(p => p.category === categoryFilter)
    }

    if (chipFilter !== '全部') {
      if (chipFilter === 'M3 Pro') {
        result = result.filter(p => p.name.includes('M3 Pro'))
      } else if (chipFilter === 'M3 Max') {
        result = result.filter(p => p.name.includes('M3 Max'))
      } else if (chipFilter === 'M4 Pro') {
        result = result.filter(p => p.name.includes('M4 Pro'))
      } else if (chipFilter === 'M4 Max') {
        result = result.filter(p => p.name.includes('M4 Max'))
      } else {
        result = result.filter(p => p.name.includes(chipFilter))
      }
    }

    // Price filter using preset
    const preset = PRICE_PRESETS[pricePreset]
    if (preset.max !== Infinity) {
      result = result.filter(p => p.price >= preset.min && p.price <= preset.max)
    } else {
      result = result.filter(p => p.price >= preset.min)
    }

    if (storageFilter !== '全部') {
      result = result.filter(p =>
        p.name.includes(storageFilter) ||
        (typeof p.specs_detail === 'string' && p.specs_detail.includes(storageFilter))
      )
    }

    if (memoryFilter !== '全部') {
      result = result.filter(p =>
        p.name.includes(memoryFilter) ||
        (typeof p.specs_detail === 'string' && p.specs_detail.includes(memoryFilter))
      )
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase()
      result = result.filter(p =>
        p.name.toLowerCase().includes(query) ||
        p.category.toLowerCase().includes(query) ||
        (p.description && p.description.toLowerCase().includes(query))
      )
    }

    // Sort
    switch (sortBy) {
      case 'price_asc':
        result.sort((a, b) => a.price - b.price)
        break
      case 'price_desc':
        result.sort((a, b) => b.price - a.price)
        break
      default:
        // Keep original order
        break
    }

    return result
  }, [products, categoryFilter, chipFilter, pricePreset, storageFilter, memoryFilter, searchQuery, sortBy])

  // Notify parent of filtered count
  useEffect(() => {
    onFilteredCountChange?.(filteredProducts.length)
  }, [filteredProducts.length, onFilteredCountChange])

  // Notify parent of categories
  useEffect(() => {
    onCategoriesChange?.(categories.filter(c => c !== '全部'))
  }, [categories, onCategoriesChange])

  const resetFilters = () => {
    setCategoryFilter('全部')
    setChipFilter('全部')
    setPricePreset(0)
    setStorageFilter('全部')
    setMemoryFilter('全部')
    setSearchQuery('')
    setSortBy('default')
  }

  const hasActiveFilters = categoryFilter !== '全部' || chipFilter !== '全部' ||
    pricePreset !== 0 || storageFilter !== '全部' || memoryFilter !== '全部' || !!searchQuery

  if (loading) {
    return (
      <div className="min-h-screen bg-[#F5F5F7] flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-3 border-[#0071E3] border-t-transparent rounded-full animate-spin mx-auto mb-3" />
          <p className="text-sm text-gray-500">加载中...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[#F5F5F7]">
      <div className="max-w-7xl mx-auto px-4 py-4">
        {/* Search */}
        <div className="mb-4">
          <div className="relative">
            <div className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400">
              <SearchIcon />
            </div>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="搜索产品名称、规格..."
              className="w-full pl-11 pr-10 py-2.5 bg-white border border-gray-200 rounded-xl text-sm text-[#1D1D1F] placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-[#0071E3] focus:border-transparent transition-all"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 p-1"
              >
                <CloseIcon />
              </button>
            )}
          </div>
        </div>

        {/* Category Filter Row */}
        <div className="flex items-center gap-2 mb-3 overflow-x-auto pb-1 scrollbar-hide">
          <span className="text-xs text-gray-500 flex-shrink-0">分类:</span>
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setCategoryFilter(cat === categoryFilter ? '全部' : cat)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-all ${
                categoryFilter === cat
                  ? 'bg-[#0071E3] text-white'
                  : 'bg-white border border-gray-200 text-[#1D1D1F] hover:bg-gray-50'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>

        {/* Price Filter Row */}
        <div className="flex items-center gap-2 mb-3 overflow-x-auto pb-1 scrollbar-hide">
          <span className="text-xs text-gray-500 flex-shrink-0">价格:</span>
          {PRICE_PRESETS.map((preset, index) => (
            <button
              key={preset.label}
              onClick={() => setPricePreset(index === pricePreset ? 0 : index)}
              className={`px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-all ${
                pricePreset === index
                  ? 'bg-[#0071E3] text-white'
                  : 'bg-white border border-gray-200 text-[#1D1D1F] hover:bg-gray-50'
              }`}
            >
              {preset.label}
            </button>
          ))}
        </div>

        {/* Advanced Filters Toggle */}
        <button
          onClick={() => setShowAdvancedFilters(!showAdvancedFilters)}
          className="flex items-center gap-1.5 mb-3 text-xs text-gray-500 hover:text-[#0071E3] transition-colors"
        >
          <FilterIcon />
          {showAdvancedFilters ? '收起筛选' : '更多筛选选项'}
          {hasActiveFilters && !showAdvancedFilters && (
            <span className="ml-1 px-1.5 py-0.5 bg-[#0071E3] text-white rounded-full">
              {Object.values({
                category: categoryFilter, chip: chipFilter, price: pricePreset,
                storage: storageFilter, memory: memoryFilter
              }).filter(v => v !== '全部' && v !== 0).length}
            </span>
          )}
        </button>

        {/* Advanced Filters */}
        {showAdvancedFilters && (
          <div className="mb-4 p-4 bg-white rounded-2xl border border-gray-200">
            {/* Chip Filter */}
            <div className="mb-4">
              <div className="text-xs text-gray-500 mb-2">芯片</div>
              <div className="flex flex-wrap gap-2">
                {CHIP_OPTIONS.map(chip => (
                  <button
                    key={chip}
                    onClick={() => setChipFilter(chip === chipFilter ? '全部' : chip)}
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      chipFilter === chip
                        ? 'bg-[#0071E3] text-white'
                        : 'bg-gray-100 text-[#1D1D1F] hover:bg-gray-200'
                    }`}
                  >
                    {chip}
                  </button>
                ))}
              </div>
            </div>

            {/* Storage & Memory Filters */}
            <div className="grid grid-cols-2 gap-4 mb-4">
              <div>
                <div className="text-xs text-gray-500 mb-2">存储</div>
                <div className="flex flex-wrap gap-2">
                  {STORAGE_OPTIONS.map(storage => (
                    <button
                      key={storage}
                      onClick={() => setStorageFilter(storage === storageFilter ? '全部' : storage)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        storageFilter === storage
                          ? 'bg-[#0071E3] text-white'
                          : 'bg-gray-100 text-[#1D1D1F] hover:bg-gray-200'
                      }`}
                    >
                      {storage}
                    </button>
                  ))}
                </div>
              </div>
              <div>
                <div className="text-xs text-gray-500 mb-2">内存</div>
                <div className="flex flex-wrap gap-2">
                  {MEMORY_OPTIONS.map(memory => (
                    <button
                      key={memory}
                      onClick={() => setMemoryFilter(memory === memoryFilter ? '全部' : memory)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        memoryFilter === memory
                          ? 'bg-[#0071E3] text-white'
                          : 'bg-gray-100 text-[#1D1D1F] hover:bg-gray-200'
                      }`}
                    >
                      {memory}
                    </button>
                  ))}
                </div>
              </div>
            </div>

            {/* Clear Filters */}
            {hasActiveFilters && (
              <button
                onClick={resetFilters}
                className="w-full py-2 bg-gray-100 hover:bg-gray-200 rounded-lg text-xs text-[#1D1D1F] font-medium transition-colors"
              >
                清除所有筛选
              </button>
            )}
          </div>
        )}

        {/* Sort & Result Count */}
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="text-sm text-gray-600">
              找到 <span className="font-semibold text-[#0071E3]">{filteredProducts.length}</span> 款产品
            </span>
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
              className="text-xs bg-white border border-gray-200 rounded-lg px-3 py-1.5 text-[#1D1D1F] focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
            >
              {SORT_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Active Filter Tags */}
        {hasActiveFilters && !showAdvancedFilters && (
          <div className="flex flex-wrap gap-2 mb-4">
            {categoryFilter !== '全部' && (
              <span className="inline-flex items-center gap-1 px-2.5 py-1 bg-white border border-gray-200 rounded-full text-xs">
                {categoryFilter}
                <button onClick={() => setCategoryFilter('全部')} className="ml-1 text-gray-400 hover:text-gray-600">×</button>
              </span>
            )}
            {chipFilter !== '全部' && (
              <span className="inline-flex items-center gap-1 px-2.5 py-1 bg-white border border-gray-200 rounded-full text-xs">
                {chipFilter}
                <button onClick={() => setChipFilter('全部')} className="ml-1 text-gray-400 hover:text-gray-600">×</button>
              </span>
            )}
            {storageFilter !== '全部' && (
              <span className="inline-flex items-center gap-1 px-2.5 py-1 bg-white border border-gray-200 rounded-full text-xs">
                {storageFilter}
                <button onClick={() => setStorageFilter('全部')} className="ml-1 text-gray-400 hover:text-gray-600">×</button>
              </span>
            )}
            {memoryFilter !== '全部' && (
              <span className="inline-flex items-center gap-1 px-2.5 py-1 bg-white border border-gray-200 rounded-full text-xs">
                {memoryFilter}
                <button onClick={() => setMemoryFilter('全部')} className="ml-1 text-gray-400 hover:text-gray-600">×</button>
              </span>
            )}
          </div>
        )}

        {/* Product List - Single Column */}
        {filteredProducts.length > 0 ? (
          <div className="space-y-3">
            {filteredProducts.map(product => (
              <ProductCard
                key={product.id}
                product={product}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-20">
            <div className="text-4xl mb-3">🔍</div>
            <p className="text-gray-500 mb-1">没有找到匹配的产品</p>
            <p className="text-xs text-gray-400 mb-5">试试调整筛选条件</p>
            {hasActiveFilters && (
              <button
                onClick={resetFilters}
                className="px-5 py-2 bg-[#0071E3] text-white rounded-xl hover:bg-[#0077ED] transition-colors text-sm"
              >
                清除筛选
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
