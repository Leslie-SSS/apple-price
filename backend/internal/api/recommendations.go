package api

import (
	"apple-price/internal/model"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// RecommendationRequest 推荐请求
type RecommendationRequest struct {
	BudgetMin  *float64 `json:"budget_min"`
	BudgetMax  *float64 `json:"budget_max"`
	Category   string   `json:"category"` // 前端发送的具体分类，如 "MacBook Air"
	UseCase    string   `json:"use_case"` // office_portable, office_desktop, creative, coding, study, entertainment, fitness, daily, ""
	Chip       string   `json:"chip"`     // M1, M2, M3, M3 Max, M3 Pro, ""
	StorageMin *int     `json:"storage_min"`
	StorageMax *int     `json:"storage_max"`
	PreferHigh bool     `json:"prefer_high"` // 高预算时，优先推荐价位高的产品
}

// RecommendationResult 推荐结果
type RecommendationResult struct {
	Product *model.Product `json:"product"`
	Score   float64        `json:"score"`
	Reasons []string       `json:"reasons"`
}

// RecommendationResponse 推荐响应
type RecommendationResponse struct {
	Results    []*RecommendationResult `json:"results"`
	TotalCount int                     `json:"total_count"`
}

// categoryMapping 前端具体分类到后端通用分类的映射
var categoryMapping = map[string][]string{
	"MacBook Air":   {"Mac", "Air"},
	"MacBook Pro":   {"Mac", "Pro"},
	"Mac mini":      {"Mac", "mini"},
	"iPad Pro":      {"iPad", "Pro"},
	"iPad Air":      {"iPad", "Air"},
	"iPad":          {"iPad"},
	"Watch":         {"Watch"},
	"Accessory":     {"Accessory"},
}

// Validate validates the recommendation request
func (r *RecommendationRequest) Validate() error {
	if r.BudgetMin != nil && *r.BudgetMin < 0 {
		return errors.New("budget_min cannot be negative")
	}
	if r.BudgetMax != nil && *r.BudgetMax < 0 {
		return errors.New("budget_max cannot be negative")
	}
	if r.BudgetMin != nil && r.BudgetMax != nil && *r.BudgetMin > *r.BudgetMax {
		return errors.New("budget_min cannot be greater than budget_max")
	}
	if r.StorageMin != nil && *r.StorageMin < 0 {
		return errors.New("storage_min cannot be negative")
	}
	if r.StorageMax != nil && *r.StorageMax < 0 {
		return errors.New("storage_max cannot be negative")
	}
	if r.StorageMin != nil && r.StorageMax != nil && *r.StorageMin > *r.StorageMax {
		return errors.New("storage_min cannot be greater than storage_max")
	}
	return nil
}

// HandleRecommendation 处理推荐请求
func (h *Handlers) HandleRecommendation(c *gin.Context) {
	var req RecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	results := h.recommend(req)

	c.JSON(200, RecommendationResponse{
		Results:    results,
		TotalCount: len(results),
	})
}

// recommend 根据请求生成推荐列表
func (h *Handlers) recommend(req RecommendationRequest) []*RecommendationResult {
	// 获取所有产品
	products := h.store.GetAllProducts()

	// 严格筛选候选产品 - 预算必须精确匹配
	candidates := h.filterCandidates(products, req)

	// 如果没有筛选结果，只放宽分类，不放宽预算
	if len(candidates) == 0 {
		candidates = h.filterCandidatesRelaxed(products, req)
	}

	// 为每个产品计算推荐分数和理由
	var results []*RecommendationResult
	for _, product := range candidates {
		score, reasons := h.calculateRecommendationScore(product, req)
		results = append(results, &RecommendationResult{
			Product: product,
			Score:   score,
			Reasons: reasons,
		})
	}

	// 按分数排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// 如果用户偏好高价位，且分数相同，按价格降序
	if req.PreferHigh {
		sort.SliceStable(results, func(i, j int) bool {
			if abs(results[i].Score-results[j].Score) < 5 {
				return results[i].Product.Price > results[j].Product.Price
			}
			return results[i].Score > results[j].Score
		})
	}

	// 限制返回数量
	const maxResultsPerRequest = 20
	if len(results) > maxResultsPerRequest {
		results = results[:maxResultsPerRequest]
	}

	return results
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// filterCandidates 严格筛选符合条件的产品
func (h *Handlers) filterCandidates(products []*model.Product, req RecommendationRequest) []*model.Product {
	var candidates []*model.Product

	for _, p := range products {
		// 预算筛选 - 严格匹配，不放宽
		if req.BudgetMin != nil && p.Price < *req.BudgetMin {
			continue
		}
		if req.BudgetMax != nil && p.Price > *req.BudgetMax {
			continue
		}

		// 分类筛选 - 支持模糊匹配
		if req.Category != "" && !h.matchCategory(p, req.Category) {
			continue
		}

		// 芯片筛选
		if req.Chip != "" {
			chip := extractChip(p.Name)
			if chip != req.Chip {
				continue
			}
		}

		// 存储筛选
		storage := extractStorage(p.Name)
		if req.StorageMin != nil && storage < *req.StorageMin {
			continue
		}
		if req.StorageMax != nil && storage > *req.StorageMax {
			continue
		}

		candidates = append(candidates, p)
	}

	return candidates
}

// filterCandidatesRelaxed 宽松筛选 - 只放宽分类，不放宽预算
func (h *Handlers) filterCandidatesRelaxed(products []*model.Product, req RecommendationRequest) []*model.Product {
	var candidates []*model.Product

	for _, p := range products {
		// 预算筛选 - 保持严格
		if req.BudgetMin != nil && p.Price < *req.BudgetMin {
			continue
		}
		if req.BudgetMax != nil && p.Price > *req.BudgetMax {
			continue
		}

		// 分类筛选 - 只检查大类，放宽具体型号匹配
		if req.Category != "" {
			mappedCategories, ok := categoryMapping[req.Category]
			if ok {
				// 检查是否匹配大类（如 "Mac", "iPad"）
				baseCategory := strings.ToLower(mappedCategories[0])
				productCategory := strings.ToLower(p.Category)
				if !strings.Contains(productCategory, baseCategory) {
					continue
				}
			}
		}

		candidates = append(candidates, p)
	}

	return candidates
}

// matchCategory 检查产品是否匹配分类
func (h *Handlers) matchCategory(product *model.Product, frontendCategory string) bool {
	productName := strings.ToLower(product.Name)
	productCategory := strings.ToLower(product.Category)

	// 获取映射的分类
	mappedCategories, ok := categoryMapping[frontendCategory]
	if !ok {
		// 如果没有映射，直接检查是否包含
		return strings.Contains(productCategory, strings.ToLower(frontendCategory)) ||
			strings.Contains(productName, strings.ToLower(frontendCategory))
	}

	// 检查大类匹配
	baseCategory := strings.ToLower(mappedCategories[0])
	if !strings.Contains(productCategory, baseCategory) {
		return false
	}

	// 检查具体产品名称是否包含关键词
	hasKeywords := false
	for _, keyword := range mappedCategories[1:] {
		if strings.Contains(productName, strings.ToLower(keyword)) {
			hasKeywords = true
		}
	}

	// 如果有具体关键词但都不匹配，则返回false
	if len(mappedCategories) > 1 && !hasKeywords {
		return false
	}

	return true
}

// calculateRecommendationScore 计算推荐分数和理由
func (h *Handlers) calculateRecommendationScore(product *model.Product, req RecommendationRequest) (float64, []string) {
	score := 50.0 // 基础分
	var reasons []string

	// 计算节省金额
	savings := int(product.OriginalPrice - product.Price)

	// 1. 预算匹配 - 最优先的理由
	if req.BudgetMax != nil {
		if product.Price <= *req.BudgetMax {
			score += 20
			reasons = append(reasons, fmt.Sprintf("符合你的¥%d预算，比新机省¥%d", int(*req.BudgetMax), savings))
		}
	} else if req.BudgetMin != nil {
		score += 15
		reasons = append(reasons, fmt.Sprintf("比新机省¥%d，同样的Apple品质保证", savings))
	}

	// 2. 官方翻新品质保证 - 始终显示
	if len(reasons) < 3 {
		reasons = append(reasons, "官方翻新=全新外观，电池>80%，享受1年保修")
	}

	// 3. 库存状态 (0-15分)
	if product.StockStatus == "available" {
		score += 15
		if len(reasons) < 3 {
			reasons = append(reasons, "现货速发，今天下单明天到手")
		}
	}

	// 4. 价格位置 (0-15分)
	history := h.store.GetPriceHistory(product.ID)
	if len(history) > 1 {
		minPrice := history[0].Price
		maxPrice := history[0].Price
		for _, h := range history {
			if h.Price < minPrice {
				minPrice = h.Price
			}
			if h.Price > maxPrice {
				maxPrice = h.Price
			}
		}

		if maxPrice > minPrice {
			position := (product.Price - minPrice) / (maxPrice - minPrice)
			if position <= 0.2 {
				score += 15
				if len(reasons) < 3 {
					reasons = append(reasons, "当前价格接近历史低位，是好时机")
				}
			}
		}
	}

	// 5. 性价比评分 (0-15分)
	valueScore := product.ValueScore
	if valueScore >= 80 {
		score += 15
		if len(reasons) < 3 {
			reasons = append(reasons, fmt.Sprintf("性价比评分%.0f分，非常值得入手", valueScore))
		}
	}

	// 6. 用途匹配 (0-25分)
	score += h.useCaseScore(product, req.UseCase, &reasons)

	// 7. 价格偏好调整
	if req.PreferHigh && req.BudgetMax != nil {
		budgetUtilization := product.Price / *req.BudgetMax
		if budgetUtilization >= 0.8 {
			score += 10
		}
	} else if req.BudgetMax != nil {
		budgetUtilization := product.Price / *req.BudgetMax
		if budgetUtilization <= 0.5 {
			score += 10
		}
	}

	// 限制理由数量为3条
	if len(reasons) > 3 {
		reasons = reasons[:3]
	}

	return score, reasons
}

// useCaseScore 计算用途匹配分数
func (h *Handlers) useCaseScore(product *model.Product, useCase string, reasons *[]string) float64 {
	score := 0.0

	chip := extractChip(product.Name)
	storage := extractStorage(product.Name)
	name := strings.ToLower(product.Name)
	category := strings.ToLower(product.Category)

	switch useCase {
	case "office", "office_portable":
		if strings.Contains(name, "air") {
			score += 20
			*reasons = append(*reasons, "💼 MacBook Air 轻薄便携")
		} else if strings.Contains(name, "13寸") || strings.Contains(name, "14寸") {
			score += 15
			*reasons = append(*reasons, "💼 适中尺寸，便携办公")
		}
		if chip == "M2" || chip == "M3" {
			score += 10
		}

	case "office_desktop":
		if strings.Contains(name, "pro") {
			score += 20
			*reasons = append(*reasons, "🖥️ MacBook Pro 性能强劲")
		} else if strings.Contains(category, "mini") || strings.Contains(name, "mac mini") {
			score += 25
			*reasons = append(*reasons, "🖥️ Mac mini 桌面办公性价比之选")
		}
		if storage >= 512 {
			score += 5
		}

	case "creative":
		if chip == "M3 Max" {
			score += 30
			*reasons = append(*reasons, "🎨 M3 Max 顶级创作性能")
		} else if chip == "M3 Pro" || chip == "M2 Max" || chip == "M2 Ultra" {
			score += 25
			*reasons = append(*reasons, "🎨 专业芯片满足创作需求")
		} else if strings.Contains(name, "pro") {
			score += 15
		}
		if storage >= 512 {
			score += 5
			*reasons = append(*reasons, "💾 大容量存储适合创作文件")
		}

	case "coding":
		if chip == "M3 Max" || chip == "M2 Max" {
			score += 30
			*reasons = append(*reasons, "👨‍💻 Max 系列芯片编译性能顶尖")
		} else if chip == "M3 Pro" || chip == "M2 Pro" {
			score += 25
			*reasons = append(*reasons, "👨‍💻 Pro 系列芯片适合开发")
		}
		if storage >= 512 {
			score += 5
		}
		if strings.Contains(category, "mini") || strings.Contains(name, "mac mini") {
			score += 10
			*reasons = append(*reasons, "💻 Mac mini 性价比开发利器")
		}

	case "study":
		if strings.Contains(category, "ipad") {
			score += 20
			*reasons = append(*reasons, "📚 iPad 适合笔记和学习")
		}
		if strings.Contains(name, "air") {
			score += 10
		}

	case "entertainment":
		if strings.Contains(category, "ipad") {
			score += 20
			*reasons = append(*reasons, "🎬 iPad 娱乐体验佳")
		} else if product.Price < 8000 {
			score += 15
			*reasons = append(*reasons, "🎬 性价比高，适合日常娱乐")
		}

	case "fitness":
		if strings.Contains(category, "watch") {
			score += 30
			*reasons = append(*reasons, "🏃 Apple Watch 运动追踪，健康监测")
		}

	case "daily":
		if strings.Contains(category, "watch") {
			score += 30
			*reasons = append(*reasons, "🚶 Apple Watch 消息提醒，接打电话")
		}
	}

	return score
}

// extractChip 从产品名称提取芯片型号
func extractChip(name string) string {
	nameLower := strings.ToLower(name)
	// 按优先级匹配，Max > Pro > 基础型号
	if strings.Contains(nameLower, "m3 max") {
		return "M3 Max"
	}
	if strings.Contains(nameLower, "m3 pro") {
		return "M3 Pro"
	}
	if strings.Contains(nameLower, "m2 ultra") {
		return "M2 Ultra"
	}
	if strings.Contains(nameLower, "m2 max") {
		return "M2 Max"
	}
	if strings.Contains(nameLower, "m2 pro") {
		return "M2 Pro"
	}
	if strings.Contains(nameLower, "m3") {
		return "M3"
	}
	if strings.Contains(nameLower, "m2") {
		return "M2"
	}
	if strings.Contains(nameLower, "m1") {
		return "M1"
	}
	return ""
}

// extractStorage 从产品名称提取存储容量（GB）
func extractStorage(name string) int {
	re := regexp.MustCompile(`(\d+)\s*(GB|TB)`)
	matches := re.FindStringSubmatch(name)
	if len(matches) >= 3 {
		var value int
		fmt.Sscanf(matches[1], "%d", &value)
		if matches[2] == "TB" {
			return value * 1024
		}
		return value
	}
	return 0
}
