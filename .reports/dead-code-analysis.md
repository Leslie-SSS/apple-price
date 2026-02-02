# Dead Code Analysis Report

**Project:** apple-price
**Analysis Date:** 2025-02-02
**Project Type:** Go backend + React/Vite/TypeScript frontend

---

## Executive Summary

This report identifies potentially unused code, dependencies, and files in the apple-price project. The codebase is relatively well-organized with minimal dead code detected.

**Overall Health:** GOOD - Most code is actively used

---

## Backend (Go) Analysis

### Project Structure
```
backend/
├── cmd/server/main.go       # Entry point - ACTIVE
├── internal/
│   ├── api/                 # HTTP handlers - ACTIVE
│   ├── config/              # Configuration - ACTIVE
│   ├── model/               # Data models - ACTIVE
│   ├── notify/              # Bark/Email notifications - ACTIVE
│   ├── scraper/             # Web scraping - ACTIVE
│   └── store/               # Storage (JSON + SQLite) - ACTIVE
├── data/                    # Runtime data directory - ACTIVE
├── server                   # Compiled binary (current)
├── server-new               # Compiled binary (old)
└── go.mod / go.sum          # Dependencies
```

### SAFE - Candidates for Removal

| Item | Type | Location | Reason |
|------|------|----------|--------|
| `server-new` | Binary | `/home/leslie/keepbuild/projects/apple-price/backend/server-new` | Old compiled binary, ~17MB. Current binary is `server` |

### DANGER - DO NOT REMOVE (Critical Dependencies)

All Go dependencies are actively used:
- `github.com/gin-gonic/gin` - HTTP framework
- `github.com/joho/godotenv` - Environment configuration
- `github.com/mattn/go-sqlite3` - SQLite database driver
- All indirect dependencies (sonic, validator, etc.) - Required by Gin

### Active Code Files (All Verified Used)

| File | Status | Notes |
|------|--------|-------|
| `cmd/server/main.go` | ACTIVE | Entry point |
| `internal/api/handlers.go` | ACTIVE | HTTP request handlers |
| `internal/api/routes.go` | ACTIVE | Route definitions |
| `internal/api/recommendations.go` | ACTIVE | AI recommendation logic |
| `internal/config/config.go` | ACTIVE | Configuration loading |
| `internal/model/product.go` | ACTIVE | Data models |
| `internal/notify/bark.go` | ACTIVE | Bark notification service |
| `internal/notify/email.go` | ACTIVE | Email notification service |
| `internal/notify/dispatcher.go` | ACTIVE | Notification dispatcher |
| `internal/scraper/interface.go` | ACTIVE | Scraper interface |
| `internal/scraper/client.go` | ACTIVE | HTTP client for scraping |
| `internal/scraper/scheduler.go` | ACTIVE | Scraping scheduler |
| `internal/scraper/detail_scraper.go` | ACTIVE | Async detail fetching |
| `internal/scraper/apple_scraper.go` | ACTIVE | Apple website scraper |
| `internal/scraper/specs_parser.go` | ACTIVE | Product specs parsing |
| `internal/store/interface.go` | ACTIVE | Store interface |
| `internal/store/store.go` | ACTIVE | JSON store implementation |
| `internal/store/sqlite.go` | ACTIVE | SQLite store implementation |

---

## Frontend (React/TypeScript/Vite) Analysis

### Project Structure
```
frontend/
├── src/
│   ├── components/          # React components
│   ├── hooks/              # Custom hooks
│   ├── pages/              # Page components
│   ├── services/           # API services
│   └── utils/              # Utility functions
├── public/                 # Static assets
├── dist/                   # Build output
├── node_modules/           # Dependencies
├── package.json
├── vite.config.ts
└── tsconfig.json
```

### SAFE - Unused Dependencies

| Package | Version | Type | Size | Reason |
|---------|---------|------|------|--------|
| `date-fns` | ^3.0.0 | dependency | ~80KB | NOT imported anywhere in codebase |
| `zustand` | ^4.4.7 | dependency | ~5KB | NOT imported anywhere in codebase |

