package main

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Страница /1 — категории идут заголовком, дальше позиции:
// "#### SICAK İÇECEK", затем "##### ÇAY DEMLEME" + "###### Fiyatı: 18 som" :contentReference[oaicite:4]{index=4}

func scrapeBuffet1() (BuffetOut, error) {
	doc, err := fetchDoc(buffetURL)
	if err != nil {
		return BuffetOut{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	catMapRU := map[string]string{
		"SICAK İÇECEK":     "Горячие напитки",
		"PİZZA VE PİDELER": "Пицца и пиде",
		"UNLU MAMÜLLER":    "Выпечка",
		"KAHVALTILIKLAR":   "Завтраки",
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
