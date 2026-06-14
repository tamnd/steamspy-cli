package steamspy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// wireApp is the raw JSON shape returned by the SteamSpy API.
type wireApp struct {
	AppID          int            `json:"appid"`
	Name           string         `json:"name"`
	Developer      string         `json:"developer"`
	Publisher      string         `json:"publisher"`
	Positive       int            `json:"positive"`
	Negative       int            `json:"negative"`
	Owners         string         `json:"owners"`
	AverageForever int            `json:"average_forever"`
	Average2Weeks  int            `json:"average_2weeks"`
	Price          string         `json:"price"`
	Genre          string         `json:"genre"`
	Tags           map[string]int `json:"tags"`
}

// App is the output record for a Steam game.
type App struct {
	AppID     int    `kit:"id" json:"appid"`
	Name      string `json:"name"`
	Developer string `json:"developer"`
	Genre     string `json:"genre"`
	Owners    string `json:"owners"`
	Positive  int    `json:"positive"`
	Negative  int    `json:"negative"`
	Price     string `json:"price"`
	AvgHours  string `json:"avg_hours"`
	TopTags   string `json:"top_tags"`
}

// toApp converts a wireApp into the output App, computing derived fields.
func toApp(w wireApp) App {
	// Format price: "0" -> "Free", otherwise parse cents and format as dollars.
	price := "Free"
	if w.Price != "0" && w.Price != "" {
		if cents, err := strconv.ParseFloat(w.Price, 64); err == nil {
			price = fmt.Sprintf("$%.2f", cents/100)
		} else {
			price = w.Price
		}
	}

	// Average hours: average_forever minutes / 60, formatted to 1 decimal.
	avgHours := fmt.Sprintf("%.1f", float64(w.AverageForever)/60)

	// Top 5 tags by vote count.
	topTags := topNTags(w.Tags, 5)

	return App{
		AppID:     w.AppID,
		Name:      w.Name,
		Developer: w.Developer,
		Genre:     w.Genre,
		Owners:    w.Owners,
		Positive:  w.Positive,
		Negative:  w.Negative,
		Price:     price,
		AvgHours:  avgHours,
		TopTags:   topTags,
	}
}

// topNTags returns the top n tags by vote count, comma-joined.
func topNTags(tags map[string]int, n int) string {
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(tags))
	for k, v := range tags {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	if len(pairs) > n {
		pairs = pairs[:n]
	}
	names := make([]string, len(pairs))
	for i, p := range pairs {
		names[i] = p.k
	}
	return strings.Join(names, ", ")
}
