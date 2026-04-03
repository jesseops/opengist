package db

type GistCollaborator struct {
	GistID uint `gorm:"primaryKey;index"`
	UserID uint `gorm:"primaryKey;index"`
	Gist   Gist `gorm:"constraint:OnDelete:CASCADE"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`
}

// GetCollaboratorsForGist returns all collaborators for a given gist.
func GetCollaboratorsForGist(gistID uint) ([]*GistCollaborator, error) {
	var collaborators []*GistCollaborator
	err := db.Preload("User").Where("gist_id = ?", gistID).Find(&collaborators).Error
	return collaborators, err
}

// IsCollaborator checks if a user is a collaborator on a gist.
func IsCollaborator(gistID uint, userID uint) (bool, error) {
	var count int64
	err := db.Model(&GistCollaborator{}).
		Where("gist_id = ? AND user_id = ?", gistID, userID).
		Count(&count).Error
	return count > 0, err
}

// AddCollaborator adds a user as a collaborator on a gist.
func AddCollaborator(gistID uint, userID uint) error {
	collab := &GistCollaborator{
		GistID: gistID,
		UserID: userID,
	}
	return db.Create(collab).Error
}

// RemoveCollaborator removes a user as a collaborator on a gist.
func RemoveCollaborator(gistID uint, userID uint) error {
	return db.Where("gist_id = ? AND user_id = ?", gistID, userID).
		Delete(&GistCollaborator{}).Error
}

// SearchUsers searches for users by username prefix, excluding a given user ID.
func SearchUsers(query string, excludeUserID uint, limit int) ([]*User, error) {
	var users []*User
	err := db.Where("username_normalized LIKE ? AND id != ?", query+"%", excludeUserID).
		Limit(limit).
		Find(&users).Error
	return users, err
}
