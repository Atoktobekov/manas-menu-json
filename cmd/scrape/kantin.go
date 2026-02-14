package main

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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
