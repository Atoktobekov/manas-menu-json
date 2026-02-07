package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	timezone   = "Asia/Bishkek"
	currency   = "KGS"
	kantinURL  = "https://beslenme.manas.edu.kg/menu"
	buffetURL1 = "https://beslenme.manas.edu.kg/1"
)

var (
	reDateTR = regexp.MustCompile(`^\s*(\d{2})\.(\d{2})\.(\d{4})\b`) // 07.02.2026 ...
	reKcal   = regexp.MustCompile(`(?i)Kalori:\s*([0-9]+)`)
	rePrice  = regexp.MustCompile(`(?i)Fiyat[ıi]:\s*([0-9]+)`)
)

// ---------- Output models ----------

type KantinFood struct {
	ID   string `json:"id"`
	Name struct {
		TR string `json:"tr"`
		RU string `json:"ru"`
		EN string `json:"en"`
	} `json:"name"`
	CaloriesKcal int `json:"caloriesKcal"`
}

type KantinMenuDay struct {
	Date  string   `json:"date"`  // YYYY-MM-DD
	Items []string `json:"items"` // food IDs
}

type KantinMeta struct {
	Timezone    string `json:"timezone"`
	Source      string `json:"source"`
	LastUpdated string `json:"lastUpdated"`
}

type KantinOut struct {
	Foods []KantinFood    `json:"foods"`
	Menus []KantinMenuDay `json:"menus"`
	Meta  KantinMeta      `json:"meta"`
}

type BuffetItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

type BuffetCategory struct {
	ID    string       `json:"id"`
	Title string       `json:"title"`
	Items []BuffetItem `json:"items"`
}

type BuffetMeta struct {
	Timezone    string `json:"timezone"`
	Currency    string `json:"currency"`
	LastUpdated string `json:"lastUpdated"`
}

type BuffetOut struct {
	Categories []BuffetCategory `json:"categories"`
	Meta       BuffetMeta       `json:"meta"`
}

// ---------- Helpers ----------

func fetchDoc(url string) (*goquery.Document, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; manas-menu-scraper/1.0)")
	client := &http.Client{Timeout: 30 * time.Second}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("bad status: %s", res.Status)
	}
	return goquery.NewDocumentFromReader(res.Body)
}

// Turkish → ASCII slug, safe for IDs.
func slugTR(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	replacer := strings.NewReplacer(
		"ç", "c", "Ç", "c",
		"ğ", "g", "Ğ", "g",
		"ı", "i", "I", "i", // dotless i + capital I
		"İ", "i", "i̇", "i", // sometimes comes as i + dot
		"ö", "o", "Ö", "o",
		"ş", "s", "Ş", "s",
		"ü", "u", "Ü", "u",
	)
	s = replacer.Replace(s)

	// Replace separators with underscore
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "/", " ")
	s = strings.ReplaceAll(s, "’", "")
	s = strings.ReplaceAll(s, "'", "")

	// Keep only [a-z0-9_]
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if r == ' ' || r == '_' {
			if !prevUnderscore {
				b.WriteByte('_')
				prevUnderscore = true
			}
		}
		// ignore other punctuation
	}

	out := strings.Trim(b.String(), "_")
	out = regexp.MustCompile(`_+`).ReplaceAllString(out, "_")
	if out == "" {
		out = "item"
	}
	return out
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func parseKcal(s string) (int, bool) {
	m := reKcal.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

func parsePrice(s string) (int, bool) {
	m := rePrice.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

func parseDate(s string) (string, bool) {
	m := reDateTR.FindStringSubmatch(s)
	if len(m) != 4 {
		return "", false
		// dd mm yyyy
	}
	return fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1]), true
}

// ---------- Scrapers ----------