**Estimated Savings:** ~85KB

### SAFE - Unused DevDependencies

| Package | Version | Reason |
|---------|---------|--------|
| `autoprefixer` | ^10.4.16 | PostCSS handles this automatically via Tailwind |
| `postcss` | ^8.4.32 | Used indirectly by Tailwind, can be indirect dependency |
| `@typescript-eslint/eslint-plugin` | ^6.14.0 | Only needed if using `npm run lint` |
| `@typescript-eslint/parser` | ^6.14.0 | Only needed if using `npm run lint` |
| `eslint` | ^8.55.0 | Only needed if using `npm run lint` |
| `eslint-plugin-react-hooks` | ^4.6.0 | Only needed if using `npm run lint` |
| `eslint-plugin-react-refresh` | ^0.4.5 | Only needed if using `npm run lint` |

### SAFE - Unused Icon Components

| Component | Location | Reason |
|-----------|----------|--------|
| `ChevronDownIcon` | `src/components/icons.tsx` | Exported but never imported/used |

### CAUTION - Component Usage Verification

| Component | Status | Notes |
|-----------|--------|-------|
| `Header.tsx` | ACTIVE | Used in App.tsx |
| `Home.tsx` | ACTIVE | Used in App.tsx |
| `NotificationModal.tsx` | ACTIVE | Used in App.tsx |
| `ProductCard.tsx` | ACTIVE | Used in Home.tsx |
| `ProductDetailModal.tsx` | ACTIVE | NOT currently imported - potential dead code |
| `SpecsTable.tsx` | ACTIVE | Used in ProductDetailModal.tsx |
| `icons.tsx` | ACTIVE | Used by multiple components |

### SAFE - Unused Component

| Component | Location | Reason |
|-----------|----------|--------|
| `ProductDetailModal.tsx` | `src/components/ProductDetailModal.tsx` | Defined but NOT imported anywhere in the codebase. Note: `SpecsTable.tsx` depends on it, so if ProductDetailModal is unused, SpecsTable might also be unused. |

**Verification:** `SpecsTable` is only imported by `ProductDetailModal`. If ProductDetailModal is not used, SpecsTable is also dead code.

---

## Cleanup Plan

### Priority 1: Safe to Remove Immediately

#### Backend
1. **Remove old compiled binary**
   - File: `/home/leslie/keepbuild/projects/apple-price/backend/server-new`
   - Command: `rm /home/leslie/keepbuild/projects/apple-price/backend/server-new`
   - Space Saved: ~17MB

#### Frontend
2. **Remove unused npm dependencies**
   ```bash
   cd /home/leslie/keepbuild/projects/apple-price/frontend
   npm uninstall date-fns zustand
   ```
   - Estimated Savings: ~85KB

3. **Remove unused icon component**
   - File: `ChevronDownIcon` from `src/components/icons.tsx`
   - Lines: 19-25

4. **Remove unused modal component** (IF confirmed not needed)
   - File: `src/components/ProductDetailModal.tsx`
   - Note: This is a significant component. Verify with team before removal.

5. **Remove unused SpecsTable component** (IF ProductDetailModal is removed)
   - File: `src/components/SpecsTable.tsx`
   - Note: Only used by ProductDetailModal

### Priority 2: Review Before Removal

1. **ESLint and related packages**
   - Only remove if project doesn't use `npm run lint`
   - Safe to keep for development quality

2. **autoprefixer and postcss**
   - Currently listed as direct dependencies
   - Used by Tailwind CSS build process
   - Recommendation: Keep for now, verify build still works after removal

---

## Risk Assessment Matrix

| Item | Risk Level | Impact | Effort |
|------|------------|--------|--------|
| Remove `server-new` binary | SAFE | 17MB freed | Low |
| Remove `date-fns`, `zustand` | SAFE | 85KB freed | Low |
| Remove `ChevronDownIcon` | SAFE | ~100 bytes | Low |
| Remove ProductDetailModal | CAUTION | ~3KB | Medium |
| Remove ESLint packages | CAUTION | None | Low |
| Remove autoprefixer/postcss | MEDIUM | Build may break | Medium |

