package rest

type pingResponse struct {
	Replies map[string]string `json:"replies"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type updateStatusResponse struct {
	Status string `json:"status"`
}

type updateStatsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type comicResponse struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type searchResponse struct {
	Comics []comicResponse `json:"comics"`
	Total  int             `json:"total"`
}