// NOTE: страницы выглядят как последовательность заголовков:
// - "#####  07.02.2026 Cumartesi" (дата)
// - далее блоки блюд: "#####  Yayla Çorbası" + "###### Kalori: 175" :contentReference[oaicite:2]{index=2}
func scrapeKantin() (KantinOut, error) {
	doc, err := fetchDoc(kantinURL)
	if err != nil {
		return KantinOut{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	foodsByID := map[string]KantinFood{}
	menusByDate := map[string][]string{}

	var currentDate string
	var lastFoodName string

	// Берём контент и идём по заголовкам: h5/h6 часто используются (в HTML),
	// но чтобы быть устойчивым, собираем текст всех h5/h6 подряд.
	doc.Find("h5, h6").Each(func(i int, sel *goquery.Selection) {
		t := strings.TrimSpace(sel.Text())

		// Дата?
		if d, ok := parseDate(t); ok {
			currentDate = d
			return
		}

		// Если это "Kalori: N"
		if kcal, ok := parseKcal(t); ok {
			if currentDate == "" || lastFoodName == "" {
				return
			}
			id := slugTR(lastFoodName)

			// уникализируем id если вдруг коллизия
			uniqueID := id
			for {
				if _, exists := foodsByID[uniqueID]; !exists {
					break
				}
				// если уже есть — проверим, это то же блюдо?
				ex := foodsByID[uniqueID]
				if ex.Name.TR == lastFoodName && ex.CaloriesKcal == kcal {
					break
				}
				uniqueID = uniqueID + "_2"
			}

			var f KantinFood
			f.ID = uniqueID
			f.Name.TR = lastFoodName
			// Пока нет переводов на сайте — дублируем TR.
			f.Name.RU = lastFoodName
			f.Name.EN = lastFoodName
			f.CaloriesKcal = kcal
			foodsByID[uniqueID] = f

			menusByDate[currentDate] = append(menusByDate[currentDate], uniqueID)
			lastFoodName = ""
			return
		}

		// Иначе считаем, что это имя блюда (h5)
		// (на странице меню это как раз заголовок блюда) :contentReference[oaicite:3]{index=3}
		if currentDate != "" {
			lastFoodName = t
		}
	})

	// Собираем в стабильном порядке (по дате)
	dates := make([]string, 0, len(menusByDate))
	for d := range menusByDate {
		dates = append(dates, d)
	}
	sortStringsISO(dates)

	menus := make([]KantinMenuDay, 0, len(dates))
	for _, d := range dates {
		menus = append(menus, KantinMenuDay{
			Date:  d,
			Items: menusByDate[d],
		})
	}

	foods := make([]KantinFood, 0, len(foodsByID))
	for _, f := range foodsByID {
		foods = append(foods, f)
	}
	// сортируем foods по id, чтобы диффы были аккуратные
	sortFoodsByID(foods)

	return KantinOut{
		Foods: foods,
		Menus: menus,
		Meta: KantinMeta{
			Timezone:    timezone,
			Source:      "manas_kantin",
			LastUpdated: now,
		},
	}, nil
}

// Страница /1 — категории идут заголовком, дальше позиции:
// "#### SICAK İÇECEK", затем "##### ÇAY DEMLEME" + "###### Fiyatı: 18 som" :contentReference[oaicite:4]{index=4}
func scrapeBuffet1() (BuffetOut, error) {
	doc, err := fetchDoc(buffetURL1)
	if err != nil {
		return BuffetOut{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	catMapRU := map[string]string{
		"SICAK İÇECEK":       "Горячие напитки",
		"PİZZA VE PİDELER":   "Пицца и пиде",
		"UNLU MAMÜLLER":      "Выпечка",
		"KAHVALTILIKLAR":     "Завтраки",
		// если появятся новые категории — добавишь сюда
	}

	var categories []BuffetCategory
	var current *BuffetCategory
	var lastItemName string

	doc.Find("h4, h5, h6").Each(func(i int, sel *goquery.Selection) {
		t := strings.TrimSpace(sel.Text())

		if sel.Is("h4") {
			// новая категория
			if current != nil {
				categories = append(categories, *current)
			}
			id := slugTR(t)
			title := catMapRU[t]
			if title == "" {
				// если нет перевода — оставим как есть (или потом дополнишь map)
				title = t
			}
			current = &BuffetCategory{ID: id, Title: title}
			lastItemName = ""
			return
		}

		if sel.Is("h6") {
			if current == nil || lastItemName == "" {
				return
			}
			price, ok := parsePrice(t)
			if !ok {
				return
			}
			item := BuffetItem{
				ID:    slugTR(lastItemName),
				Name:  lastItemName,
				Price: price,
			}
			current.Items = append(current.Items, item)
			lastItemName = ""
			return
		}

		if sel.Is("h5") {
			if current != nil {
				lastItemName = t
			}
		}
	})

	if current != nil {
		categories = append(categories, *current)
	}

	return BuffetOut{
		Categories: categories,
		Meta: BuffetMeta{
			Timezone:    timezone,
			Currency:    currency,
			LastUpdated: now,
		},
	}, nil
}

// ---------- Sorting helpers (to keep JSON diffs clean) ----------

func sortStringsISO(a []string) {
	// YYYY-MM-DD лексикографически уже сортится как дата
	for i := 0; i < len(a)-1; i++ {
		for j := i + 1; j < len(a); j++ {
			if a[j] < a[i] {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}

func sortFoodsByID(a []KantinFood) {
	for i := 0; i < len(a)-1; i++ {
		for j := i + 1; j < len(a); j++ {
			if a[j].ID < a[i].ID {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}

func main() {
	kantin, err := scrapeKantin()
	if err != nil {
		log.Fatal(err)
	}
	buffet, err := scrapeBuffet1()
	if err != nil {
		log.Fatal(err)
	}

	if err := writeJSON("public/manas_kantin.json", kantin); err != nil {
		log.Fatal(err)
	}
	if err := writeJSON("public/buffet_1.json", buffet); err != nil {
		log.Fatal(err)
	}

	fmt.Println("OK: wrote public/manas_kantin.json and public/buffet_1.json")
}
