package auth_middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EvolutionAPI/evolution-go/pkg/config"
	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	instance_service "github.com/EvolutionAPI/evolution-go/pkg/instance/service"
	"github.com/gin-gonic/gin"
)

type instanceTokenService struct {
	instance_service.InstanceService
	instance *instance_model.Instance
}

func (s instanceTokenService) GetInstanceByToken(token string) (*instance_model.Instance, error) {
	if token == s.instance.Token {
		return s.instance, nil
	}
	return nil, errors.New("unknown instance token")
}

func TestAdminAuthenticationStillRequiresGlobalKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	middleware := NewMiddleware(&config.Config{GlobalApiKey: "admin-secret"}, nil)
	router := gin.New()
	router.GET("/admin", middleware.AuthAdmin, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, test := range []struct {
		key  string
		want int
	}{
		{"", http.StatusUnauthorized},
		{"wrong-key", http.StatusUnauthorized},
		{"instance-secret", http.StatusUnauthorized},
		{"admin-secret", http.StatusNoContent},
	} {
		t.Run(test.key, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/admin", nil)
			request.Header.Set("apikey", test.key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("got HTTP %d; want %d", response.Code, test.want)
			}
		})
	}
}

func TestInstanceAuthenticationRequiresInstanceToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	instance := &instance_model.Instance{Token: "instance-secret"}
	middleware := NewMiddleware(&config.Config{GlobalApiKey: "admin-secret"}, instanceTokenService{instance: instance})
	router := gin.New()
	router.GET("/instance", middleware.Auth, func(c *gin.Context) {
		if c.MustGet("instance") != instance {
			t.Error("authenticated instance was not passed to the handler")
		}
		c.Status(http.StatusNoContent)
	})
	for _, test := range []struct {
		key  string
		want int
	}{
		{"", http.StatusUnauthorized},
		{"wrong-key", http.StatusUnauthorized},
		{"admin-secret", http.StatusUnauthorized},
		{"instance-secret", http.StatusNoContent},
	} {
		t.Run(test.key, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/instance", nil)
			request.Header.Set("apikey", test.key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("got HTTP %d; want %d", response.Code, test.want)
			}
		})
	}
}
