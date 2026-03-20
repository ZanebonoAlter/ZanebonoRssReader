package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"

	"my-robot-backend/internal/domain/models"
	"my-robot-backend/internal/domain/topicanalysis"
	"my-robot-backend/internal/platform/config"
	"my-robot-backend/internal/platform/database"
)

// Event keywords for classification
var eventKeywords = []string{
	"发布会", "发布", "版本更新", "会议", "峰会", "论坛", "展览", "展会",
	"发布会", "发布日", "发布", "发布会", "首发", "正式发布", "公测",
	"发布会", "发布会", "发布会", "发布会", "发布会", "发布会", "发布会",
	"发布", "发布", "发布", "发布", "发布", "发布", "发布",
	"会议", "峰会", "论坛", "展览", "展会", "活动", "典礼",
	"launch", "release", "conference", "event", "summit", "forum",
}

// Person name patterns - Chinese and Western names
var chineseNamePattern = regexp.MustCompile(`^[\x{4e00}-\x{9fa5}]{2,4}$`)
var westernNamePattern = regexp.MustCompile(`^[A-Z][a-z]+(\s+[A-Z][a-z]+)*$`)
var personTitlePattern = regexp.MustCompile(`(?:CEO|CTO|CFO|COO|创始人|创始人|董事长|总裁|副总裁|总监|经理|工程师|博士|教授|负责人)$`)

// isPersonLabel checks if the label is likely a person's name
func isPersonLabel(label string) bool {
	trimmed := strings.TrimSpace(label)

	// Check Chinese name pattern (2-4 characters, common Chinese name length)
	if chineseNamePattern.MatchString(trimmed) {
		return true
	}

	// Check Western name pattern
	if westernNamePattern.MatchString(trimmed) {
		return true
	}

	// Check if it ends with a person title
	if personTitlePattern.MatchString(trimmed) {
		return true
	}

	// Common person name indicators
	lowerLabel := strings.ToLower(trimmed)
	personIndicators := []string{"ceo", "cto", "cfo", "coo", "founder", "创始人", "董事长", "总裁", "vp", " president", "ceo", "cto"}
	for _, indicator := range personIndicators {
		if strings.Contains(lowerLabel, indicator) {
			return true
		}
	}

	return false
}

// isEventLabel checks if the label contains event-related keywords
func isEventLabel(label string) bool {
	lowerLabel := strings.ToLower(label)

	for _, keyword := range eventKeywords {
		if strings.Contains(lowerLabel, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// classifyTag determines the category for a tag based on heuristics
func classifyTag(tag *models.TopicTag) string {
	label := tag.Label

	// If kind was "entity", check for person names first
	if tag.Kind == "entity" {
		if isPersonLabel(label) {
			return models.TagCategoryPerson
		}
	}

	// Check for event keywords
	if isEventLabel(label) {
		return models.TagCategoryEvent
	}

	// Check for person names even if kind was not "entity"
	if isPersonLabel(label) {
		return models.TagCategoryPerson
	}

	// Default to keyword
	return models.TagCategoryKeyword
}

// migrationResult holds the results of the migration
type migrationResult struct {
	total      int
	updated    int
	skipped    int
	errors     int
	embeddings int
}

// migrateTags performs the tag category migration
func migrateTags(dryRun bool, generateEmbeddings bool) (*migrationResult, error) {
	result := &migrationResult{}

	// Fetch all tags
	var tags []models.TopicTag
	if err := database.DB.Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}

	result.total = len(tags)

	// Initialize embedding service if needed
	var embeddingSvc *topicanalysis.EmbeddingService
	if generateEmbeddings {
		embeddingSvc = topicanalysis.NewEmbeddingService()
	}

	for i, tag := range tags {
		// Determine new category
		newCategory := classifyTag(&tag)
		oldCategory := tag.Category

		// Check if update is needed
		if oldCategory != newCategory {
			if dryRun {
				log.Printf("[DRY RUN] Would update tag %d: %q category: %q -> %q (kind: %q)",
					tag.ID, tag.Label, oldCategory, newCategory, tag.Kind)
				result.skipped++
			} else {
				// Update the tag
				tag.Category = newCategory
				tag.Kind = newCategory // Backward compatibility

				if err := database.DB.Save(&tag).Error; err != nil {
					log.Printf("ERROR: Failed to update tag %d: %v", tag.ID, err)
					result.errors++
					continue
				}

				log.Printf("Updated tag %d: %q category: %q -> %q (kind: %q)",
					tag.ID, tag.Label, oldCategory, newCategory, tag.Kind)
				result.updated++
			}
		} else {
			result.skipped++
			log.Printf("Skipped tag %d: %q already has correct category %q", tag.ID, tag.Label, newCategory)
		}

		// Generate embedding if requested and category was updated
		if generateEmbeddings && embeddingSvc != nil && (dryRun || oldCategory != newCategory) {
			if dryRun {
				log.Printf("[DRY RUN] Would generate embedding for tag %d: %q", tag.ID, tag.Label)
				result.embeddings++
			} else {
				// Generate embedding
				embedding, err := embeddingSvc.GenerateEmbedding(nil, &tag)
				if err != nil {
					log.Printf("WARNING: Failed to generate embedding for tag %d: %v", tag.ID, err)
				} else {
					// Save embedding
					if err := embeddingSvc.SaveEmbedding(embedding); err != nil {
						log.Printf("WARNING: Failed to save embedding for tag %d: %v", tag.ID, err)
					} else {
						log.Printf("Generated embedding for tag %d: %q", tag.ID, tag.Label)
						result.embeddings++
					}
				}
			}
		}

		// Progress indicator
		if (i+1)%100 == 0 {
			log.Printf("Processed %d/%d tags...", i+1, len(tags))
		}
	}

	return result, nil
}

func main() {
	dryRun := flag.Bool("dry-run", false, "Preview changes without committing")
	generateEmbeddings := flag.Bool("embeddings", false, "Generate embeddings for tags (requires embedding API)")
	flag.Parse()

	log.Println("Starting tag category migration...")

	// Load config
	if err := config.LoadConfig("./configs"); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
	}

	// Initialize database
	if err := database.InitDB(config.AppConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migration
	result, err := migrateTags(*dryRun, *generateEmbeddings)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Print summary
	fmt.Println("\n=== Migration Summary ===")
	fmt.Printf("Total tags: %d\n", result.total)
	fmt.Printf("Updated: %d\n", result.updated)
	fmt.Printf("Skipped (no change): %d\n", result.skipped)
	fmt.Printf("Errors: %d\n", result.errors)

	if *generateEmbeddings {
		fmt.Printf("Embeddings generated: %d\n", result.embeddings)
	}

	if *dryRun {
		fmt.Println("\n*** DRY RUN - No changes were committed ***")
	}

	fmt.Println("\nMigration completed successfully")
}
