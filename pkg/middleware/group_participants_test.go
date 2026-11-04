package auth_middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGroupParticipantArrayValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		body string
		want int
	}{
		{`{"groupJid":"120363000000000000@g.us","participants":["5511999999999","123456789012345@lid"]}`, 204},
		{`{"groupJid":"120363000000000000@g.us","participants":"5511999999999"}`, 400},
		{`{"participants":[]}`, 400}, {`{"participants":[""]}`, 400},
		{`{"participants":[42]}`, 400}, {`{"participants":[null]}`, 400},
		{`{"participants":null}`, 400}, {`{}`, 400}, {`null`, 400},
	} {
		t.Run(test.body, func(t *testing.T) {
			m := NewJIDValidationMiddleware()
			router := gin.New()
			router.POST("/group/participant", m.ValidateJIDFields("groupJid"), m.ValidateMultipleNumbers("participants"), func(c *gin.Context) {
				var data struct {
					GroupJID     string   `json:"groupJid"`
					Participants []string `json:"participants"`
				}
				if err := c.ShouldBindJSON(&data); err != nil {
					t.Fatal(err)
				}
				if data.GroupJID != "120363000000000000@g.us" || len(data.Participants) != 2 || data.Participants[1] != "123456789012345@lid" {
					t.Fatalf("middleware lost the group or participants: %#v", data)
				}
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/group/participant", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("got %d, want %d: %s", response.Code, test.want, response.Body.String())
			}
		})
	}
}
