package gist_test

import (
	"io"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/git"
	webtest "github.com/thomiceli/opengist/internal/web/test"
)

func TestCollaborators(t *testing.T) {
	s := webtest.Setup(t)
	defer webtest.Teardown(t)

	s.Register(t, "alice")
	s.Register(t, "carol")

	_, gist, username, identifier := s.CreateGist(t, "2") // private, owned by thomas
	base := "/" + username + "/" + identifier

	s.Login(t, "thomas")
	s.Request(t, "POST", base+"/collaborators", url.Values{"username": {"alice"}}, 302)
	s.Logout()

	t.Run("Visibility", func(t *testing.T) {
		s.Login(t, "alice")
		s.Request(t, "GET", base, nil, 200)
		s.Login(t, "carol")
		s.Request(t, "GET", base, nil, 404)
		s.Logout()
	})

	t.Run("CollaboratorCannotManage", func(t *testing.T) {
		s.Login(t, "alice")
		s.Request(t, "GET", base+"/collaborators", nil, 403)
		s.Request(t, "POST", base+"/collaborators", url.Values{"username": {"carol"}}, 403)
		s.Request(t, "POST", base+"/visibility", url.Values{"private": {"0"}}, 403)
		s.Request(t, "POST", base+"/delete", nil, 403)
		s.Logout()
	})

	t.Run("ListedForCollaborator", func(t *testing.T) {
		alice, err := db.GetUserByUsername("alice")
		require.NoError(t, err)
		carol, err := db.GetUserByUsername("carol")
		require.NoError(t, err)
		thomas, err := db.GetUserByUsername("thomas")
		require.NoError(t, err)

		aliceGists, err := db.GetAllGistsForCurrentUser(alice.ID, 0, "created", "desc")
		require.NoError(t, err)
		require.Len(t, aliceGists, 1)

		carolGists, err := db.GetAllGistsForCurrentUser(carol.ID, 0, "created", "desc")
		require.NoError(t, err)
		require.Empty(t, carolGists)

		n, err := db.CountAllGistsFromUser(thomas.ID, alice.ID)
		require.NoError(t, err)
		require.EqualValues(t, 1, n)
	})

	t.Run("SearchExcludesExistingAndEscapesWildcards", func(t *testing.T) {
		users, err := db.SearchUsers("a", gist.ID, gist.UserID, 10)
		require.NoError(t, err)
		require.Empty(t, users, "alice is already a collaborator")

		users, err = db.SearchUsers("c", gist.ID, gist.UserID, 10)
		require.NoError(t, err)
		require.Len(t, users, 1)

		users, err = db.SearchUsers("%", gist.ID, gist.UserID, 10)
		require.NoError(t, err)
		require.Empty(t, users, "% must not act as a wildcard")
	})

	t.Run("EditByCollaborator", func(t *testing.T) {
		s.Login(t, "alice")
		resp := s.Request(t, "POST", base+"/edit", url.Values{
			"title":   {"Test"},
			"name":    {"file.txt"},
			"content": {"edited by alice"},
		}, 302)
		require.Equal(t, base, resp.Header.Get("Location"), "redirects to the owner's gist URL")
		s.Logout()

		g, err := db.GetGist(username, identifier)
		require.NoError(t, err)
		commits, err := g.Log(0, git.DiffOptions{})
		require.NoError(t, err)
		require.Equal(t, "alice", commits[0].AuthorName)
	})

	t.Run("RemovedCollaboratorLosesAccess", func(t *testing.T) {
		s.Login(t, "thomas")
		s.Request(t, "POST", base+"/collaborators/remove", url.Values{"username": {"alice"}}, 302)
		s.Logout()

		s.Login(t, "alice")
		s.Request(t, "GET", base, nil, 404)
		s.Logout()
	})
}

