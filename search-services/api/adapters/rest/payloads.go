package rest

type pingResponse struct {
	Replies map[string]string `json:"replies"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// update payloads
type updateStatusResponse struct {
	Status string `json:"status"`
}

type updateStatsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

// search payloads
type comicResponse struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type searchResponse struct {
	Comics []comicResponse `json:"comics"`
	Total  int             `json:"total"`
}

// auth payloads
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type botTelegramLoginRequest struct {
	TgID      int64  `json:"tg_id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

// favorites payloads
type favoriteItemResponse struct {
	ComicID       int32 `json:"comic_id"`
	CreatedAtUnix int64 `json:"created_at_unix"`
}

type favoritesListResponse struct {
	Items []favoriteItemResponse `json:"items"`
}
