package admin

import (
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	passwordpkg "github.com/thomiceli/opengist/internal/auth/password"
	"github.com/thomiceli/opengist/internal/db"
	"github.com/thomiceli/opengist/internal/validator"
	"github.com/thomiceli/opengist/internal/web/context"
)

var (
	reservedNameValidator = validator.NewValidator()
	usernameRe            = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	emailRe               = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type BulkResult struct {
	Line     int
	Username string
	Success  bool
	Error    string
}

func AdminCreateUsers(ctx *context.Context) error {
	ctx.SetData("htmlTitle", ctx.TrH("admin.create-users.title")+" - "+ctx.TrH("admin.admin_panel"))
	ctx.SetData("adminHeaderPage", "create-users")
	return ctx.Html("admin_create_users.html")
}

func AdminCreateUserProcess(ctx *context.Context) error {
	ctx.SetData("htmlTitle", ctx.TrH("admin.create-users.title")+" - "+ctx.TrH("admin.admin_panel"))
	ctx.SetData("adminHeaderPage", "create-users")

	username := strings.TrimSpace(ctx.FormValue("username"))
	email := strings.TrimSpace(ctx.FormValue("email"))
	password := ctx.FormValue("password")

	if err := validateUserFields(username, email, password); err != "" {
		ctx.AddFlash(ctx.Tr(err), "error")
		return ctx.RedirectTo("/admin-panel/create-users")
	}

	if exists, err := db.UserExists(username); err != nil {
		return ctx.ErrorRes(500, "Cannot check user existence", err)
	} else if exists {
		ctx.AddFlash(ctx.Tr("flash.auth.username-exists"), "error")
		return ctx.RedirectTo("/admin-panel/create-users")
	}

	if err := createUser(username, email, password); err != nil {
		return ctx.ErrorRes(500, "Cannot create user", err)
	}

	ctx.AddFlash(ctx.Tr("flash.admin.user-created", username), "success")
	return ctx.RedirectTo("/admin-panel/create-users")
}

func AdminCreateUsersBulkProcess(ctx *context.Context) error {
	ctx.SetData("htmlTitle", ctx.TrH("admin.create-users.title")+" - "+ctx.TrH("admin.admin_panel"))
	ctx.SetData("adminHeaderPage", "create-users")

	csvdata := ctx.FormValue("csvdata")
	if strings.TrimSpace(csvdata) == "" {
		ctx.AddFlash(ctx.Tr("admin.create-users.error.csv-empty"), "error")
		return ctx.RedirectTo("/admin-panel/create-users")
	}

	reader := csv.NewReader(strings.NewReader(csvdata))
	reader.FieldsPerRecord = -1 // allow variable fields
	records, err := reader.ReadAll()
	if err != nil {
		ctx.AddFlash(ctx.Tr("admin.create-users.error.csv-parse", err.Error()), "error")
		return ctx.RedirectTo("/admin-panel/create-users")
	}

	var results []BulkResult
	created := 0
	failed := 0

	for i, record := range records {
		lineNum := i + 1

		// Skip empty lines
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		if len(record) < 3 {
			results = append(results, BulkResult{
				Line:     lineNum,
				Username: safeGet(record, 0),
				Success:  false,
				Error:    ctx.Tr("admin.create-users.error.csv-fields"),
			})
			failed++
			continue
		}

		username := strings.TrimSpace(record[0])
		email := strings.TrimSpace(record[1])
		password := strings.TrimSpace(record[2])

		if validationErr := validateUserFields(username, email, password); validationErr != "" {
			results = append(results, BulkResult{
				Line:     lineNum,
				Username: username,
				Success:  false,
				Error:    ctx.Tr(validationErr),
			})
			failed++
			continue
		}

		if exists, err := db.UserExists(username); err != nil {
			results = append(results, BulkResult{
				Line:     lineNum,
				Username: username,
				Success:  false,
				Error:    ctx.Tr("admin.create-users.error.database", err.Error()),
			})
			failed++
			continue
		} else if exists {
			results = append(results, BulkResult{
				Line:     lineNum,
				Username: username,
				Success:  false,
				Error:    ctx.Tr("flash.auth.username-exists"),
			})
			failed++
			continue
		}

		if err := createUser(username, email, password); err != nil {
			results = append(results, BulkResult{
				Line:     lineNum,
				Username: username,
				Success:  false,
				Error:    ctx.Tr("admin.create-users.error.create-failed", err.Error()),
			})
			failed++
			continue
		}

		results = append(results, BulkResult{
			Line:     lineNum,
			Username: username,
			Success:  true,
		})
		created++
	}

	ctx.SetData("bulkResults", results)
	ctx.AddFlash(ctx.Tr("admin.create-users.bulk.results",
		strconv.Itoa(created), strconv.Itoa(failed)), "success")
	return ctx.Html("admin_create_users.html")
}

// validateUserFields checks username, email, and password and returns the i18n key of the
// error message, or an empty string if the fields are valid.
func validateUserFields(username, email, password string) string {
	if username == "" {
		return "admin.create-users.error.username-required"
	}
	if len(username) > 24 {
		return "admin.create-users.error.username-too-long"
	}
	if !usernameRe.MatchString(username) {
		return "admin.create-users.error.username-invalid"
	}
	if reservedNameValidator.Var(username, "notreserved") != nil {
		return "admin.create-users.error.username-reserved"
	}
	if password == "" {
		return "admin.create-users.error.password-required"
	}
	if len(password) < 6 {
		return "admin.create-users.error.password-too-short"
	}
	if email != "" {
		if !emailRe.MatchString(email) {
			return "admin.create-users.error.email-invalid"
		}
	}
	return ""
}

// createUser hashes the password and creates the user in the database.
func createUser(username, email, password string) error {
	hashedPassword, err := passwordpkg.HashPassword(password)
	if err != nil {
		return err
	}

	var hash string
	if email == "" {
		hash = fmt.Sprintf("%x", md5.Sum([]byte(time.Now().String()+username)))
	} else {
		hash = fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email)))))
	}

	user := &db.User{
		Username: username,
		Password: hashedPassword,
		Email:    strings.ToLower(email),
		MD5Hash:  hash,
	}

	return user.Create()
}

func safeGet(s []string, i int) string {
	if i < len(s) {
		return strings.TrimSpace(s[i])
	}
	return ""
}
