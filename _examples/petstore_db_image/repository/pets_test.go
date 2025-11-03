package repository

import (
	"context"
	"github.com/gofrs/uuid"
	"testing"
	"time"
)

func TestRepository_ListPets(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, _ = testDb.Exec("DELETE FROM pets")

		result, err := testRepo.ListPets(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Error("Expected empty result")
		}
	})
	t.Run("non-empty", func(t *testing.T) {
		_, _ = testDb.Exec("DELETE FROM pets")
		id, _ := uuid.NewV4()
		insertDb(t, "pets", map[string]any{
			"id":          id.String(),
			"name":        "Felix",
			"dob":         time.Now(),
			"category_id": rawQuery("(SELECT id from categories WHERE name = 'Cats')"),
		})

		result, err := testRepo.ListPets(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 {
			t.Error("Expected non-empty result")
		}
	})
}

func TestRepository_GetPet(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		result, err := testRepo.GetPet(context.Background(), "non-existent-id")
		if err != nil {
			t.Fatal(err)
		}
		if result != nil {
			t.Error("Expected nil result")
		}
	})
	t.Run("found", func(t *testing.T) {
		id, _ := uuid.NewV4()
		insertDb(t, "pets", map[string]any{
			"id":          id.String(),
			"name":        "Felix",
			"dob":         time.Now(),
			"category_id": rawQuery("(SELECT id from categories WHERE name = 'Cats')"),
		})
		result, err := testRepo.GetPet(context.Background(), id.String())
		if err != nil {
			t.Fatal(err)
		}
		if result == nil {
			t.Error("Expected result")
		}
	})
}
