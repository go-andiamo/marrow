package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-andiamo/columbus"
)

type categories interface {
	ListCategories(ctx context.Context) ([]map[string]any, error)
	GetCategory(ctx context.Context, id string) (map[string]any, error)
}

var categoriesMapper = columbus.MustNewMapper(`id,name`,
	columbus.Query(`FROM categories`),
	columbus.Mappings{
		"id": {PostProcess: func(ctx context.Context, sqli columbus.SqlInterface, row map[string]any, value any) (replace bool, replaceValue any, err error) {
			row["$ref"] = fmt.Sprintf("/api/categories/%v", value)
			return false, nil, nil
		}},
	})

func (r *repository) ListCategories(ctx context.Context) ([]map[string]any, error) {
	return categoriesMapper.Rows(ctx, r.db, nil, columbus.AddClause("ORDER BY name"))
}

func (r *repository) GetCategory(ctx context.Context, id string) (map[string]any, error) {
	if result, err := categoriesMapper.ExactlyOneRow(ctx, r.db, []any{id}, columbus.AddClause("WHERE id = ?")); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	} else {
		return result, nil
	}
}
