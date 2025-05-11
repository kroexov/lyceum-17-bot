package botsrv

type StudentForm struct {
	TgId     string `json:"tgId"`
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
	Class    string `json:"class"`
}

type GraduateForm struct {
	TgId           string `json:"tgId"`
	Nickname       string `json:"nickname"`
	Name           string `json:"name"`
	Year           string `json:"year"`
	Class          string `json:"class"`
	CityInfo       string `json:"cityInfo"`
	UniversityInfo string `json:"universityInfo"`
	WorkInfo       string `json:"workInfo"`
	ExtraInfo      string `json:"extraInfo"`
}
