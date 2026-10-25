package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/pz6-gorm/internal/models"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Handlers struct{ db *gorm.DB }

func NewHandlers(db *gorm.DB) *Handlers { return &Handlers{db: db} }

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.Email) == "" {
		writeErr(w, http.StatusBadRequest, "name and email are required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	u := models.User{Name: strings.TrimSpace(in.Name), Email: strings.TrimSpace(in.Email)}
	if err := h.db.WithContext(ctx).Create(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			writeErr(w, http.StatusConflict, "email already exists")
		} else {
			writeErr(w, http.StatusInternalServerError, "database error")
		}
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *Handlers) CreateNote(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		UserID  uint     `json:"userId"`
		Tags    []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Title) == "" || in.UserID == 0 {
		writeErr(w, http.StatusBadRequest, "title and userId are required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	note := models.Note{Title: strings.TrimSpace(in.Title), Content: in.Content, UserID: in.UserID}
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, in.UserID).Error; err != nil {
			return err
		}
		seen := make(map[string]bool)
		for _, name := range in.Tags {
			name = strings.TrimSpace(name)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			tag := models.Tag{Name: name}
			if err := tx.FirstOrCreate(&tag, models.Tag{Name: name}).Error; err != nil {
				return err
			}
			note.Tags = append(note.Tags, tag)
		}
		return tx.Create(&note).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, gorm.ErrForeignKeyViolated) {
			writeErr(w, http.StatusBadRequest, "user does not exist")
		} else {
			writeErr(w, http.StatusBadRequest, "cannot create note")
		}
		return
	}
	if err := h.db.WithContext(ctx).Preload("User").Preload("Tags").First(&note, note.ID).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

func (h *Handlers) GetNoteByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, "bad id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var note models.Note
	err = h.db.WithContext(ctx).Preload("User").Preload("Tags").First(&note, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		writeErr(w, http.StatusNotFound, "note not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}
