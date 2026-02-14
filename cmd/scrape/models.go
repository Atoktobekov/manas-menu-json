package main

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
