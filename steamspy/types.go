package steamspy

// Game holds metadata and stats for one Steam game from SteamSpy.
type Game struct {
	AppID          int    `json:"appid"`
	Name           string `json:"name"`
	Developer      string `json:"developer"`
	Publisher      string `json:"publisher"`
	Positive       int    `json:"positive"`
	Negative       int    `json:"negative"`
	Owners         string `json:"owners"`          // e.g. "10,000,000 .. 20,000,000"
	AverageForever int    `json:"average_forever"` // average play time in minutes, all time
	Average2Weeks  int    `json:"average_2weeks"`  // average play time in minutes, last 2 weeks
	Price          string `json:"price"`           // cents as string, "0" = free
	CCU            int    `json:"ccu"`             // current concurrent users
	Languages      string `json:"languages"`
	Genre          string `json:"genre"`
	ScoreRank      string `json:"score_rank"`
}
