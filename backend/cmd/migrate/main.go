// Command to migrate data from JSON files to SQLite database
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"apple-price/internal/model"

	_ "github.com/mattn/go-sqlite3"
)

const version = "1.0.0"

func main() {
	dataDir := flag.String("dir", "./data", "Data directory containing JSON files")
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making changes")
	force := flag.Bool("force", false, "Force overwrite existing SQLite database")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("migrate version %s\n", version)
		return
	}

	fmt.Printf("=== ApplePrice 数据迁移工具 v%s ===\n\n", version)

	// Verify data directory exists
	if _, err := os.Stat(*dataDir); os.IsNotExist(err) {
		fmt.Printf("错误: 数据目录不存在: %s\n", *dataDir)
		os.Exit(1)
	}

	dbPath := filepath.Join(*dataDir, "apple-price.db")

	// Check if SQLite database already exists
	if _, err := os.Stat(dbPath); err == nil && !*force {
		fmt.Printf("错误: SQLite 数据库已存在: %s\n", dbPath)
		fmt.Println("使用 --force 参数覆盖现有数据库，或先手动删除数据库文件")
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("=== 预演模式 (不会修改任何数据) ===")
	}

	// Step 1: Backup existing JSON files
	backupDir := *dataDir + "_backup_" + time.Now().Format("20060102_150405")
	if err := backupJSONFiles(*dataDir, backupDir); err != nil {
		fmt.Printf("警告: 备份失败: %v\n", err)
	} else {
		fmt.Printf("备份完成: %s\n", backupDir)
	}

	// Step 2: Load JSON data
	fmt.Println("\n正在读取 JSON 文件...")
	products, err := readProducts(*dataDir)
	if err != nil {
		fmt.Printf("警告: 无法读取产品数据: %v\n", err)
		products = []*model.Product{}
	}
	fmt.Printf("找到 %d 个产品\n", len(products))

	history, err := readHistory(*dataDir)
	if err != nil {
		fmt.Printf("警告: 无法读取历史数据: %v\n", err)
		history = make(map[string][]model.PriceHistory)
	}

	subs, err := readSubscriptions(*dataDir)
	if err != nil {
		fmt.Printf("警告: 无法读取订阅数据: %v\n", err)
		subs = make(map[string]*model.Subscription)
	}
	fmt.Printf("找到 %d 个订阅\n", len(subs))

	// Step 3: Create SQLite database and schema
	var db *sql.DB
	if !*dryRun {
		fmt.Println("\n正在创建 SQLite 数据库...")
		db, err = createSQLiteDB(dbPath)
		if err != nil {
			fmt.Printf("错误: 无法创建 SQLite 数据库: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()
		fmt.Println("SQLite 数据库创建成功")
	}

	// Step 4: Migrate products
	fmt.Println("\n正在迁移产品...")
	migratedProducts := 0
	for _, p := range products {
		// Calculate value score
		productHistory := history[p.ID]
		valueScore := calculateValueScore(p, productHistory)

		// Calculate lowest/highest price and trend
		lowest, highest, trend := calculatePriceStats(productHistory)

		if !*dryRun {
			_, err = db.Exec(`
				INSERT INTO products (
					id, name, category, region, price, original_price, discount,
					image_url, product_url, specs, stock_status, value_score,
					lowest_price, highest_price, price_trend, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, p.ID, p.Name, p.Category, p.Region, p.Price, p.OriginalPrice,
				p.Discount, p.ImageURL, p.ProductURL, p.Specs, p.StockStatus,
				valueScore, lowest, highest, trend, p.CreatedAt.Unix(), p.UpdatedAt.Unix())
			if err != nil {
				fmt.Printf("警告: 产品迁移失败 %s: %v\n", p.ID, err)
			} else {
				migratedProducts++
			}
		} else {
			migratedProducts++
		}
	}
	fmt.Printf("产品迁移完成: %d/%d\n", migratedProducts, len(products))

	// Step 5: Migrate price history
	fmt.Println("\n正在迁移价格历史...")
	historyCount := 0
	for productID, histList := range history {
		for _, h := range histList {
			if !*dryRun {
				_, err = db.Exec(`
					INSERT INTO price_history (product_id, price, discount, recorded_at)
					VALUES (?, ?, ?, ?)
				`, productID, h.Price, h.Discount, h.Timestamp.Unix())
				if err == nil {
					historyCount++
				}
			} else {
				historyCount++
			}
		}
	}
	fmt.Printf("历史记录迁移完成: %d 条\n", historyCount)

	// Step 6: Migrate subscriptions
	fmt.Println("\n正在迁移订阅...")
	subCount := 0
	for _, sub := range subs {
		if !*dryRun {
			_, err = db.Exec(`
				INSERT INTO subscriptions (id, product_id, bark_key, created_at)
				VALUES (?, ?, ?, ?)
			`, sub.ID, sub.ProductID, sub.BarkKey, sub.CreatedAt.Unix())
			if err == nil {
				subCount++
			}
		} else {
			subCount++
		}
	}
	fmt.Printf("订阅迁移完成: %d/%d\n", subCount, len(subs))

	// Summary
	fmt.Println("\n" + repeat("=", 50))
	fmt.Println("迁移完成!")
	fmt.Println(repeat("=", 50))
	fmt.Printf("\n产品数: %d\n", migratedProducts)
	fmt.Printf("历史记录: %d\n", historyCount)
	fmt.Printf("订阅数: %d\n", subCount)
	fmt.Printf("\n数据库位置: %s\n", dbPath)
	fmt.Printf("备份位置: %s\n", backupDir)

	if !*dryRun {
		fmt.Println("\n提示: 如需回滚，可以使用备份目录中的 JSON 文件")
		fmt.Println("下一步: 启动服务器测试数据是否正确迁移")
	}
}

// createSQLiteDB creates the database and runs migrations
func createSQLiteDB(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// Open database with WAL mode and foreign keys
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL&_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		region TEXT NOT NULL,
		price REAL NOT NULL,
		original_price REAL NOT NULL,
		discount REAL NOT NULL,
		image_url TEXT,
		product_url TEXT NOT NULL,
		specs TEXT,
		stock_status TEXT NOT NULL DEFAULT 'available',
		value_score REAL DEFAULT 0,
		lowest_price REAL,
		highest_price REAL,
		price_trend TEXT DEFAULT 'stable',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS price_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id TEXT NOT NULL,
		price REAL NOT NULL,
		discount REAL NOT NULL,
		recorded_at INTEGER NOT NULL,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS subscriptions (
		id TEXT PRIMARY KEY,
		product_id TEXT NOT NULL,
		bark_key TEXT,
		email TEXT,
		created_at INTEGER NOT NULL,
		FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
	CREATE INDEX IF NOT EXISTS idx_products_region ON products(region);
	CREATE INDEX IF NOT EXISTS idx_products_stock_status ON products(stock_status);
	CREATE INDEX IF NOT EXISTS idx_products_value_score ON products(value_score DESC);
	CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_price_history_product_id ON price_history(product_id);
	CREATE INDEX IF NOT EXISTS idx_price_history_product_recorded ON price_history(product_id, recorded_at DESC);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_product_id ON subscriptions(product_id);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// calculateValueScore calculates value score based on historical data
func calculateValueScore(product *model.Product, history []model.PriceHistory) float64 {
	score := 50.0 // Base score

	// 1. Discount score (0-30 points)
	score += discountScore(product.Discount)

	// 2. Price trend score (0-25 points)
	score += trendScore(history)

	// 3. Stock status score (0-15 points)
	score += stockScore(product.StockStatus)

	// 4. Price position score (0-20 points)
	score += pricePositionScore(product.Price, history)

	// 5. Age score (0-10 points)
	score += ageScore(product.CreatedAt)

	// Cap at 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

func discountScore(discount float64) float64 {
	switch {
	case discount >= 15:
		return 30
	case discount >= 12:
		return 25
	case discount >= 10:
		return 20
	case discount >= 8:
		return 15
	case discount >= 5:
		return 10
	default:
		return discount * 2
	}
}

func trendScore(history []model.PriceHistory) float64 {
	if len(history) < 3 {
		return 0
	}

	recent := history[len(history)-3:]
	startPrice := recent[0].Price
	endPrice := recent[2].Price
	change := (endPrice - startPrice) / startPrice

	if change < -0.02 { // Fell more than 2%
		return 25
	} else if change < -0.01 { // Fell more than 1%
		return 20
	} else if change < 0 { // Falling
		return 15
	} else if change > 0.02 { // Rose more than 2%
		return 0
	}
	return 10 // Stable
}

func stockScore(status string) float64 {
	switch status {
	case "available":
		return 15
	case "limited":
		return 10
	default:
		return 0
	}
}

func pricePositionScore(currentPrice float64, history []model.PriceHistory) float64 {
	if len(history) == 0 {
		return 10
	}

	min, max := history[0].Price, history[0].Price
	for _, h := range history {
		if h.Price < min {
			min = h.Price
		}
		if h.Price > max {
			max = h.Price
		}
	}

	if max == min {
		return 10
	}

	position := (currentPrice - min) / (max - min)
	if position <= 0.1 {
		return 20
	} else if position <= 0.3 {
		return 15
	} else if position <= 0.5 {
		return 10
	} else if position <= 0.7 {
		return 5
	}
	return 0
}

func ageScore(createdAt time.Time) float64 {
	days := time.Since(createdAt).Hours() / 24
	switch {
	case days <= 7:
		return 10
	case days <= 30:
		return 7
	case days <= 90:
		return 3
	default:
		return 0
	}
}

// calculatePriceStats calculates lowest price, highest price, and trend
func calculatePriceStats(history []model.PriceHistory) (lowest, highest float64, trend string) {
	if len(history) == 0 {
		return 0, 0, "stable"
	}

	lowest = history[0].Price
	highest = history[0].Price
	for _, h := range history {
		if h.Price < lowest {
			lowest = h.Price
		}
		if h.Price > highest {
			highest = h.Price
		}
	}

	// Determine trend
	trend = "stable"
	if len(history) >= 3 {
		recent := history[len(history)-3:]
		change := (recent[2].Price - recent[0].Price) / recent[0].Price
		if change < -0.02 {
			trend = "falling"
		} else if change > 0.02 {
			trend = "rising"
		}
	}

	return lowest, highest, trend
}

// backupJSONFiles creates a backup of all JSON files
func backupJSONFiles(dataDir, backupDir string) error {
	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	// Files to backup
	files := []string{
		"products.json",
		"history.json",
		"subscriptions.json",
	}

	for _, file := range files {
		src := filepath.Join(dataDir, file)
		if _, err := os.Stat(src); err == nil {
			dst := filepath.Join(backupDir, file)
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func readProducts(dataDir string) ([]*model.Product, error) {
	productsFile := filepath.Join(dataDir, "products.json")
	data, err := os.ReadFile(productsFile)
	if err != nil {
		return nil, err
	}

	var products []*model.Product
	if err := json.Unmarshal(data, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func readHistory(dataDir string) (map[string][]model.PriceHistory, error) {
	historyFile := filepath.Join(dataDir, "history.json")
	data, err := os.ReadFile(historyFile)
	if err != nil {
		// File might not exist, return empty map
		return make(map[string][]model.PriceHistory), nil
	}

	var history map[string][]model.PriceHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	if history == nil {
		return make(map[string][]model.PriceHistory), nil
	}

	return history, nil
}

func readSubscriptions(dataDir string) (map[string]*model.Subscription, error) {
	subsFile := filepath.Join(dataDir, "subscriptions.json")
	data, err := os.ReadFile(subsFile)
	if err != nil {
		// File might not exist, return empty map
		return make(map[string]*model.Subscription), nil
	}

	var subs map[string]*model.Subscription
	if err := json.Unmarshal(data, &subs); err != nil {
		return nil, err
	}

	if subs == nil {
		return make(map[string]*model.Subscription), nil
	}

	return subs, nil
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
