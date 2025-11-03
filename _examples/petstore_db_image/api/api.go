package api

import (
	"app/config"
	"app/models"
	"app/repository"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
)

type Api interface {
	Start()
}

func NewApi(cfg config.Api, repo repository.Repository) Api {
	return &api{
		cfg:  cfg,
		repo: repo,
	}
}

type api struct {
	cfg  config.Api
	repo repository.Repository
}

const uuidPattern = "^([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12})$"

func (a *api) Start() {
	router := chi.NewRouter()
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	router.Get("/api", a.getRoot)
	router.Post("/api/pets", a.postPets)
	router.Get("/api/pets", a.getPets)
	router.Get("/api/pets/{id:"+uuidPattern+"}", a.getPet)
	//router.Put("/api/pets/{id:"+uuidPattern+"}", a.putPet)
	router.Delete("/api/pets/{id:"+uuidPattern+"}", a.deletePet)
	router.Get("/api/categories", a.getCategories)
	router.Get("/api/categories/{id:"+uuidPattern+"}", a.getCategory)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", a.cfg.Port), router); err != nil {
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
	if result, err := a.repo.ListPets(r.Context()); err == nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	} else {
		fmt.Println("ERROR!!!", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	}
}

func (a *api) postPets(w http.ResponseWriter, r *http.Request) {
	var err error
	var status int
	add := models.AddPet{}
	if err = json.NewDecoder(r.Body).Decode(&add); err == nil {
		var result map[string]any
		if result, err = a.repo.AddPet(r.Context(), add); err == nil {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(result)
		}
	} else {
		status = http.StatusBadRequest
	}
	if err != nil {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	}
}

func (a *api) getPet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if result, err := a.repo.GetPet(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	} else if result == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func (a *api) deletePet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if ok, err := a.repo.DeletePet(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	} else if !ok {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (a *api) getCategories(w http.ResponseWriter, r *http.Request) {
	if result, err := a.repo.ListCategories(r.Context()); err == nil {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	}
}

func (a *api) getCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if result, err := a.repo.GetCategory(r.Context(), id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
	} else if result == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}
}