func TestComments(t *testing.T) {
	s := webtest.Setup(t)
	defer webtest.Teardown(t)

	s.Register(t, "alice")
	s.Register(t, "carol")

	_, gist, username, identifier := s.CreateGist(t, "2")
	base := "/" + username + "/" + identifier

	s.Login(t, "thomas")
	s.Request(t, "POST", base+"/collaborators", url.Values{"username": {"alice"}}, 302)
	s.Logout()

	t.Run("CollaboratorCanComment", func(t *testing.T) {
		s.Login(t, "alice")
		s.Request(t, "POST", base+"/comments", url.Values{"content": {"hello"}}, 302)
		s.Logout()
	})

	t.Run("NonCollaboratorCannotCommentOnPrivate", func(t *testing.T) {
		s.Login(t, "carol")
		s.Request(t, "POST", base+"/comments", url.Values{"content": {"hello"}}, 404)
		s.Logout()
	})

	comments, err := db.GetCommentsByGist(gist.ID, "")
	require.NoError(t, err)
	require.Len(t, comments, 1)
	deleteUrl := base + "/comments/" + strconv.Itoa(int(comments[0].ID)) + "/delete"

	t.Run("OtherUserCannotDelete", func(t *testing.T) {
		s.Login(t, "carol")
		s.Request(t, "POST", deleteUrl, nil, 404)
		s.Logout()
	})

	t.Run("GistOwnerCanDelete", func(t *testing.T) {
		s.Login(t, "thomas")
		s.Request(t, "POST", deleteUrl, nil, 302)
		s.Logout()

		comments, err := db.GetCommentsByGist(gist.ID, "")
		require.NoError(t, err)
		require.Empty(t, comments)
	})

	t.Run("DeletingGistRemovesComments", func(t *testing.T) {
		s.Login(t, "thomas")
		s.Request(t, "POST", base+"/comments", url.Values{"content": {"mine"}}, 302)
		s.Request(t, "POST", base+"/delete", nil, 302)
		s.Logout()

		comments, err := db.GetCommentsByGist(gist.ID, "")
		require.NoError(t, err)
		require.Empty(t, comments)
	})
}

func TestCollaboratorLeave(t *testing.T) {
	s := webtest.Setup(t)
	defer webtest.Teardown(t)

	s.Register(t, "alice")
	s.Register(t, "carol")

	_, gist, username, identifier := s.CreateGist(t, "2")
	base := "/" + username + "/" + identifier

	s.Login(t, "thomas")
	s.Request(t, "POST", base+"/collaborators", url.Values{"username": {"alice"}}, 302)
	s.Logout()

	t.Run("NonCollaboratorLeaveIsNoop", func(t *testing.T) {
		s.Login(t, "carol")
		s.Request(t, "POST", base+"/collaborators/leave", nil, 404) // cannot see the private gist
		s.Logout()
	})

	t.Run("CollaboratorLeaves", func(t *testing.T) {
		s.Login(t, "alice")
		resp := s.Request(t, "POST", base+"/collaborators/leave", nil, 302)
		require.Equal(t, "/all", resp.Header.Get("Location"))
		s.Request(t, "GET", base, nil, 404)
		s.Logout()

		alice, err := db.GetUserByUsername("alice")
		require.NoError(t, err)
		isCollab, err := db.IsCollaborator(gist.ID, alice.ID)
		require.NoError(t, err)
		require.False(t, isCollab)
	})
}

func TestRevisionComments(t *testing.T) {
	s := webtest.Setup(t)
	defer webtest.Teardown(t)

	_, gist, username, identifier := s.CreateGist(t, "0")
	base := "/" + username + "/" + identifier

	s.Login(t, "thomas")
	// second commit so that there is an older revision to comment on
	s.Request(t, "POST", base+"/edit", url.Values{"title": {"Test"}, "name": {"file.txt"}, "content": {"v2"}}, 302)

	commits, err := gist.Log(0, git.DiffOptions{})
	require.NoError(t, err)
	require.Len(t, commits, 2)
	oldHash := commits[1].Hash
	shortHash := oldHash[:7]

	t.Run("CommentViaShortHashIsStoredAsFullHash", func(t *testing.T) {
		s.Request(t, "POST", base+"/comments", url.Values{"content": {"old rev"}, "revision": {shortHash}}, 302)
		comments, err := db.GetCommentsByGist(gist.ID, oldHash)
		require.NoError(t, err)
		require.Len(t, comments, 1)
		require.Equal(t, oldHash, comments[0].Revision)
	})

	t.Run("ShownOnShortAndFullHashPages", func(t *testing.T) {
		for _, rev := range []string{shortHash, oldHash} {
			resp := s.Request(t, "GET", base+"/rev/"+rev, nil, 200)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "old rev", "revision %s", rev)
		}
	})

	t.Run("InvalidRevisionIsRejectedNotGistWide", func(t *testing.T) {
		resp := s.Request(t, "POST", base+"/comments", url.Values{"content": {"bogus"}, "revision": {"does-not-exist"}}, 302)
		require.Equal(t, base, resp.Header.Get("Location"))
		comments, err := db.GetCommentsByGist(gist.ID, "")
		require.NoError(t, err)
		for _, c := range comments {
			require.NotEqual(t, "bogus", c.Content)
		}
	})

	t.Run("UnknownRevisionPageIs404", func(t *testing.T) {
		s.Request(t, "GET", base+"/rev/deadbeef", nil, 404)
	})
}
