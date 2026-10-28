package db

import "strings"

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

// GetCollaboratorIDsForGist returns just the user IDs of collaborators for a gist.
func GetCollaboratorIDsForGist(gistID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&GistCollaborator{}).
		Where("gist_id = ?", gistID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// SearchUsers searches for users by username prefix, excluding the gist owner and
// users who are already collaborators on the gist.
func SearchUsers(query string, gistID uint, ownerID uint, limit int) ([]*User, error) {
	// "!" is used as the LIKE escape character since it needs no special quoting on any supported database.
	escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query)

	var users []*User
	err := db.Where("username_normalized LIKE ? ESCAPE '!' AND id != ?", escaped+"%", ownerID).
		Where("id NOT IN (SELECT user_id FROM gist_collaborators WHERE gist_id = ?)", gistID).
		Limit(limit).
		Find(&users).Error
	return users, err
}
