import { Product } from "@/services/api";
import { parseSpecs } from "@/utils/product";

interface ProductCardProps {
  product: Product;
}

// 从 description 中提取规格信息
function extractSpecsFromDescription(
  description: string,
): Record<string, string> {
  const specs: Record<string, string> = {};

  // 提取内存 (统一内存) - 多种模式
  const memPatterns = [
    /(\d+)\s*GB\s*统一[\s\xa0]*内存/,
    /(\d+)\s*GB\s*内存/,
    /(\d+)\s*GB\s*unified[\s\xa0]*memory/i,
    /(\d+)\s*GB\s*memory/i,
    /(\d+)\s*GB\s*RAM/i,
  ];
  for (const pattern of memPatterns) {
    const match = description.match(pattern);
    if (match) {
      specs.memory = match[1] + "GB";
      break;
    }
  }

  // 提取存储 (固态硬盘) - 多种模式
  const storagePatterns = [
    /(\d+)\s*(TB|GB)\s*固态[\s\xa0]*硬盘/,
    /(\d+)\s*(TB|GB)\s*SSD/i,
    /(\d+)\s*(TB|GB)\s*storage/i,
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
    specs.screen_size = screenMatch[1] + "英寸";
  }

  // 提取摄像头
  const cameraPatterns = [
    /(\d+)\s*MP\s*Center Stage/,
    /(\d+)\s*MP/,
    /(\d+)\s*万像素/,
  ];
  for (const pattern of cameraPatterns) {
    const match = description.match(pattern);
    if (match) {
      specs.camera = match[1] + "MP";
      break;
    }
  }

  // 提取触控ID
  if (description.includes("触控 ID") || description.includes("Touch ID")) {
    specs.touch_id = "触控ID";
  }

  // 提取面容ID
  if (description.includes("面容 ID") || description.includes("Face ID")) {
    specs.face_id = "面容ID";
  }

  // 提取端口信息
  if (description.includes("雷霆 5") || description.includes("雷雳 5")) {
    specs.ports = "雷雳 5";
  } else if (description.includes("雷霆 4") || description.includes("雷雳 4")) {
    specs.ports = "雷雳 4";
  } else if (description.includes("Thunderbolt")) {
    specs.ports = "Thunderbolt";
  }

  return specs;
}

export default function ProductCard({ product }: ProductCardProps) {
  const specs = parseSpecs(product.specs_detail);

  // 如果 specs_detail 为空或信息不全，尝试从 description 中提取
  const descSpecs = product.description
    ? extractSpecsFromDescription(product.description)
    : {};

  // 合并规格信息 - 优先使用 description 提取的值（更详细）
  const allSpecs: Record<string, string> = { ...specs, ...descSpecs };

  // 构建完整规格显示数组 - 按优先级排序
  const specItems: { label: string; value: string }[] = [];

  // 芯片
  if (allSpecs.chip) {
    const chipValue = allSpecs.chip;
    let cpuInfo = "";
    if (allSpecs.cpu_cores) cpuInfo += `${allSpecs.cpu_cores}核CPU`;
    if (allSpecs.gpu_cores)
      cpuInfo += (cpuInfo ? "/" : "") + `${allSpecs.gpu_cores}核GPU`;
    specItems.push({
      label: "芯片",
      value: cpuInfo ? `${chipValue} (${cpuInfo})` : chipValue,
    });
  }

  // 内存
  if (allSpecs.memory) {
    specItems.push({ label: "内存", value: allSpecs.memory });
  }

  // 存储
  if (allSpecs.storage) {
    specItems.push({ label: "存储", value: allSpecs.storage });
  }

  // 屏幕
  if (allSpecs.screen_size) {
    specItems.push({ label: "屏幕", value: allSpecs.screen_size });
  }

  // 网络类型
  if (allSpecs.connectivity) {
    specItems.push({ label: "网络", value: allSpecs.connectivity });
  }

  // 颜色
  if (allSpecs.color) {
    specItems.push({ label: "颜色", value: allSpecs.color });
  }

  // 显示类型
  if (allSpecs.display_type) {
    specItems.push({ label: "玻璃", value: allSpecs.display_type });
  }

  // 支架类型
  if (allSpecs.stand_type) {
    specItems.push({ label: "支架", value: allSpecs.stand_type });
  }

  // 表壳尺寸
  if (allSpecs.case_size) {
    specItems.push({ label: "表壳", value: allSpecs.case_size });
  }

  // 表带类型
  if (allSpecs.band_type) {
    specItems.push({ label: "表带", value: allSpecs.band_type });
  }

  // 千兆以太网
  if (allSpecs.ethernet) {
    specItems.push({ label: "网口", value: "千兆" });
  }

  // 端口
  if (allSpecs.ports) {
    specItems.push({ label: "接口", value: allSpecs.ports });
  }

  // 型号
  if (allSpecs.model) {
    specItems.push({ label: "型号", value: allSpecs.model });
  }

  // 摄像头
  if (allSpecs.camera) {
    specItems.push({ label: "摄像头", value: allSpecs.camera });
  }

  // 触控ID/面容ID
  if (allSpecs.touch_id) {
    specItems.push({ label: "解锁", value: "触控ID" });
  } else if (allSpecs.face_id) {
    specItems.push({ label: "解锁", value: "面容ID" });
  }

  const originalPrice = Math.round(product.price / 0.85);
  const savings = originalPrice - product.price;

  return (
    <a
      href={product.product_url}
      target="_blank"
      rel="noopener noreferrer"
      className="block bg-white rounded-xl overflow-hidden hover:shadow-md transition-all duration-200 border border-gray-100 group"
    >
      <div className="flex items-center gap-3 p-2.5">
        {/* Image */}
        <div className="flex-shrink-0 w-20 h-20 bg-gray-50 rounded-lg overflow-hidden">
          {product.image_url ? (
            <img
              src={product.image_url}
              alt={product.name}
              className="w-full h-full object-contain"
              loading="lazy"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-gray-300 text-2xl">
              💻
            </div>
          )}
        </div>

        {/* Info Section */}
        <div className="flex-1 min-w-0">
          {/* Title */}
          <h3 className="text-sm font-medium text-[#1D1D1F] truncate group-hover:text-[#0071E3] transition-colors mb-1.5">
            {product.name}
          </h3>

          {/* Specs - Full Display with Labels */}
          {specItems.length > 0 ? (
            <div className="flex flex-wrap gap-x-2 gap-y-0.5 text-xs">
              {specItems.slice(0, 10).map((item, index) => (
                <span key={index} className="inline">
                  <span className="text-gray-400">{item.label}:</span>
                  <span className="text-gray-700 ml-0.5 font-medium">
                    {item.value}
                  </span>
                  {index < Math.min(specItems.length, 10) - 1 && (
                    <span className="text-gray-300 mx-1">|</span>
                  )}
                </span>
              ))}
              {specItems.length > 10 && (
                <span className="text-gray-400">
                  +{specItems.length - 10}项
                </span>
              )}
            </div>
          ) : (
            // 如果没有规格信息，显示 description 的一部分
            product.description && (
              <div className="text-xs text-gray-500 line-clamp-2">
                {product.description.slice(0, 100)}
                {product.description.length > 100 && "..."}
              </div>
            )
          )}
        </div>

        {/* Price Section */}
        <div className="flex-shrink-0 text-right">
          <div className="text-lg font-bold text-[#0071E3]">
            ¥{product.price?.toLocaleString()}
          </div>
          {savings > 0 && (
            <div className="text-[10px] text-green-600">
              省¥{savings.toLocaleString()}
            </div>
          )}
        </div>
      </div>
    </a>
  );
}