---

## Detailed Findings

### Backend Dependency Analysis (Go)

All Go dependencies are verified to be in use:

1. **Direct Dependencies:**
   - `github.com/gin-gonic/gin` - HTTP server framework (main.go)
   - `github.com/joho/godotenv` - .env file loading (config.go)
   - `github.com/mattn/go-sqlite3` - SQLite driver (sqlite.go)

2. **Indirect Dependencies** (all required by Gin or its dependencies):
   - sonic, validator/v10, go-json, etc.
   - No unused transitive dependencies detected

### Frontend Dependency Analysis (TypeScript/React)

**Verified Active Dependencies:**
- `react` - Core framework
- `react-dom` - DOM rendering
- `axios` - HTTP client (used in api.ts)
- `vite` - Build tool
- `@vitejs/plugin-react` - React plugin for Vite
- `typescript` - Type checking
- `tailwindcss` - CSS framework
- `@types/react`, `@types/react-dom` - Type definitions

**Unused Dependencies:**
- `date-fns` - No imports found
- `zustand` - No imports found (state management library)

**Potentially Unused Development Dependencies:**
- ESLint ecosystem - Only if not using `npm run lint`
- `autoprefixer`, `postcss` - May be indirect dependencies

---

## Recommendations

### Immediate Actions

1. **Remove old binary:**
   ```bash
   rm /home/leslie/keepbuild/projects/apple-price/backend/server-new
   ```

2. **Remove unused frontend dependencies:**
   ```bash
   cd /home/leslie/keepbuild/projects/apple-price/frontend
   npm uninstall date-fns zustand
   ```

3. **Remove unused icon:**
   Edit `src/components/icons.tsx` and remove `ChevronDownIcon` function

### Consider for Future

1. **ProductDetailModal component** - This appears to be incomplete functionality. Consider:
   - Completing the implementation and integrating it
   - Removing it if not planned to be used

2. **ESLint setup** - Either:
   - Start using it regularly (`npm run lint` before commits)
   - Remove to reduce dependencies

3. **Add to .gitignore:**
   ```
   # Compiled binaries
   backend/server
   backend/server-new
   backend/server.exe
   ```

---

## Files Analyzed

### Backend Files (18)
- cmd/server/main.go
- internal/api/handlers.go
- internal/api/routes.go
- internal/api/recommendations.go
- internal/config/config.go
- internal/model/product.go
- internal/notify/bark.go
- internal/notify/email.go
- internal/notify/dispatcher.go
- internal/scraper/interface.go
- internal/scraper/client.go
- internal/scraper/scheduler.go
- internal/scraper/detail_scraper.go
- internal/store/interface.go
- internal/store/store.go
- internal/store/sqlite.go
- go.mod
- go.sum

### Frontend Files (13)
- src/App.tsx
- src/main.tsx
- src/components/Header.tsx
- src/components/ProductCard.tsx
- src/components/NotificationModal.tsx
- src/components/ProductDetailModal.tsx
- src/components/SpecsTable.tsx
- src/components/icons.tsx
- src/pages/Home.tsx
- src/hooks/useProducts.ts
- src/services/api.ts
- src/utils/product.ts
- package.json

---

## Tools Used

1. **depcheck** - Frontend dependency analysis
2. **Manual code review** - Import/export analysis
3. **go mod graph** - Backend dependency verification
4. **grep/search** - Pattern matching for usage verification

---

## Summary

**Total Items Safe to Remove:** 5
**Total Items Requiring Review:** 4
**Total Space Savings Potential:** ~17.1 MB (mostly from compiled binary)

**Code Quality:** Good - Minimal dead code found
**Dependency Health:** Good - Most dependencies are actively used

---

**Report Generated:** 2025-02-02
**Analysis Tool:** Manual + depcheck
