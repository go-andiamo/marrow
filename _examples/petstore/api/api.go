package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/gofrs/uuid"
	"net/http"
	"os"
	"sync"
)

type api struct {
	mutex      sync.RWMutex
	pets       map[string]map[string]any
	categories map[string]map[string]any
}

type Api interface {
	Start(port int)
}

func NewApi() Api {
	catId := "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	return &api{
		pets: make(map[string]map[string]any),
		categories: map[string]map[string]any{
			catId: {
				"$ref": "/api/categories/" + catId,
				"id":   catId,
				"name": "Cats",
			},
		},
	}
}

const uuidPattern = "^([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12})$"

func (a *api) Start(port int) {
	router := chi.NewRouter()
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	router.Get("/api", a.getRoot)
	router.Get("/api/pets", a.getPets)
	router.Post("/api/pets", a.postPets)
	router.Get("/api/pets/{id:"+uuidPattern+"}", a.getPet)
	router.Put("/api/pets/{id:"+uuidPattern+"}", a.putPet)
	router.Delete("/api/pets/{id:"+uuidPattern+"}", a.deletePet)
	router.Get("/api/categories", a.getCategories)
	router.Get("/api/categories/{id:"+uuidPattern+"}", a.getCategory)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), router); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}
}

func (a *api) getRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"hello": "world",
	})
}

func (a *api) getPets(w http.ResponseWriter, r *http.Request) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	w.WriteHeader(http.StatusOK)
	result := make([]any, 0, len(a.pets))
	for _, pet := range a.pets {
		result = append(result, pet)
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (a *api) postPets(w http.ResponseWriter, r *http.Request) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	var err error
	status := http.StatusCreated
	var pet map[string]any
	reqBody := map[string]any{}
	if err = json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
		var category map[string]any
		if category, err = a.categoryFromRequest(reqBody); err == nil {
			id, _ := uuid.NewV4()
			pet = map[string]any{
				"$ref":     "/api/pets/" + id.String(),
				"id":       id.String(),
				"name":     reqBody["name"],
				"category": category,
			}
			a.pets[id.String()] = pet
		} else {
			status = http.StatusUnprocessableEntity
		}
	} else {
		status = http.StatusBadRequest
	}
	if err == nil {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(pet)
	} else {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	}
}

func (a *api) getPet(w http.ResponseWriter, r *http.Request) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	id := chi.URLParam(r, "id")
	if pet, ok := a.pets[id]; ok {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(pet)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) putPet(w http.ResponseWriter, r *http.Request) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	id := chi.URLParam(r, "id")
	if pet, ok := a.pets[id]; ok {
		reqBody := map[string]any{}
		var err error
		if err = json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			var category map[string]any
			if category, err = a.categoryFromRequest(reqBody); err == nil {
				w.WriteHeader(http.StatusOK)
				pet["name"] = reqBody["name"]
				pet["dob"] = reqBody["dob"]
				pet["category"] = category
				_ = json.NewEncoder(w).Encode(pet)
			}
		}
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": err.Error(),
			})
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) deletePet(w http.ResponseWriter, r *http.Request) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	id := chi.URLParam(r, "id")
	if _, ok := a.pets[id]; ok {
		delete(a.pets, id)
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) getCategories(w http.ResponseWriter, r *http.Request) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	w.WriteHeader(http.StatusOK)
	result := make([]any, 0, len(a.categories))
	for _, category := range a.categories {
		result = append(result, category)
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (a *api) getCategory(w http.ResponseWriter, r *http.Request) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	id := chi.URLParam(r, "id")
	if category, ok := a.categories[id]; ok {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(category)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) categoryFromRequest(req map[string]any) (category map[string]any, err error) {
	if c, ok := req["category"]; ok {
		if cm, ok := c.(map[string]any); ok {
			if id, ok := stringProperty(cm, "id"); ok {
				if category, ok = a.categories[id]; ok {
					return category, nil
				} else {
					return nil, errors.New("category not found")
				}
			} else if name, ok := stringProperty(cm, "name"); ok {
				for _, cat := range a.categories {
					if cat["name"].(string) == name {
						return cat, nil
					}
				}
				return nil, errors.New("category not found")
			}
		}
		return nil, errors.New("invalid category property")
	} else {
		return nil, errors.New("category must be specified")
	}
}

func stringProperty(m map[string]any, pty string) (string, bool) {
	if v, ok := m[pty]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}
