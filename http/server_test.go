package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"io/ioutil"
	"time"

	"github.com/caalberts/localroast/types"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	port := "8888"
	server := NewServer(port).(*server)
	assert.Equal(t, ":8888", server.Server.Addr)
}

func TestServer_Watch(t *testing.T) {
	server := &server{router: newRouter()}
	schemaChannel := make(chan []types.Schema)
	server.Watch(schemaChannel)

	t.Run("new server has empty router", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()
		server.router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusNotImplemented, resp.Code)
	})

	t.Run("with updated schema", func(t *testing.T) {
		schemas := []types.Schema{
			{
				Method:   "GET",
				Path:     "/",
				Status:   200,
				Response: []byte(`{"success": true}`),
			},
			{
				Method:   "GET",
				Path:     "/users",
				Status:   200,
				Response: []byte(`{"success": true, "ids": [1, 2]}`),
			},
			{
				Method:   "POST",
				Path:     "/users",
				Status:   201,
				Response: []byte(`{"success": true, "id": 3}`),
			},
		}
		schemaChannel <- schemas
		time.Sleep(1 * time.Millisecond)

		type testData struct {
			Success bool   `json:"success"`
			ID      int    `json:"id"`
			IDs     []int  `json:"ids"`
			Message string `json:"message"`
		}

		var expected, actual testData
		for _, schema := range schemas {
			t.Run(schema.Method+schema.Path, func(t *testing.T) {
				req := httptest.NewRequest(schema.Method, schema.Path, nil)
				resp := httptest.NewRecorder()
				server.router.ServeHTTP(resp, req)

				assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
				assert.Equal(t, schema.Status, resp.Code)

				body := resp.Result().Body
				defer body.Close()
				json.NewDecoder(body).Decode(&actual)
				json.Unmarshal(schema.Response, &expected)
				assert.Equal(t, expected, actual)
			})
		}

		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		resp := httptest.NewRecorder()
		server.router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
		body := resp.Result().Body
		defer body.Close()
		bodyBytes, _ := ioutil.ReadAll(body)
		assert.Equal(t, "404 page not found\n", string(bodyBytes))
	})
}

func TestPathParam(t *testing.T) {
	schema := types.Schema{
		Method:   "GET",
		Path:     "/users/:id",
		Status:   200,
		Response: []byte(`{"success": true}`),
	}

	router := newRouter()
	router.updateSchema([]types.Schema{schema})

	testPath := "/users/1"

	req := httptest.NewRequest(schema.Method, testPath, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, schema.Status, resp.Code)
}

func TestPrettyJSON(t *testing.T) {
	schema := types.Schema{
		Method:   "GET",
		Path:     "/users",
		Status:   200,
		Response: []byte(`{"success": true}`),
	}
	router := newRouter()
	router.updateSchema([]types.Schema{schema})
	testPath := "/users?pretty"
	req := httptest.NewRequest(schema.Method, testPath, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, schema.Status, resp.Code)
	assert.JSONEq(t, string(schema.Response), resp.Body.String())
}
