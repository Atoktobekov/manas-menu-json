package main

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
