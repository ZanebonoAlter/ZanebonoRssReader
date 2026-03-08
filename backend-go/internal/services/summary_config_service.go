package services

import (
	"encoding/json"
	"errors"

	"gorm.io/gorm"
	"my-robot-backend/internal/models"
	"my-robot-backend/pkg/database"
)

const summaryConfigKey = "summary_config"

func LoadSummaryConfig() (map[string]interface{}, *models.AISettings, error) {
	var settings models.AISettings
	err := database.DB.Where("key = ?", summaryConfigKey).First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]interface{}{}, nil, nil
		}
		return nil, nil, err
	}

	config := map[string]interface{}{}
	if settings.Value == "" {
		return config, &settings, nil
	}

	if err := json.Unmarshal([]byte(settings.Value), &config); err != nil {
		return nil, nil, err
	}

	return config, &settings, nil
}

func SaveSummaryConfig(config map[string]interface{}, description string) error {
	configJSON, err := models.ToJSONValue(config)
	if err != nil {
		return err
	}

	var settings models.AISettings
	dbErr := database.DB.Where("key = ?", summaryConfigKey).First(&settings).Error
	if dbErr == nil {
		settings.Value = configJSON
		if description != "" {
			settings.Description = description
		}
		return database.DB.Save(&settings).Error
	}

	if !errors.Is(dbErr, gorm.ErrRecordNotFound) {
		return dbErr
	}

	settings = models.AISettings{
		Key:         summaryConfigKey,
		Value:       configJSON,
		Description: description,
	}

	return database.DB.Create(&settings).Error
}
