package storage

import (
	"log"
	"time"

	"tg-antispam/internal/models"

	"gorm.io/gorm"
)

// GroupRepository handles database operations for GroupInfo
type GroupRepository struct {
	db *gorm.DB
}

// NewGroupRepository creates a new GroupRepository
func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// MigrateTable ensures the GroupInfo table exists with the right schema
func (r *GroupRepository) MigrateTable() error {
	// Create or update table schema
	err := r.db.AutoMigrate(&models.GroupInfo{})
	if err != nil {
		return err
	}

	// Check if the Language column exists, if not, add it
	if !r.db.Migrator().HasColumn(&models.GroupInfo{}, "Language") {
		err = r.db.Migrator().AddColumn(&models.GroupInfo{}, "Language")
		if err != nil {
			return err
		}
		// Set default value for existing records
		r.db.Model(&models.GroupInfo{}).Where("language = ? OR language IS NULL", "").Update("language", models.LangSimplifiedChinese)
	}

	return nil
}

// GetGroupInfo retrieves group information from the database by GroupID
func (r *GroupRepository) GetGroupInfo(groupID int64) (*models.GroupInfo, error) {
	var groupInfo models.GroupInfo
	result := r.db.Where("group_id = ?", groupID).First(&groupInfo)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &groupInfo, nil
}

// GetGroupInfoByID retrieves group information from the database by ID
func (r *GroupRepository) GetGroupInfoByID(id uint) (*models.GroupInfo, error) {
	var groupInfo models.GroupInfo
	result := r.db.First(&groupInfo, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &groupInfo, nil
}

// CreateOrUpdateGroupInfo creates a new group info record or updates an existing one
func (r *GroupRepository) CreateOrUpdateGroupInfo(groupInfo *models.GroupInfo) error {
	var existing models.GroupInfo
	result := r.db.Where("group_id = ?", groupInfo.GroupID).First(&existing)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// New record, set created time
			groupInfo.CreatedAt = time.Now()
			groupInfo.UpdatedAt = time.Now()
			return r.db.Create(groupInfo).Error
		}
		return result.Error
	}

	// Existing record, update it
	groupInfo.ID = existing.ID
	groupInfo.CreatedAt = existing.CreatedAt
	groupInfo.UpdatedAt = time.Now()

	return r.db.Save(groupInfo).Error
}

// GetAllGroupInfo retrieves all group information from the database
func (r *GroupRepository) GetAllGroupInfo() ([]*models.GroupInfo, error) {
	var groups []*models.GroupInfo
	result := r.db.Find(&groups)
	if result.Error != nil {
		return nil, result.Error
	}
	return groups, nil
}

// DeleteGroupInfo removes a group info record from the database by GroupID
func (r *GroupRepository) DeleteGroupInfo(groupID int64) error {
	result := r.db.Where("group_id = ?", groupID).Delete(&models.GroupInfo{})
	return result.Error
}

// DeleteGroupInfoByID removes a group info record from the database by ID
func (r *GroupRepository) DeleteGroupInfoByID(id uint) error {
	result := r.db.Delete(&models.GroupInfo{}, id)
	return result.Error
}

// InitializeGroups loads all groups from the database into the cache
func InitializeGroups(groupInfoManager *models.GroupInfoManager) error {
	if DB == nil {
		log.Printf("Database is not enabled, skipping group initialization")
		return nil
	}

	repo := NewGroupRepository(DB)
	groups, err := repo.GetAllGroupInfo()
	if err != nil {
		return err
	}

	for _, group := range groups {
		groupInfoManager.AddGroupInfo(group)
	}

	log.Printf("Loaded %d groups from database into cache", len(groups))
	return nil
}
