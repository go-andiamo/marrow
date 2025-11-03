package repository

import (
	"context"
	"testing"
)

func TestRepository_ListCategories(t *testing.T) {
	result, err := testRepo.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Error("Expected some categories")
	}
}

func TestRepository_GetCategory(t *testing.T) {
	t.Run("Category not exists", func(t *testing.T) {
		result, err := testRepo.GetCategory(context.Background(), "non-existent")
		if err != nil {
			t.Fatal(err)
		}
		if result != nil {
			t.Error("Expected no result for not found category")
		}
	})
	t.Run("Category exists", func(t *testing.T) {
		id := ""
		err := testDb.QueryRow("SELECT id FROM categories").Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		result, err := testRepo.GetCategory(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if result == nil {
			t.Error("Expected found category")
		}
	})
}
