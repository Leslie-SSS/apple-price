import { BellIcon } from "./icons";

export interface SystemStatus {
  last_scrape_time: string;
  last_scrape_status: "success" | "failed" | "running" | "never";
  last_scrape_error?: string;
  products_scraped: number;
  duration_ms?: number;
}

interface HeaderProps {
  productCount?: number;
  onOpenNotifications?: () => void;
  systemStatus?: SystemStatus;
}

function formatLastUpdate(time: string): string {
  if (
    !time ||
    time === "0001-01-01T00:00:00Z" ||
    time.startsWith("0001-01-01")
  ) {
    return "从未更新";
  }
  const date = new Date(time);
  if (isNaN(date.getTime())) {
    return "从未更新";
  }
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / 60000);

  if (minutes < 1) return "刚刚更新";
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  return `${Math.floor(hours / 24)}天前`;
}

export default function Header({
  productCount,
  onOpenNotifications,
  systemStatus,
}: HeaderProps) {
  const getStatusColor = () => {
    if (!systemStatus) return "bg-gray-400";
    switch (systemStatus.last_scrape_status) {
      case "success":
        return "bg-green-500";
      case "failed":
        return "bg-red-500";
      case "running":
        return "bg-blue-500 animate-pulse";
      default:
        return "bg-gray-400";
    }
  };

  return (
    <header className="bg-white/80 backdrop-blur-md sticky top-0 z-40 border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4">
        <div className="flex items-center justify-between h-14">
          <div className="flex items-center gap-4">
            <div>
              <span className="text-lg font-semibold text-[#1D1D1F]">
                ApplePrice
              </span>
              <span className="ml-2 text-xs text-gray-500">
                官方翻新 · 固定85折
              </span>
            </div>

            {/* System Status Indicator */}
            {systemStatus && (
              <div className="flex items-center gap-2 text-xs cursor-default group relative">
                <div className={`w-2 h-2 rounded-full ${getStatusColor()}`} />
                <span className="text-gray-500">
                  {formatLastUpdate(systemStatus.last_scrape_time)}
                </span>
                {systemStatus.last_scrape_status === "failed" && (
                  <span className="text-red-500 max-w-32 truncate">
                    更新失败
                  </span>
                )}

                {/* Tooltip for error details */}
                {systemStatus.last_scrape_status === "failed" &&
                  systemStatus.last_scrape_error && (
                    <div className="absolute top-full left-0 mt-1 p-2 bg-gray-900 text-white text-xs rounded-lg shadow-lg opacity-0 group-hover:opacity-100 transition-opacity z-50 max-w-xs">
                      <div className="font-medium mb-1">错误信息:</div>
                      <div className="break-all">
                        {systemStatus.last_scrape_error}
                      </div>
                    </div>
                  )}
              </div>
            )}
          </div>

          <div className="flex items-center gap-4">
            {productCount !== undefined && (
              <div className="text-right">
                <span className="text-base font-semibold text-[#0071E3]">
                  {productCount}
                </span>
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
  );
}
