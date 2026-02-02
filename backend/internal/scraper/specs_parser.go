package scraper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParsedSpecs contains detailed product specifications
type ParsedSpecs struct {
	Model        string `json:"model"`
	ScreenSize   string `json:"screen_size"`
	Chip         string `json:"chip"`
	CPUCores     int    `json:"cpu_cores"`
	GPUCores     int    `json:"gpu_cores"`
	Storage      string `json:"storage"`
	Memory       string `json:"memory"`
	Color        string `json:"color"`
	Connectivity string `json:"connectivity"`
	Ethernet     bool   `json:"ethernet"`
	DisplayType  string `json:"display_type"`
	StandType    string `json:"stand_type"`
	CaseSize     string `json:"case_size"`
	BandType     string `json:"band_type"`
}

// ParseProductSpecs extracts detailed specs from product name/title
func ParseProductSpecs(name string) ParsedSpecs {
	specs := ParsedSpecs{}
	lowerName := strings.ToLower(name)

	// Parse model/series
	specs.Model = parseModel(name)

	// Parse screen size
	specs.ScreenSize = parseScreenSize(name)

	// Parse chip
	specs.Chip, specs.CPUCores, specs.GPUCores = parseChip(name)

	// Parse storage capacity
	specs.Storage = parseStorage(name)

	// Parse memory (RAM)
	specs.Memory = parseMemory(name)

	// Parse color
	specs.Color = parseColor(name, lowerName)

	// Parse connectivity
	specs.Connectivity = parseConnectivity(name)

	// Parse ethernet
	specs.Ethernet = parseEthernet(name, lowerName)

	// Parse display type (for Studio Display)
	specs.DisplayType = parseDisplayType(name, lowerName)

	// Parse stand type (for displays)
	specs.StandType = parseStandType(name, lowerName)

	// Parse case size (for Apple Watch)
	if strings.Contains(lowerName, "watch") {
		specs.CaseSize = parseWatchCaseSize(name)
		specs.BandType = parseWatchBand(name)
	}

	return specs
}

// ToMap converts ParsedSpecs to a map for JSON serialization
func (p ParsedSpecs) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	if p.Model != "" {
		result["model"] = p.Model
	}
	if p.ScreenSize != "" {
		result["screen_size"] = p.ScreenSize
	}
	if p.Chip != "" {
		result["chip"] = p.Chip
	}
	if p.CPUCores > 0 {
		result["cpu_cores"] = p.CPUCores
	}
	if p.GPUCores > 0 {
		result["gpu_cores"] = p.GPUCores
	}
	if p.Storage != "" {
		result["storage"] = p.Storage
	}
	if p.Memory != "" {
		result["memory"] = p.Memory
	}
	if p.Color != "" {
		result["color"] = p.Color
	}
	if p.Connectivity != "" {
		result["connectivity"] = p.Connectivity
	}
	if p.Ethernet {
		result["ethernet"] = true
	}
	if p.DisplayType != "" {
		result["display_type"] = p.DisplayType
	}
	if p.StandType != "" {
		result["stand_type"] = p.StandType
	}
	if p.CaseSize != "" {
		result["case_size"] = p.CaseSize
	}
	if p.BandType != "" {
		result["band_type"] = p.BandType
	}
	return result
}

// parseModel extracts product model from name
func parseModel(name string) string {
	models := []struct {
		pattern string
		label   string
	}{
		{`(?i)MacBook Pro`, "MacBook Pro"},
		{`(?i)MacBook Air`, "MacBook Air"},
		{`(?i)Mac mini`, "Mac mini"},
		{`(?i)iMac`, "iMac"},
		{`(?i)Mac Studio`, "Mac Studio"},
		{`(?i)iPad Pro`, "iPad Pro"},
		{`(?i)iPad Air`, "iPad Air"},
		{`(?i)iPad mini`, "iPad mini"},
		{`(?i)\biPad\b(?!.*Air|.*Pro|.*mini)`, "iPad"},
		{`(?i)Apple Watch SE`, "Apple Watch SE"},
		{`(?i)Apple Watch Ultra`, "Apple Watch Ultra"},
		{`(?i)Apple Watch Series \d+`, "Apple Watch"},
		{`(?i)Apple Watch(?!\s+Ultra|\s+SE|\s+Series)`, "Apple Watch"},
		{`(?i)AirPods Pro`, "AirPods Pro"},
		{`(?i)AirPods Max`, "AirPods Max"},
		{`(?i)AirPods(?!.*Pro|.*Max)`, "AirPods"},
		{`(?i)HomePod(?!.*mini)`, "HomePod"},
		{`(?i)HomePod mini`, "HomePod mini"},
		{`(?i)Studio Display`, "Studio Display"},
		{`(?i)Pro Display`, "Pro Display"},
	}

	for _, m := range models {
		if matched, _ := regexp.MatchString(m.pattern, name); matched {
			return m.label
		}
	}
	return ""
}

