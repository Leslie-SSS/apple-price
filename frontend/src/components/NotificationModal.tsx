import { useState, useEffect } from "react";
import { CloseIcon, InfoIcon, EditIcon, PlayIcon, PauseIcon } from "./icons";
import { storage, maskBarkKey } from "../utils/storage";

interface NewArrivalSubscription {
  id: string;
  name: string;
  categories: string[];
  models?: string[];
  max_price: number;
  min_price: number;
  bark_key: string; // Masked in display, full key stored in localStorage
  enabled: boolean;
  paused: boolean;
  notification_count: number;
  created_at: string;
}

interface NotificationHistoryItem {
  id: string;
  subscription_id: string;
  product_id: string;
  product_name: string;
  product_category: string;
  product_price: number;
  product_image_url: string;
  product_specs: string;
  status: "sent" | "failed";
  error_message?: string;
  created_at: string;
}

interface NotificationModalProps {
  isOpen: boolean;
  onClose: () => void;
  categories: string[];
}

export default function NotificationModal({
  isOpen,
  onClose,
  categories,
}: NotificationModalProps) {
  const [subscriptions, setSubscriptions] = useState<NewArrivalSubscription[]>(
    [],
  );
  const [notificationHistory, setNotificationHistory] = useState<
    NotificationHistoryItem[]
  >([]);
  const [loading, setLoading] = useState(false);
  const [showBarkHelp, setShowBarkHelp] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);

  // Bark Key state - stored in localStorage, per user
  const [barkKey, setBarkKey] = useState("");
  const [showBarkKeyInput, setShowBarkKeyInput] = useState(false);
  const [barkKeyInput, setBarkKeyInput] = useState("");

  // Form state
  const [name, setName] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [selectedModels, setSelectedModels] = useState<string[]>([]);
  const [availableModels, setAvailableModels] = useState<string[]>([]);
  const [minPrice, setMinPrice] = useState("");
  const [maxPrice, setMaxPrice] = useState("");

  // Initialize Bark Key from localStorage
  useEffect(() => {
    if (isOpen) {
      const cachedKey = storage.getBarkKey();
      setBarkKey(cachedKey);
      setBarkKeyInput("");
      setShowBarkKeyInput(!cachedKey);
      fetchSubscriptions(cachedKey);
      fetchNotificationHistory(cachedKey);
    }
  }, [isOpen]);

  // When selected category changes, fetch available models
  useEffect(() => {
    if (selectedCategory && isOpen) {
      fetchFilterOptions();
    } else {
      setAvailableModels([]);
    }
  }, [selectedCategory, isOpen]);

  const fetchFilterOptions = async () => {
    try {
      const res = await fetch(
        `/api/filter-options?category=${encodeURIComponent(selectedCategory)}`,
      );
      const data = await res.json();
      setAvailableModels(data.models || []);
    } catch (error) {
      console.error("Failed to fetch filter options:", error);
      setAvailableModels([]);
    }
  };

  // Fetch subscriptions filtered by Bark Key
  const fetchSubscriptions = async (key: string) => {
    if (!key) {
      setSubscriptions([]);
      return;
    }
    try {
      const res = await fetch(
        `/api/new-arrival-subscriptions?bark_key=${encodeURIComponent(key)}`,
      );
      const data = await res.json();
      setSubscriptions(data.subscriptions || []);
    } catch (error) {
      console.error("Failed to fetch subscriptions:", error);
    }
  };

  const fetchNotificationHistory = async (key: string) => {
    if (!key) {
      setNotificationHistory([]);
      return;
    }
    try {
      const res = await fetch(
        `/api/notification-history?limit=20&bark_key=${encodeURIComponent(key)}`,
      );
      const data = await res.json();
      setNotificationHistory(data.data || []);
    } catch (error) {
      console.error("Failed to fetch notification history:", error);
    }
  };

  const resetForm = () => {
    setName("");
    setSelectedCategory("");
    setSelectedModels([]);
    setMinPrice("");
    setMaxPrice("");
    setEditingId(null);
  };

  const startEdit = (sub: NewArrivalSubscription) => {
    setEditingId(sub.id);
    setName(sub.name);
    setSelectedCategory(sub.categories?.[0] || "");
    setSelectedModels(sub.models || []);
    setMinPrice(sub.min_price > 0 ? String(sub.min_price) : "");
    setMaxPrice(sub.max_price > 0 ? String(sub.max_price) : "");
  };

  // Save Bark Key to localStorage
  const handleSaveBarkKey = () => {
    if (!barkKeyInput.trim()) return;

    const key = barkKeyInput.trim();
    storage.setBarkKey(key);
    setBarkKey(key);
    setShowBarkKeyInput(false);
    setBarkKeyInput("");

    // Fetch subscriptions and notification history for this Bark Key
    fetchSubscriptions(key);
    fetchNotificationHistory(key);
  };

  // Clear Bark Key from localStorage
  const handleClearBarkKey = () => {
    storage.clearBarkKey();
    setBarkKey("");
    setShowBarkKeyInput(true);
    setSubscriptions([]);
    setNotificationHistory([]);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Check if Bark Key is configured
    if (!barkKey) {
      alert("请先配置 Bark Key");
      return;
    }

    setLoading(true);

    try {
      const payload: Record<string, unknown> = {
        name,
        categories: selectedCategory ? [selectedCategory] : [],
        models: selectedModels,
        bark_key: barkKey, // Include Bark Key with subscription
        enabled: true,
      };

      if (minPrice) payload.min_price = parseFloat(minPrice);
      if (maxPrice) payload.max_price = parseFloat(maxPrice);

      const url = editingId
        ? `/api/new-arrival-subscriptions/${editingId}`
        : "/api/new-arrival-subscriptions";

      const method = editingId ? "PUT" : "POST";

      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        resetForm();
        fetchSubscriptions(barkKey);
      } else {
        const data = await res.json();
        alert(data.error || "保存失败");
      }
    } catch (error) {
      console.error("Failed to save subscription:", error);
      alert("保存失败，请重试");
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("确定要删除这个通知设置吗？")) return;
    try {
      await fetch(`/api/new-arrival-subscriptions/${id}`, { method: "DELETE" });
      fetchSubscriptions(barkKey);
      if (editingId === id) resetForm();
    } catch (error) {
      console.error("Failed to delete subscription:", error);
    }
  };

  const handlePauseResume = async (id: string, paused: boolean) => {
    try {
      const endpoint = paused ? "/pause" : "/resume";
      await fetch(`/api/new-arrival-subscriptions/${id}${endpoint}`, {
        method: "PATCH",
      });
      fetchSubscriptions(barkKey);
    } catch (error) {
      console.error("Failed to pause/resume subscription:", error);
    }
  };

  const toggleCategory = (cat: string) => {
    setSelectedCategory(selectedCategory === cat ? "" : cat);
    setSelectedModels([]);
  };

  const toggleModel = (model: string) => {
    setSelectedModels(
      selectedModels.includes(model)
        ? selectedModels.filter((m) => m !== model)
        : [...selectedModels, model],
    );
  };

  if (!isOpen) return null;

  return (
    <>
      <div
        className="fixed inset-0 bg-black/40 backdrop-blur-sm z-50"
        onClick={onClose}
      />
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div
          className="bg-white rounded-3xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-hidden animate-slideUp"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="sticky top-0 z-10 bg-white/95 backdrop-blur-sm border-b border-gray-100 px-6 py-4 flex justify-between items-center">
            <h2 className="text-xl font-semibold text-[#1D1D1F]">
              {editingId ? "编辑通知设置" : "上新通知设置"}
            </h2>
            <button
              onClick={() => {
                resetForm();
                onClose();
              }}
              className="w-9 h-9 flex items-center justify-center rounded-full hover:bg-gray-100 text-gray-500"
            >
              <CloseIcon className="w-5 h-5" />
            </button>
          </div>

          {/* Content */}
          <div className="p-6 overflow-y-auto max-h-[calc(90vh-140px)]">
            {/* Bark Key Configuration */}
            <div className="mb-6 p-4 bg-yellow-50 border border-yellow-200 rounded-2xl">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-yellow-500 rounded-full flex items-center justify-center flex-shrink-0">
                  <span className="text-white text-lg">🔔</span>
                </div>
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h4 className="text-sm font-semibold text-gray-800">
                      Bark 推送 Key
                    </h4>
                    {barkKey && (
                      <span className="text-xs text-green-600 bg-green-100 px-2 py-0.5 rounded-full">
                        ✓ 已配置
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-gray-500 mt-0.5">
                    {barkKey
                      ? `当前: ${maskBarkKey(barkKey)}`
                      : "配置后可接收新品上架通知"}
                  </p>
                </div>
                {barkKey && (
                  <button
                    type="button"
                    onClick={handleClearBarkKey}
                    className="text-xs text-red-500 hover:text-red-600"
                  >
                    清除
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setShowBarkKeyInput(!showBarkKeyInput)}
                  className="text-sm text-blue-600 hover:text-blue-700 font-medium"
                >
                  {barkKey ? "修改" : "配置"}
                </button>
              </div>

              {showBarkKeyInput && (
                <div className="mt-3 flex gap-2">
                  <input
                    type="password"
                    value={barkKeyInput}
                    onChange={(e) => setBarkKeyInput(e.target.value)}
                    placeholder="输入 Bark Key"
                    className="flex-1 px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  />
                  <button
                    type="button"
                    onClick={handleSaveBarkKey}
                    disabled={!barkKeyInput.trim()}
                    className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium disabled:opacity-50 hover:bg-blue-600 transition-colors"
                  >
                    保存
                  </button>
                </div>
              )}

              {/* Bark Help Link */}
              <button
                type="button"
                onClick={() => setShowBarkHelp(!showBarkHelp)}
                className="mt-2 text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
              >
                <InfoIcon className="w-3 h-3" />
                {showBarkHelp ? "收起使用指南" : "如何获取 Bark Key?"}
              </button>

              {showBarkHelp && (
                <div className="mt-3 p-3 bg-white rounded-lg text-xs text-gray-600 space-y-2">
                  <p>
                    <strong>步骤 1:</strong> 在 App Store 搜索并下载 "Bark" 应用
                  </p>
                  <p>
                    <strong>步骤 2:</strong> 打开 Bark 应用，首页会显示你的推送
                    Key
                  </p>
                  <p>
                    <strong>步骤 3:</strong> 复制推送 Key，粘贴到上方输入框中
                  </p>
                  <p>
                    <strong>步骤 4:</strong> 保存后即可创建订阅，接收新品通知
                  </p>
                  <p className="text-gray-400 pt-2 border-t border-gray-100">
                    开源项目:{" "}
                    <a
                      href="https://github.com/Finb/Bark"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-600 hover:underline"
                    >
                      github.com/Finb/Bark
                    </a>
                  </p>
                </div>
              )}
            </div>

            {/* Warning if Bark Key not configured */}
            {!barkKey && (
              <div className="mb-6 p-3 bg-red-50 border border-red-200 rounded-xl text-sm text-red-700">
                请先配置 Bark Key，否则无法接收通知
              </div>
            )}

            {/* Add/Edit Subscription Form */}
            <form
              onSubmit={handleSubmit}
              className="mb-6 p-4 bg-gray-50 rounded-2xl"
            >
              <div className="flex justify-between items-center mb-3">
                <h3 className="text-sm font-semibold text-[#1D1D1F]">
                  {editingId ? "编辑通知" : "添加新通知"}
                </h3>
                {editingId && (
                  <button
                    type="button"
                    onClick={resetForm}
                    className="text-xs text-gray-500 hover:text-gray-700"
                  >
                    取消编辑
                  </button>
                )}
              </div>

              {/* Name */}
              <div className="mb-3">
                <label className="block text-xs text-gray-500 mb-1">
                  名称 <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="例如: MacBook Pro M3 通知"
                  className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  required
                />
              </div>

              {/* Categories */}
              <div className="mb-3">
                <label className="block text-xs text-gray-500 mb-1">
                  产品类型 <span className="text-gray-400">(可选)</span>
                </label>
                <div className="flex flex-wrap gap-2">
                  {categories.map((cat) => (
                    <button
                      key={cat}
                      type="button"
                      onClick={() => toggleCategory(cat)}
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        selectedCategory === cat
                          ? "bg-[#0071E3] text-white"
                          : "bg-white border border-gray-200 text-[#1D1D1F] hover:bg-gray-50"
                      }`}
                    >
                      {cat}
                    </button>
                  ))}
                </div>
              </div>

              {/* Models - Show when category is selected */}
              {selectedCategory && availableModels.length > 0 && (
                <div className="mb-3 p-3 bg-white rounded-xl border border-gray-100">
                  <div className="flex items-center justify-between mb-2">
                    <label className="block text-xs text-gray-500">
                      产品型号{" "}
                      <span className="text-gray-400">(可选，多选)</span>
                    </label>
                    {selectedModels.length > 0 && (
                      <button
                        type="button"
                        onClick={() => setSelectedModels([])}
                        className="text-xs text-gray-400 hover:text-gray-600"
                      >
                        清除选择
                      </button>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {availableModels.map((model) => (
                      <button
                        key={model}
                        type="button"
                        onClick={() => toggleModel(model)}
                        className={`px-2 py-1 rounded text-xs font-medium transition-all ${
                          selectedModels.includes(model)
                            ? "bg-[#0071E3] text-white"
                            : "bg-white border border-gray-200 text-gray-700 hover:bg-gray-50"
                        }`}
                      >
                        {model}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Price Range */}
              <div className="mb-3 flex gap-3">
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">
                    最低价格 (可选)
                  </label>
                  <input
                    type="number"
                    value={minPrice}
                    onChange={(e) => setMinPrice(e.target.value)}
                    placeholder="0"
                    className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-gray-500 mb-1">
                    最高价格 (可选)
                  </label>
                  <input
                    type="number"
                    value={maxPrice}
                    onChange={(e) => setMaxPrice(e.target.value)}
                    placeholder="无限制"
                    className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
                  />
                </div>
              </div>

              <button
                type="submit"
                disabled={loading || !barkKey}
                className="w-full py-2.5 bg-[#0071E3] text-white rounded-xl font-semibold hover:bg-[#0077ED] transition-colors disabled:opacity-50"
              >
                {loading
                  ? "保存中..."
                  : editingId
                    ? "更新通知设置"
                    : "保存通知设置"}
              </button>
            </form>

            {/* Existing Subscriptions */}
            <div>
              <h3 className="text-sm font-semibold text-[#1D1D1F] mb-3">
                已设置的通知
              </h3>
              {!barkKey ? (
                <p className="text-sm text-gray-400 text-center py-4">
                  请先配置 Bark Key 查看您的订阅
                </p>
              ) : subscriptions.length === 0 ? (
                <p className="text-sm text-gray-400 text-center py-4">
                  暂无通知设置
                </p>
              ) : (
                <div className="space-y-2">
                  {subscriptions.map((sub) => (
                    <div
                      key={sub.id}
                      className={`p-3 rounded-xl ${sub.paused ? "bg-gray-100 opacity-75" : "bg-gray-50"}`}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-sm font-medium text-[#1D1D1F]">
                              {sub.name}
                            </span>
                            {sub.paused && (
                              <span className="px-1.5 py-0.5 rounded text-[10px] bg-orange-100 text-orange-700">
                                已暂停
                              </span>
                            )}
                            {!sub.enabled && (
                              <span className="px-1.5 py-0.5 rounded text-[10px] bg-gray-200 text-gray-500">
                                禁用
                              </span>
                            )}
                            {sub.notification_count > 0 && (
                              <span className="text-xs text-gray-400">
                                已通知 {sub.notification_count} 次
                              </span>
                            )}
                          </div>
                          <div className="text-xs text-gray-500 mt-1">
                            {sub.categories.length > 0
                              ? sub.categories.join(", ")
                              : "全部分类"}
                            {sub.models &&
                              sub.models.length > 0 &&
                              ` · 型号: ${sub.models.join(", ")}`}
                            {sub.min_price > 0 && ` · ¥${sub.min_price}+`}
                            {sub.max_price > 0 && ` · ¥${sub.max_price}-`}
                          </div>
                        </div>
                        <div className="flex items-center gap-1">
                          <button
                            onClick={() =>
                              handlePauseResume(sub.id, !sub.paused)
                            }
                            className="p-2 text-gray-400 hover:text-blue-500 hover:bg-blue-50 rounded-lg transition-colors"
                            title={sub.paused ? "恢复通知" : "暂停通知"}
                          >
                            {sub.paused ? (
                              <PlayIcon className="w-4 h-4" />
                            ) : (
                              <PauseIcon className="w-4 h-4" />
                            )}
                          </button>
                          <button
                            onClick={() => startEdit(sub)}
                            className="p-2 text-gray-400 hover:text-blue-500 hover:bg-blue-50 rounded-lg transition-colors"
                            title="编辑"
                          >
                            <EditIcon className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleDelete(sub.id)}
                            className="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                            title="删除"
                          >
                            ×
                          </button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Notification History */}
            <div className="mt-6 border-t border-gray-100 pt-4">
              <h3 className="text-sm font-semibold text-[#1D1D1F] mb-3">
                通知历史
              </h3>
              {notificationHistory.length === 0 ? (
                <p className="text-sm text-gray-400 text-center py-4">
                  暂无通知记录
                </p>
              ) : (
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {notificationHistory.map((h) => (
                    <div key={h.id} className="p-3 bg-gray-50 rounded-xl">
                      <div className="flex items-start gap-3">
                        {h.product_image_url && (
                          <img
                            src={h.product_image_url}
                            alt={h.product_name}
                            className="w-10 h-10 object-contain rounded-lg bg-white"
                          />
                        )}
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-sm font-medium text-[#1D1D1F] truncate">
                              {h.product_name}
                            </span>
                            <span
                              className={`text-xs px-1.5 py-0.5 rounded ${
                                h.status === "sent"
                                  ? "bg-green-100 text-green-700"
                                  : "bg-red-100 text-red-700"
                              }`}
                            >
                              {h.status === "sent" ? "已发送" : "失败"}
                            </span>
                          </div>
                          <div className="text-xs text-gray-500 mt-1">
                            ¥{h.product_price.toLocaleString()} ·{" "}
                            {h.product_category}
                          </div>
                          <div className="text-xs text-gray-400 mt-1">
                            {new Date(h.created_at).toLocaleString("zh-CN")}
                          </div>
                          {h.error_message && (
                            <div className="text-xs text-red-500 mt-1 bg-red-50 p-2 rounded">
                              {h.error_message}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
