package narrative

import (
	"fmt"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/platform/database"
)

func ListActiveConcepts() ([]models.BoardConcept, error) {
	var concepts []models.BoardConcept
	if err := database.DB.Where("is_active = ?", true).
		Order("display_order ASC, id ASC").
		Find(&concepts).Error; err != nil {
		return nil, fmt.Errorf("list active board concepts: %w", err)
	}
	return concepts, nil
}

func GetConceptByID(id uint) (*models.BoardConcept, error) {
	var concept models.BoardConcept
	if err := database.DB.Where("id = ?", id).First(&concept).Error; err != nil {
		return nil, fmt.Errorf("get board concept %d: %w", id, err)
	}
	return &concept, nil
}

func CreateConcept(name, description, scopeType string, scopeCategoryID *uint) (*models.BoardConcept, error) {
	concept := &models.BoardConcept{
		Name:            name,
		Description:     description,
		ScopeType:       scopeType,
		ScopeCategoryID: scopeCategoryID,
		IsActive:        true,
		IsSystem:        false,
	}

	if err := database.DB.Create(concept).Error; err != nil {
		return nil, fmt.Errorf("create board concept: %w", err)
	}
	return concept, nil
}

func UpdateConcept(id uint, name, description string) (*models.BoardConcept, error) {
	concept, err := GetConceptByID(id)
	if err != nil {
		return nil, err
	}

	concept.Name = name
	concept.Description = description

	if err := database.DB.Model(concept).Updates(map[string]interface{}{
		"name":        name,
		"description": description,
	}).Error; err != nil {
		return nil, fmt.Errorf("update board concept %d: %w", id, err)
	}
	return concept, nil
}

func DeactivateConcept(id uint) error {
	result := database.DB.Model(&models.BoardConcept{}).
		Where("id = ?", id).
		Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("deactivate board concept %d: %w", id, result.Error)
	}
	return nil
}
