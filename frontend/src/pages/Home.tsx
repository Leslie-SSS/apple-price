import { useState, useMemo, useEffect } from "react";
import { useProducts } from "@/hooks/useProducts";
import ProductCard from "@/components/ProductCard";
import { SearchIcon, CloseIcon } from "@/components/icons";

// 从 description 中提取规格信息 - 需要精确区分内存和存储
function extractSpecsFromDescription(
  description: string,
): Record<string, string> {
  const specs: Record<string, string> = {};

  // 提取内存 - 只匹配明确标注为"内存"的
  const memPatterns = [
    /(\d+)\s*GB\s*统一[\s\xa0]*记忆(?:体|系统)?/, // 统一记忆/统一内存
    /(\d+)\s*GB\s*统一[\s\xa0]*内存/,
    /(\d+)\s*GB\s*unified[\s\xa0]*memory/i,
    /(\d+)\s*GB\s*内存(?!\s*和)/, // 内存后面不跟"和"（避免匹配"内存和存储"）
    /(\d+)\s*GB[\s\xa0]*LPDDR[X\d]?/, // LPDDR类型的是内存
    /(\d+)\s*GB[\s\xa0]*HBM/, // HBM是内存
  ];
  for (const pattern of memPatterns) {
    const match = description.match(pattern);
    if (match) {
      specs.memory = match[1] + "GB";
      break;
    }
  }

  // 提取存储 - 只匹配明确标注为"存储/硬盘/SSD"的
  const storagePatterns = [
    /(\d+)\s*(TB|GB)\s*固态[\s\xa0]*硬盘/,
    /(\d+)\s*(TB|GB)\s*SSD/i,
    /(\d+)\s*(TB|GB)\s*存储/,
    /(\d+)\s*(TB|GB)\s*硬盘/,
    /(\d+)\s*(TB|GB)\s*Flash[\s\xa0]*storage/i,
  ];
  for (const pattern of storagePatterns) {
    const match = description.match(pattern);
    if (match) {
      specs.storage = match[1] + match[2];
      break;
    }
  }

  // 提取屏幕尺寸
  const screenMatch = description.match(/(\d+(?:\.\d+)?)["\s]*英寸/);
  if (screenMatch) {
    specs.screen_size = screenMatch[1] + '"';
  }

  // 提取颜色
  const colorPatterns = [
    /深空黑色/,
    /深空黑/,
    /深空灰/,
    /太空灰/,
    /银色/,
    /金色/,
    /星光色/,
    /午夜色/,
    /午夜/,
    /深空黑色/,
    /蓝色/,
    /紫色/,
    /绿色/,
    /粉色/,
    /橙色/,
    /黄色/,
    /红色/,
    /黑色/,
    /白色/,
    /玫瑰金/,
  ];
  for (const pattern of colorPatterns) {
    const match = description.match(pattern);
    if (match) {
      specs.color = match[0];
      break;
    }
  }

  return specs;
}

// 获取产品的规格信息（用于筛选）- 精确匹配
function getProductSpecs(product: any): {
  memory?: string;
  storage?: string;
  screen_size?: string;
  color?: string;
} {
  // 先从 specs_detail 解析
  let specs: Record<string, string> = {};
  if (product.specs_detail && typeof product.specs_detail === "object") {
    specs = { ...product.specs_detail };
  }

  // 再从 description 提取（优先级更高，因为更详细）
  const descSpecs = product.description
    ? extractSpecsFromDescription(product.description)
    : {};

  return {
    memory: descSpecs.memory || specs.memory,
    storage: descSpecs.storage || specs.storage,
    screen_size: descSpecs.screen_size || specs.screen_size,
    color: descSpecs.color || specs.color,
  };
}

// 判断是否为 Mac 产品（需要内存筛选）
function isMacProduct(category: string): boolean {
  return category === "Mac";
}

// 判断是否为 iPad（需要存储和颜色筛选，不需要内存）
function isIPad(category: string): boolean {
  return category === "iPad";
}

// 判断是否为 Watch（需要尺寸和颜色筛选）
function isWatch(category: string): boolean {
  return category === "Watch";
}

// 判断是否为 iPhone（需要存储和颜色筛选）
function isIPhone(category: string): boolean {
  return category === "iPhone";
}

interface HomeProps {
  onFilteredCountChange?: (count: number) => void;
  onCategoriesChange?: (categories: string[]) => void;
}

// Filter options from API
interface FilterOptions {
  chips: string[];
  storages: string[];
  memories: string[];
  screen_sizes: string[];
  colors: string[];
  models: string[];
}

// Mac sub-categories
const MAC_SUBCATEGORIES = [
  "MacBook Air",
  "MacBook Pro",
  "Mac mini",
  "Mac Studio",
  "iMac",
  "Mac Pro",
];

// 价格预设
const PRICE_PRESETS = [
  { label: "全部", min: 0, max: Infinity },
  { label: "¥3000以下", min: 0, max: 3000 },
  { label: "¥3000-6000", min: 3000, max: 6000 },
  { label: "¥6000-10000", min: 6000, max: 10000 },
  { label: "¥10000-15000", min: 10000, max: 15000 },
  { label: "¥15000-20000", min: 15000, max: 20000 },
  { label: "¥20000以上", min: 20000, max: Infinity },
];

// 排序选项
const SORT_OPTIONS = [
  { label: "默认", value: "default" },
  { label: "价格低到高", value: "price_asc" },
  { label: "价格高到低", value: "price_desc" },
  { label: "最新上架", value: "newest" },
];

export default function Home({
  onFilteredCountChange,
  onCategoriesChange,
}: HomeProps) {
  const [categoryFilter, setCategoryFilter] = useState<string>("全部");
  const [macSubCategoryFilter, setMacSubCategoryFilter] =
    useState<string>("全部");
  const [chipFilter, setChipFilter] = useState<string>("全部");
  const [pricePreset, setPricePreset] = useState<number>(0);
  const [storageFilter, setStorageFilter] = useState<string>("全部");
  const [memoryFilter, setMemoryFilter] = useState<string>("全部");
  const [screenSizeFilter, setScreenSizeFilter] = useState<string>("全部");
  const [colorFilter, setColorFilter] = useState<string>("全部");
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [sortBy, setSortBy] = useState<string>("default");
  const [filterOptions, setFilterOptions] = useState<FilterOptions>({
    chips: [],
    storages: [],
    memories: [],
    screen_sizes: [],
    colors: [],
    models: [],
  });

  const { products, loading } = useProducts({
    category: "",
    sort: "score",
    order: "desc",
  });

  // Fetch filter options when category changes
  useEffect(() => {
    const fetchFilterOptions = async () => {
      try {
        const params = new URLSearchParams();
        if (categoryFilter !== "全部") {
          params.set("category", categoryFilter);
        }
        const res = await fetch(`/api/filter-options?${params}`);
        const data: FilterOptions = await res.json();
        setFilterOptions(data);
      } catch (err) {
        console.error("Failed to fetch filter options:", err);
      }
    };
    fetchFilterOptions();
  }, [categoryFilter]);

  // 获取分类列表和每个分类的代表图片
  const categoryInfo = useMemo(() => {
    if (!products) return [];
    const catMap = new Map<
      string,
      { name: string; image: string; count: number }
    >();

    for (const p of products) {
      const existing = catMap.get(p.category);
      if (!existing) {
        catMap.set(p.category, {
          name: p.category,
          image: p.image_url || "",
          count: 1,
        });
      } else {
        existing.count++;
      }
    }

    return Array.from(catMap.values()).sort((a, b) => b.count - a.count);
  }, [products]);

  const categories = useMemo(() => {
    return ["全部", ...categoryInfo.map((c) => c.name)];
  }, [categoryInfo]);

  // 根据当前分类和动态筛选选项确定应该显示哪些筛选器
  const filterConfig = useMemo(() => {
    const hasChips = filterOptions.chips.length > 0;
    const hasStorages = filterOptions.storages.length > 0;
    const hasMemories = filterOptions.memories.length > 0;
    const hasScreenSizes = filterOptions.screen_sizes.length > 0;
    const hasColors = filterOptions.colors.length > 0;
    const hasModels = filterOptions.models.length > 0;

    if (categoryFilter === "全部") {
      return {
        showChip: hasChips,
        showMemory: hasMemories,
        showStorage: hasStorages,
        showScreen: hasScreenSizes,
        showColor: hasColors,
        showMacSubCategory: false,
        chipOptions: ["全部", ...filterOptions.chips],
        storageOptions: ["全部", ...filterOptions.storages],
        memoryOptions: ["全部", ...filterOptions.memories],
        screenSizeOptions: ["全部", ...filterOptions.screen_sizes],
        colorOptions: ["全部", ...filterOptions.colors],
      };
    }

    // Mac - 显示芯片、内存、存储、屏幕、颜色、子分类
    if (isMacProduct(categoryFilter)) {
      return {
        showChip: hasChips,
        showMemory: hasMemories,
        showStorage: hasStorages,
        showScreen: hasScreenSizes,
        showColor: hasColors,
        showMacSubCategory: hasModels,
        chipOptions: ["全部", ...filterOptions.chips],
        storageOptions: ["全部", ...filterOptions.storages],
        memoryOptions: ["全部", ...filterOptions.memories],
        screenSizeOptions: ["全部", ...filterOptions.screen_sizes],
        colorOptions: ["全部", ...filterOptions.colors],
        macSubCategoryOptions: [
          "全部",
          ...filterOptions.models.filter((m) => MAC_SUBCATEGORIES.includes(m)),
        ],
      };
    }

    // iPad - 显示芯片、存储、屏幕、颜色（不显示内存）
    if (isIPad(categoryFilter)) {
      return {
        showChip: hasChips,
        showMemory: false,
        showStorage: hasStorages,
        showScreen: hasScreenSizes,
        showColor: hasColors,
        showMacSubCategory: false,
        chipOptions: ["全部", ...filterOptions.chips],
        storageOptions: ["全部", ...filterOptions.storages],
        memoryOptions: [],
        screenSizeOptions: ["全部", ...filterOptions.screen_sizes],
        colorOptions: ["全部", ...filterOptions.colors],
      };
    }

    // iPhone - 显示芯片、存储、屏幕、颜色（不显示内存）
    if (isIPhone(categoryFilter)) {
      return {
        showChip: hasChips,
        showMemory: false,
        showStorage: hasStorages,
        showScreen: hasScreenSizes,
        showColor: hasColors,
        showMacSubCategory: false,
        chipOptions: ["全部", ...filterOptions.chips],
        storageOptions: ["全部", ...filterOptions.storages],
        memoryOptions: [],
        screenSizeOptions: ["全部", ...filterOptions.screen_sizes],
        colorOptions: ["全部", ...filterOptions.colors],
      };
    }

    // Watch - 只显示屏幕尺寸、颜色
    if (isWatch(categoryFilter)) {
      return {
        showChip: false,
        showMemory: false,
        showStorage: false,
        showScreen: hasScreenSizes,
        showColor: hasColors,
        showMacSubCategory: false,
        chipOptions: [],
        storageOptions: [],
        memoryOptions: [],
        screenSizeOptions: ["全部", ...filterOptions.screen_sizes],
        colorOptions: ["全部", ...filterOptions.colors],
      };
    }

    // 默认配置（配件等）- 只显示颜色
    return {
      showChip: false,
      showMemory: false,
      showStorage: false,
      showScreen: false,
      showColor: hasColors,
      showMacSubCategory: false,
      chipOptions: [],
      storageOptions: [],
      memoryOptions: [],
      screenSizeOptions: [],
      colorOptions: ["全部", ...filterOptions.colors],
    };
  }, [categoryFilter, filterOptions]);

  const filteredProducts = useMemo(() => {
    if (!products) return [];
    let result = [...products];

    // 分类筛选
    if (categoryFilter !== "全部") {
      result = result.filter((p) => p.category === categoryFilter);
    }

    // Mac 子分类筛选
    if (categoryFilter === "Mac" && macSubCategoryFilter !== "全部") {
      result = result.filter((p) => {
        const nameLower = p.name.toLowerCase();
        switch (macSubCategoryFilter) {
          case "MacBook Air":
            return nameLower.includes("macbook air");
          case "MacBook Pro":
            return nameLower.includes("macbook pro");
          case "Mac mini":
            return nameLower.includes("mac mini");
          case "Mac Studio":
            return nameLower.includes("mac studio");
          case "iMac":
            return nameLower.includes("imac");
          case "Mac Pro":
            return (
              nameLower.includes("mac pro") && !nameLower.includes("macbook")
            );
          default:
            return true;
        }
      });
    }

    // 芯片筛选
    if (filterConfig.showChip && chipFilter !== "全部") {
      result = result.filter((p) => {
        const name = p.name.toLowerCase();
        const chipLower = chipFilter.toLowerCase();
        // 精确匹配芯片型号
        if (
          chipFilter === "M1 Pro" ||
          chipFilter === "M1 Max" ||
          chipFilter === "M1 Ultra" ||
          chipFilter === "M2 Pro" ||
          chipFilter === "M2 Max" ||
          chipFilter === "M2 Ultra" ||
          chipFilter === "M3 Pro" ||
          chipFilter === "M3 Max" ||
          chipFilter === "M4 Pro" ||
          chipFilter === "M4 Max"
        ) {
          return name.includes(chipLower.toLowerCase());
        }
        return (
          name.includes(chipLower) &&
          !name.includes("pro") &&
          !name.includes("max") &&
          !name.includes("ultra")
        );
      });
    }

    // 价格筛选
    const preset = PRICE_PRESETS[pricePreset];
    if (preset.max !== Infinity) {
      result = result.filter(
        (p) => p.price >= preset.min && p.price <= preset.max,
      );
    } else {
      result = result.filter((p) => p.price >= preset.min);
    }

    // 存储筛选 - 只检查明确的存储字段
    if (filterConfig.showStorage && storageFilter !== "全部") {
      result = result.filter((p) => {
        const specs = getProductSpecs(p);
        // 只匹配存储字段，不匹配内存
        if (specs.storage === storageFilter) return true;
        // 检查名称中是否包含存储规格（在产品名称中）
        const namePattern = new RegExp(
          `${storageFilter.replace("GB", "(GB|TB)")}\\s*(存储|SSD|硬盘)`,
        );
        return (
          namePattern.test(p.name) ||
          (p.description && p.description.includes(storageFilter + "固态"))
        );
      });
    }

    // 内存筛选 - 只检查明确的内存字段
    if (filterConfig.showMemory && memoryFilter !== "全部") {
      result = result.filter((p) => {
        const specs = getProductSpecs(p);
        // 只匹配内存字段，不匹配存储
        if (specs.memory === memoryFilter) return true;
        // 检查名称中是否包含内存规格
        const namePattern = new RegExp(`${memoryFilter}\\s*(GB|内存|统一)`);
        return (
          namePattern.test(p.name) ||
          (p.description &&
            p.description.includes(memoryFilter) &&
            p.description.includes("统一"))
        );
      });
    }

    // 屏幕尺寸筛选
    if (filterConfig.showScreen && screenSizeFilter !== "全部") {
      result = result.filter((p) => {
        const specs = getProductSpecs(p);
        return (
          specs.screen_size === screenSizeFilter ||
          p.name.includes(screenSizeFilter) ||
          p.name.includes(screenSizeFilter.replace('"', "英寸"))
        );
      });
    }

    // 颜色筛选
    if (filterConfig.showColor && colorFilter !== "全部") {
      result = result.filter((p) => {
        const specs = getProductSpecs(p);
        if (specs.color === colorFilter) return true;
        // 检查名称中的颜色
        const colorVariants = {
          深空黑: ["深空黑", "深空黑色"],
          深空灰: ["深空灰", "太空灰"],
          银色: ["银色", "银"],
          金色: ["金色", "金"],
          星光色: ["星光色", "星光"],
          午夜色: ["午夜色", "午夜"],
          蓝色: ["蓝色", "蓝"],
          紫色: ["紫色", "紫"],
          绿色: ["绿色", "绿"],
          红色: ["红色", "红", "(PRODUCT)RED"],
        };
        const variants = colorVariants[
          colorFilter as keyof typeof colorVariants
        ] || [colorFilter];
        return variants.some(
          (v) =>
            p.name.includes(v) || (p.description && p.description.includes(v)),
        );
      });
    }

    // 搜索筛选
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (p) =>
          p.name.toLowerCase().includes(query) ||
          p.category.toLowerCase().includes(query) ||
          (p.description && p.description.toLowerCase().includes(query)),
      );
    }

    // 排序
    switch (sortBy) {
      case "price_asc":
        result.sort((a, b) => a.price - b.price);
        break;
      case "price_desc":
        result.sort((a, b) => b.price - a.price);
        break;
      case "newest":
        // 暂时使用 ID 作为排序依据
        result.sort((a, b) => b.id.localeCompare(a.id));
        break;
      default:
        break;
    }

    return result;
  }, [
    products,
    categoryFilter,
    macSubCategoryFilter,
    chipFilter,
    pricePreset,
    storageFilter,
    memoryFilter,
    screenSizeFilter,
    colorFilter,
    searchQuery,
    sortBy,
    filterConfig,
  ]);

  // 通知父组件筛选结果数量
  useEffect(() => {
    onFilteredCountChange?.(filteredProducts.length);
  }, [filteredProducts.length, onFilteredCountChange]);

  // 通知父组件分类列表
  useEffect(() => {
    onCategoriesChange?.(categories.filter((c) => c !== "全部"));
  }, [categories, onCategoriesChange]);

  // 重置筛选
  const resetFilters = () => {
    setCategoryFilter("全部");
    setMacSubCategoryFilter("全部");
    setChipFilter("全部");
    setPricePreset(0);
    setStorageFilter("全部");
    setMemoryFilter("全部");
    setScreenSizeFilter("全部");
    setColorFilter("全部");
    setSearchQuery("");
    setSortBy("default");
  };

  // 检查是否有活动筛选
  const hasActiveFilters =
    categoryFilter !== "全部" ||
    macSubCategoryFilter !== "全部" ||
    chipFilter !== "全部" ||
    pricePreset !== 0 ||
    storageFilter !== "全部" ||
    memoryFilter !== "全部" ||
    screenSizeFilter !== "全部" ||
    colorFilter !== "全部" ||
    !!searchQuery;

  // 活动筛选数量
  const activeFilterCount = [
    categoryFilter,
    chipFilter,
    pricePreset,
    storageFilter,
    memoryFilter,
    screenSizeFilter,
    colorFilter,
    searchQuery,
  ].filter((v) => v !== "全部" && v !== 0 && v !== "").length;

  if (loading) {
    return (
      <div className="min-h-screen bg-[#F5F5F7] flex items-center justify-center">
        <div className="text-center">
          <div className="w-8 h-8 border-3 border-[#0071E3] border-t-transparent rounded-full animate-spin mx-auto mb-3" />
          <p className="text-sm text-gray-500">加载中...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-[#F5F5F7]">
      <div className="max-w-7xl mx-auto px-4 py-4">
        {/* 搜索框 */}
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
                onClick={() => setSearchQuery("")}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 p-1"
              >
                <CloseIcon />
              </button>
            )}
          </div>
        </div>

        {/* 产品分类 - 使用真实产品图片 */}
        <div className="mb-4">
          <div className="text-xs text-gray-500 mb-2">产品分类</div>
          <div className="grid grid-cols-4 sm:grid-cols-6 md:grid-cols-9 gap-2">
            {categoryInfo.map((cat) => {
              const isActive = categoryFilter === cat.name;
              return (
                <button
                  key={cat.name}
                  onClick={() =>
                    setCategoryFilter(
                      cat.name === categoryFilter ? "全部" : cat.name,
                    )
                  }
                  className={`flex flex-col items-center gap-1.5 p-3 rounded-xl transition-all bg-white ${
                    isActive
                      ? "border-2 border-[#0071E3] text-[#0071E3] shadow-md"
                      : "border border-gray-200 text-[#1D1D1F] hover:border-gray-400"
                  }`}
                >
                  <div className="w-10 h-10 flex items-center justify-center">
                    {cat.image ? (
                      <img
                        src={cat.image}
                        alt={cat.name}
                        className="w-full h-full object-contain"
                        loading="lazy"
                      />
                    ) : (
                      <span className="text-2xl">📱</span>
                    )}
                  </div>
                  <span className="text-[10px] font-medium truncate w-full text-center">
                    {cat.name}
                  </span>
                  <span
                    className={`text-[9px] ${isActive ? "text-[#0071E3]/70" : "text-gray-400"}`}
                  >
                    {cat.count}款
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* 筛选区域 - 全部显示，根据分类智能调整 */}
        <div className="mb-4 p-4 bg-white rounded-2xl border border-gray-200">
          {/* 价格筛选 - 始终显示 */}
          <div className="mb-4">
            <div className="text-xs text-gray-500 mb-2">价格区间</div>
            <div className="flex flex-wrap gap-2">
              {PRICE_PRESETS.map((preset, index) => (
                <button
                  key={preset.label}
                  onClick={() =>
                    setPricePreset(index === pricePreset ? 0 : index)
                  }
                  className={`px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-all ${
                    pricePreset === index
                      ? "bg-[#0071E3] text-white"
                      : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                  }`}
                >
                  {preset.label}
                </button>
              ))}
            </div>
          </div>

          {/* Mac 子分类筛选 */}
          {filterConfig.showMacSubCategory && (
            <div className="mb-4">
              <div className="text-xs text-gray-500 mb-2">Mac 类型</div>
              <div className="flex flex-wrap gap-2">
                {(filterConfig.macSubCategoryOptions || []).map(
                  (sub: string) => (
                    <button
                      key={sub}
                      onClick={() =>
                        setMacSubCategoryFilter(
                          sub === macSubCategoryFilter ? "全部" : sub,
                        )
                      }
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        macSubCategoryFilter === sub
                          ? "bg-[#0071E3] text-white"
                          : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                      }`}
                    >
                      {sub}
                    </button>
                  ),
                )}
              </div>
            </div>
          )}

          {/* 芯片筛选 - 只在显示时 */}
          {filterConfig.showChip && filterConfig.chipOptions.length > 1 && (
            <div className="mb-4">
              <div className="text-xs text-gray-500 mb-2">芯片型号</div>
              <div className="flex flex-wrap gap-2">
                {filterConfig.chipOptions.map((chip: string) => (
                  <button
                    key={chip}
                    onClick={() =>
                      setChipFilter(chip === chipFilter ? "全部" : chip)
                    }
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      chipFilter === chip
                        ? "bg-[#0071E3] text-white"
                        : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                    }`}
                  >
                    {chip}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* 存储筛选 - 只在显示时 */}
          {filterConfig.showStorage && (
            <div className="mb-4">
              <div className="text-xs text-gray-500 mb-2">存储容量</div>
              <div className="flex flex-wrap gap-2">
                {filterConfig.storageOptions.map((storage: string) => (
                  <button
                    key={storage}
                    onClick={() =>
                      setStorageFilter(
                        storage === storageFilter ? "全部" : storage,
                      )
                    }
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      storageFilter === storage
                        ? "bg-[#0071E3] text-white"
                        : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                    }`}
                  >
                    {storage}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* 内存筛选 - 只在显示时 */}
          {filterConfig.showMemory && (
            <div className="mb-4">
              <div className="text-xs text-gray-500 mb-2">内存大小 (RAM)</div>
              <div className="flex flex-wrap gap-2">
                {filterConfig.memoryOptions.map((memory: string) => (
                  <button
                    key={memory}
                    onClick={() =>
                      setMemoryFilter(memory === memoryFilter ? "全部" : memory)
                    }
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      memoryFilter === memory
                        ? "bg-[#0071E3] text-white"
                        : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                    }`}
                  >
                    {memory}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* 屏幕尺寸筛选 - 只在显示时 */}
          {filterConfig.showScreen &&
            filterConfig.screenSizeOptions.length > 1 && (
              <div className="mb-4">
                <div className="text-xs text-gray-500 mb-2">屏幕尺寸</div>
                <div className="flex flex-wrap gap-2">
                  {filterConfig.screenSizeOptions.map((size: string) => (
                    <button
                      key={size}
                      onClick={() =>
                        setScreenSizeFilter(
                          size === screenSizeFilter ? "全部" : size,
                        )
                      }
                      className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        screenSizeFilter === size
                          ? "bg-[#0071E3] text-white"
                          : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                      }`}
                    >
                      {size}
                    </button>
                  ))}
                </div>
              </div>
            )}

          {/* 颜色筛选 - 只在有选项时显示 */}
          {filterConfig.showColor && filterConfig.colorOptions.length > 1 && (
            <div>
              <div className="text-xs text-gray-500 mb-2">颜色</div>
              <div className="flex flex-wrap gap-2">
                {filterConfig.colorOptions.map((color: string) => (
                  <button
                    key={color}
                    onClick={() =>
                      setColorFilter(color === colorFilter ? "全部" : color)
                    }
                    className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all ${
                      colorFilter === color
                        ? "bg-[#0071E3] text-white"
                        : "bg-gray-100 text-[#1D1D1F] hover:bg-gray-200"
                    }`}
                  >
                    {color}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* 清除筛选 */}
          {hasActiveFilters && (
            <div className="mt-4 pt-4 border-t border-gray-100">
              <button
                onClick={resetFilters}
                className="w-full py-2.5 bg-gray-100 hover:bg-gray-200 rounded-lg text-sm text-[#1D1D1F] font-medium transition-colors"
              >
                清除所有筛选 ({activeFilterCount})
              </button>
            </div>
          )}
        </div>

        {/* 排序和结果数量 */}
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3 flex-wrap">
            <span className="text-sm text-gray-600">
              找到{" "}
              <span className="font-semibold text-[#0071E3]">
                {filteredProducts.length}
              </span>{" "}
              款产品
            </span>
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
              className="text-xs bg-white border border-gray-200 rounded-lg px-3 py-1.5 text-[#1D1D1F] focus:outline-none focus:ring-2 focus:ring-[#0071E3]"
            >
              {SORT_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
          {/* 活动筛选摘要 */}
          {hasActiveFilters && (
            <div className="flex items-center gap-1 text-xs text-gray-500 flex-wrap">
              <span>已筛选:</span>
              {categoryFilter !== "全部" && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {categoryFilter}
                </span>
              )}
              {chipFilter !== "全部" && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {chipFilter}
                </span>
              )}
              {storageFilter !== "全部" && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {storageFilter}
                </span>
              )}
              {memoryFilter !== "全部" && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {memoryFilter}
                </span>
              )}
              {screenSizeFilter !== "全部" && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {screenSizeFilter}
                </span>
              )}
              {colorFilter !== "全部" && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {colorFilter}
                </span>
              )}
              {pricePreset !== 0 && (
                <span className="px-2 py-0.5 bg-blue-50 text-[#0071E3] rounded">
                  {PRICE_PRESETS[pricePreset].label}
                </span>
              )}
            </div>
          )}
        </div>

        {/* 产品列表 */}
        {filteredProducts.length > 0 ? (
          <div className="space-y-3">
            {filteredProducts.map((product) => (
              <ProductCard key={product.id} product={product} />
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
                清除所有筛选
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
