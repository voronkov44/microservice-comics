package db

import "github.com/lib/pq"

// ComicsRow - промежуточная модель для скана, не стал выносить в core/models,
// так как зависит от постгреса и pq драйвера
type ComicsRow struct {
	ID    int            `db:"id"`
	URL   string         `db:"img_url"`
	Title pq.StringArray `db:"title"`
	Alt   pq.StringArray `db:"alt"`
	Words pq.StringArray `db:"words"`
}
