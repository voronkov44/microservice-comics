package core

type UpdateStatus string

const (
	StatusUpdateUnknown UpdateStatus = "unknown"
	StatusUpdateIdle    UpdateStatus = "idle"
	StatusUpdateRunning UpdateStatus = "running"
)

type UpdateStats struct {
	WordsTotal    int
	WordsUnique   int
	ComicsFetched int
	ComicsTotal   int
}

type SearchComic struct {
	ID  int
	URL string
}

type SearchResult struct {
	Comics []SearchComic
	Total  int
}

type FavoriteItem struct {
	ComicID       int32
	CreatedAtUnix int64
}
