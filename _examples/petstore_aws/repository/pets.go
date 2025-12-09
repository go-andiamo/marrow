package repository

import (
	"appaws/models"
	"context"
	"database/sql"
	"fmt"
	"github.com/go-andiamo/columbus"
	"github.com/gofrs/uuid"
)

type pets interface {
	ListPets(ctx context.Context) ([]map[string]any, error)
	GetPet(ctx context.Context, id string) (map[string]any, error)
	AddPet(ctx context.Context, req models.AddPet) (map[string]any, error)
	DeletePet(ctx context.Context, id string) (bool, error)
}

var petsMapper = columbus.MustNewMapper(`pets.id,pets.name,pets.dob,categories.id AS category_id,categories.name AS category_name`,
	columbus.Query(`FROM pets
		JOIN categories ON categories.id = pets.category_id`),
	columbus.Mappings{
		"id": {PostProcess: func(ctx context.Context, sqli columbus.SqlInterface, row map[string]any, value any) (replace bool, replaceValue any, err error) {
			row["$ref"] = fmt.Sprintf("/api/pets/%v", value)
			return false, nil, nil
		}},
		"category_id": {
			Path:         []string{"category"},
			PropertyName: "id",
		},
		"category_name": {
			Path:         []string{"category"},
			PropertyName: "name",
		},
	})

func (r *repository) ListPets(ctx context.Context) ([]map[string]any, error) {
	return petsMapper.Rows(ctx, r.db, nil, columbus.AddClause("ORDER BY name"))
}

func (r *repository) GetPet(ctx context.Context, id string) (map[string]any, error) {
	u, _ := uuid.NewV4()
	_ = u
	if result, err := petsMapper.ExactlyOneRow(ctx, r.db, []any{id}, columbus.AddClause("WHERE pets.id = ?")); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	} else {
		return result, nil
	}
}

func (r *repository) AddPet(ctx context.Context, req models.AddPet) (map[string]any, error) {
	const queryInsert = `INSERT INTO pets (id, name, dob, category_id) VALUES (?, ?, ?, (SELECT id FROM categories WHERE id = ? OR name = ?))`
	id, _ := uuid.NewV4()
	args := []any{
		id.String(),
		req.Name,
		req.Dob,
		req.Category.Id,
		req.Category.Name,
	}
	if _, err := r.db.ExecContext(ctx, queryInsert, args...); err != nil {
		fmt.Println("ERROR", err.Error())
		return nil, err
	}
	return r.GetPet(ctx, id.String())
}

func (r *repository) DeletePet(ctx context.Context, id string) (bool, error) {
	const queryDelete = `DELETE FROM pets WHERE id = ?`
	if res, err := r.db.ExecContext(ctx, queryDelete, id); err != nil {
		return false, err
	} else if rowsAffected, err := res.RowsAffected(); err == nil {
		return rowsAffected == 1, nil
	}
	return true, nil
}