// parseScreenSize extracts screen size
func parseScreenSize(name string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[寸英寸inches]+(?i)`)
	match := re.FindStringSubmatch(name)
	if len(match) > 1 {
		return match[1] + "英寸"
	}
	return ""
}

// parseChip extracts chip info
func parseChip(name string) (chip string, cpuCores, gpuCores int) {
	reMaxPro := regexp.MustCompile(`(?i)(M[1-4])\s*(Max|Pro)\s*芯片`)
	match := reMaxPro.FindStringSubmatch(name)
	if len(match) > 0 {
		chip = match[1] + " " + match[2]
	} else {
		reBase := regexp.MustCompile(`(?i)(M[1-4])\s*芯片`)
		match = reBase.FindStringSubmatch(name)
		if len(match) > 0 {
			chip = match[1]
		}
	}

	reCPU := regexp.MustCompile(`(\d+)\s*[核core]+\s*中央处理器|(\d+)\s*[核core]+\s*CPU`)
	match = reCPU.FindStringSubmatch(name)
	if len(match) > 0 {
		cpuCores = parseSingleInt(match[1])
		if cpuCores == 0 {
			cpuCores = parseSingleInt(match[2])
		}
	}

	reGPU := regexp.MustCompile(`(\d+)\s*[核core]+\s*[图形gpu]|GPU\s*(\d+)`)
	match = reGPU.FindStringSubmatch(name)
	if len(match) > 0 {
		gpuCores = parseSingleInt(match[1])
		if gpuCores == 0 {
			gpuCores = parseSingleInt(match[2])
		}
	}

	return chip, cpuCores, gpuCores
}

// parseStorage extracts storage capacity
func parseStorage(name string) string {
	explicitPatterns := []string{
		`(\d+)\s*(GB|TB)\s*存储`,
		`(\d+)\s*(GB|TB)\s*SSD`,
		`(\d+)\s*(GB|TB)\s*硬盘`,
	}

	for _, pattern := range explicitPatterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(name)
		if len(match) > 0 {
			return formatStorage(match[1], match[2])
		}
	}

	reAfterDash := regexp.MustCompile(`-\s*(\d+)\s*(GB|TB)`)
	match := reAfterDash.FindStringSubmatch(name)
	if len(match) > 0 {
		return formatStorage(match[1], match[2])
	}

	reAll := regexp.MustCompile(`(\d+)\s*(GB|TB)`)
	matches := reAll.FindAllStringSubmatch(name, -1)

	for _, match := range matches {
		size := match[1]
		unit := match[2]

		// Get position for context check
		idx := strings.Index(name, match[0])
		if idx == -1 {
			continue
		}

		after := ""
		end := idx + len(match[0])
		if end < len(name) {
			after = strings.ToLower(strings.TrimSpace(name[end:min(end+10, len(name))]))
		}
		if strings.HasPrefix(after, "核") || strings.HasPrefix(after, "core") {
			continue
		}

		before := ""
		if idx > 0 {
			before = strings.ToLower(strings.TrimSpace(name[max(0, idx-10):idx]))
		}
		if strings.HasSuffix(before, "核") || strings.HasSuffix(before, "core") {
			continue
		}

		num := parseSingleInt(size)
		if isValidStorageSize(num) {
			return formatStorage(size, unit)
		}
	}

	return ""
}

func formatStorage(size, unit string) string {
	num := parseSingleInt(size)
	if num >= 1024 {
		return fmt.Sprintf("%dTB", num/1024)
	}
	return fmt.Sprintf("%s%s", size, unit)
}

func isValidStorageSize(size int) bool {
	validSizes := []int{16, 32, 64, 128, 256, 512, 1024, 2048, 4096}
	for _, v := range validSizes {
		if size == v {
			return true
		}
	}
	return false
}

// parseMemory extracts RAM/memory
func parseMemory(name string) string {
	patterns := []string{
		`(\d+)\s*GB\s*统一内存`,
		`(\d+)\s*GB\s*内存`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(name)
		if len(match) > 0 {
			return match[1] + "GB"
		}
	}
	return ""
}

// parseColor extracts color
func parseColor(name, lowerName string) string {
	colors := map[string]string{
		"深空灰色": "深空灰",
		"深空黑": "深空黑",
		"深空黑色": "深空黑",
		"星光色": "星光色",
		"午夜色": "午夜色",
		"银色": "银色",
		"金色": "金色",
		"玫瑰金色": "玫瑰金",
		"绿色": "绿色",
		"蓝色": "蓝色",
		"紫色": "紫色",
		"红色": "红色",
		"橙色": "橙色",
		"黄色": "黄色",
		"粉色": "粉色",
		"黑色": "黑色",
		"白色": "白色",
		"灰色": "灰色",
		"summit": "山地",
		"starlight": "星光色",
		"midnight": "午夜色",
		"silver": "银色",
		"gold": "金色",
		"space grey": "深空灰",
		"space gray": "深空灰",
		"space black": "深空黑",
		"pink": "粉色",
		"orange": "橙色",
		"blue": "蓝色",
		"purple": "紫色",
		"red": "红色",
		"green": "绿色",
	}

	for cn, value := range colors {
		if strings.Contains(name, cn) || strings.Contains(lowerName, strings.ToLower(cn)) {
			return value
		}
	}
	return ""
}

// parseConnectivity extracts network connectivity
func parseConnectivity(name string) string {
	if strings.Contains(name, "Wi-Fi") && strings.Contains(name, "蜂窝网络") {
		return "Wi-Fi + 蜂窝网络"
	}
	if strings.Contains(name, "Wi-Fi") || strings.Contains(name, "WLAN") {
		return "Wi-Fi"
	}
	if strings.Contains(name, "GPS") && strings.Contains(name, "Cellular") {
		return "GPS + 蜂窝网络"
	}
	if strings.Contains(name, "GPS") {
		return "GPS"
	}
	return ""
}

// parseEthernet checks for ethernet
func parseEthernet(name, lowerName string) bool {
	return strings.Contains(name, "千兆以太网") ||
		strings.Contains(name, "Gigabit") ||
		strings.Contains(lowerName, "ethernet")
}

// parseDisplayType extracts display glass type
func parseDisplayType(name, lowerName string) string {
	if strings.Contains(name, "纳米纹理") || strings.Contains(lowerName, "nano-texture") {
		return "纳米纹理玻璃"
	}
	if strings.Contains(name, "标准玻璃") || strings.Contains(lowerName, "standard glass") {
		return "标准玻璃"
	}
	return ""
}

// parseStandType extracts stand type
func parseStandType(name, lowerName string) string {
	if strings.Contains(name, "可调倾斜度及高度") || strings.Contains(lowerName, "tilt-height") {
		return "可调节支架"
	}
	if strings.Contains(name, "可调倾斜度") || strings.Contains(lowerName, "tilt-adjustable") {
		return "可调节支架"
	}
	if strings.Contains(name, "VESA") || strings.Contains(lowerName, "vesa") {
		return "VESA支架"
	}
	return ""
}

// parseWatchCaseSize extracts Apple Watch case size
func parseWatchCaseSize(name string) string {
	re := regexp.MustCompile(`(\d+)\s*[毫米mm]+`)
	match := re.FindStringSubmatch(name)
	if len(match) > 0 {
		return match[1] + "毫米"
	}
	return ""
}

// parseWatchBand extracts band type
func parseWatchBand(name string) string {
	bands := []struct {
		pattern string
		label   string
	}{
		{`运动型表带|运动表带|Sport Band`, "运动型表带"},
		{`回环式表带|Sport Loop`, "回环式表带"},
		{`米兰表带|Milanese`, "米兰表带"},
		{`皮革表带|Leather`, "皮表带"},
		{`链式表带|Link`, "链式表带"},
		{`编织表带|Braided`, "编织表带"},
	}

	for _, b := range bands {
		re := regexp.MustCompile(b.pattern)
		if re.MatchString(name) {
			return b.label
		}
	}
	return ""
}

func parseSingleInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
