package db

type Comment struct {
	ID        uint   `gorm:"primaryKey"`
	GistID    uint   `gorm:"index"`
	UserID    uint   `gorm:"index"`
	Content   string
	Revision  string `gorm:"size:40;index"` // optional: scopes comment to a specific revision hash
	CreatedAt int64
	UpdatedAt int64

	User User `gorm:"foreignKey:UserID"`
	Gist Gist `gorm:"foreignKey:GistID"`
}

func CreateComment(comment *Comment) error {
	return db.Create(comment).Error
}

func GetCommentsByGist(gistID uint, revision string) ([]*Comment, error) {
	var comments []*Comment
	q := db.Preload("User").Where("gist_id = ?", gistID)
	if revision != "" && revision != "HEAD" {
		q = q.Where("revision = ? OR revision = ''", revision)
	}
	err := q.Order("created_at asc").Find(&comments).Error
	return comments, err
}

func GetCommentByID(id uint) (*Comment, error) {
	var comment Comment
	err := db.Preload("User").Where("id = ?", id).First(&comment).Error
	return &comment, err
}

func DeleteComment(id uint) error {
	return db.Delete(&Comment{}, id).Error
}

func CountCommentsByGist(gistID uint) (int64, error) {
	var count int64
	err := db.Model(&Comment{}).Where("gist_id = ?", gistID).Count(&count).Error
	return count, err
}
