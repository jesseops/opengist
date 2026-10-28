package admin_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thomiceli/opengist/internal/db"
	webtest "github.com/thomiceli/opengist/internal/web/test"
)

func TestAdminCreateUsers(t *testing.T) {
	s := webtest.Setup(t)
	defer webtest.Teardown(t)

	s.Register(t, "thomas") // first user is admin
	s.Register(t, "nonadmin")

	t.Run("NonAdminForbidden", func(t *testing.T) {
		s.Login(t, "nonadmin")
		s.Request(t, "GET", "/admin-panel/create-users", nil, 404)
		s.Request(t, "POST", "/admin-panel/create-users", url.Values{"username": {"x"}, "password": {"secret1"}}, 404)
		s.Logout()
	})

	s.Login(t, "thomas")

	t.Run("CreateUser", func(t *testing.T) {
		s.Request(t, "POST", "/admin-panel/create-users", url.Values{
			"username": {"newuser"}, "email": {"new@example.com"}, "password": {"secret1"},
		}, 302)
		exists, err := db.UserExists("newuser")
		require.NoError(t, err)
		require.True(t, exists)
	})

	t.Run("RejectsReservedAndInvalidNames", func(t *testing.T) {
		for _, name := range []string{"login", "admin-panel", "settings", "bad name", "has_underscore"} {
			s.Request(t, "POST", "/admin-panel/create-users", url.Values{
				"username": {name}, "password": {"secret1"},
			}, 302)
			exists, err := db.UserExists(name)
			require.NoError(t, err)
			require.False(t, exists, "%q must not be created", name)
		}
	})

	t.Run("Bulk", func(t *testing.T) {
		s.Request(t, "POST", "/admin-panel/create-users/bulk", url.Values{
			"csvdata": {"bulk1,b1@example.com,secret1\nlogin,,secret1\nbulk2,,short"},
		}, 200)
		for name, want := range map[string]bool{"bulk1": true, "login": false, "bulk2": false} {
			exists, err := db.UserExists(name)
			require.NoError(t, err)
			require.Equal(t, want, exists, name)
		}
	})
}
