import { BellIcon } from './icons'

interface HeaderProps {
  productCount?: number
  onOpenNotifications?: () => void
}

export default function Header({ productCount, onOpenNotifications }: HeaderProps) {
  return (
    <header className="bg-white/80 backdrop-blur-md sticky top-0 z-40 border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex items-center justify-between h-14">
          <div>
            <span className="text-lg font-semibold text-[#1D1D1F]">ApplePrice</span>
            <span className="ml-2 text-xs text-gray-500">官方翻新 · 固定85折</span>
          </div>
          <div className="flex items-center gap-4">
            {productCount !== undefined && (
              <div className="text-right">
                <span className="text-base font-semibold text-[#0071E3]">{productCount}</span>
                <span className="text-xs text-gray-400 ml-1">款产品</span>
              </div>
            )}
            <button
              onClick={onOpenNotifications}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-[#0071E3] text-white rounded-lg text-sm font-medium hover:bg-[#0077ED] transition-colors"
            >
              <BellIcon className="w-4 h-4" />
              上新通知
            </button>
          </div>
        </div>
      </div>
    </header>
  )
}
