package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	r, _ := http.NewRequest("GET", "/wather/{cep}", nil)
	w := httptest.NewRecorder()

	vars := map[string]string{
		"cep": "04848400",
	}

	r = mux.SetURLVars(r, vars)
	h := NewHandler()
	h.GetWeatherHandler(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestInvalidCEP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/wather/{cep}", nil)
	w := httptest.NewRecorder()

	vars := map[string]string{
		"cep": "123-1213",
	}

	r = mux.SetURLVars(r, vars)

	h := NewHandler()
	h.GetWeatherHandler(w, r)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestNoutFoundCEP(t *testing.T) {
	r, _ := http.NewRequest("GET", "/wather/{cep}", nil)
	w := httptest.NewRecorder()

	vars := map[string]string{
		"cep": "00000000",
	}

	r = mux.SetURLVars(r, vars)

	h := NewHandler()
	h.GetWeatherHandler(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
