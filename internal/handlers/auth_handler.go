package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"eglise_ujn/internal/config"
	"eglise_ujn/internal/httputil"
	"eglise_ujn/internal/middlewares"
	"eglise_ujn/internal/models"
	"eglise_ujn/internal/services"
	"eglise_ujn/templates/pages"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth  *services.AuthService
	users *services.UserService
}

func NewAuthHandler(auth *services.AuthService, users *services.UserService) *AuthHandler {
	return &AuthHandler{auth: auth, users: users}
}

func (h *AuthHandler) LoginPage(c *gin.Context) {
	csrf := middlewares.SetCSRFCookie(c)
	httputil.Render(c, http.StatusOK, pages.Login(c.Query("alert"), csrf))
}

func (h *AuthHandler) LoginPost(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	_, token, err := h.auth.Login(c.Request.Context(), email, password)
	if err != nil {
		c.Redirect(http.StatusFound, "/login?alert=Identifiants+invalides")
		return
	}
	jcfg, err := config.LoadJWTConfig()
	if err != nil {
		c.Redirect(http.StatusFound, "/login?alert=Config+JWT")
		return
	}
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie(jcfg.CookieName, token, int(jcfg.TTL.Seconds()), jcfg.CookiePath, "", secure, jcfg.CookieHTTPOnly)
	c.Redirect(http.StatusFound, "/dashboard")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	jcfg, _ := config.LoadJWTConfig()
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetCookie(jcfg.CookieName, "", -1, jcfg.CookiePath, "", secure, true)
	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) ProfilePasswordPage(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	httputil.Render(c, http.StatusOK, pages.ProfilePasswordForm(layoutFromCtx(c, "Modifier mon mot de passe", ""), user))
}

func (h *AuthHandler) ProfileUpdatePassword(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	current := c.PostForm("current_password")
	password := c.PostForm("password")
	confirm := c.PostForm("password_confirm")
	if err := h.auth.VerifyPassword(c.Request.Context(), user.ID, current); err != nil {
		c.Redirect(http.StatusFound, "/profile/password?alert=Mot+de+passe+actuel+incorrect")
		return
	}
	if len(password) < 8 {
		c.Redirect(http.StatusFound, "/profile/password?alert=Mot+de+passe+trop+court")
		return
	}
	if password != confirm {
		c.Redirect(http.StatusFound, "/profile/password?alert=Mots+de+passe+non+identiques")
		return
	}
	if err := h.users.UpdatePassword(c.Request.Context(), user.ID, password, user.FullName()); err != nil {
		c.Redirect(http.StatusFound, "/profile/password?alert=Erreur+mise+a+jour")
		return
	}
	c.Redirect(http.StatusFound, "/profile/password?alert=Mot+de+passe+mis+a+jour")
}

type DashboardHandler struct {
	dash *services.DashboardService
}

func NewDashboardHandler(dash *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dash: dash}
}

func (h *DashboardHandler) Index(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	var userID uint
	if user != nil {
		userID = user.ID
	}
	m, _ := h.dash.Metrics(c.Request.Context(), userID)
	httputil.Render(c, http.StatusOK, pages.Dashboard(layoutFromCtx(c, "Tableau de bord", "dashboard"), m, user))
}

type UsersHandler struct {
	users  *services.UserService
	reqSvc *services.RequisitionService
}

func NewUsersHandler(users *services.UserService, reqSvc *services.RequisitionService) *UsersHandler {
	return &UsersHandler{users: users, reqSvc: reqSvc}
}

func (h *UsersHandler) List(c *gin.Context) {
	list, _ := h.users.List(c.Request.Context())
	httputil.Render(c, http.StatusOK, pages.UsersList(layoutFromCtx(c, "Utilisateurs", "users"), list))
}

func (h *UsersHandler) NewPage(c *gin.Context) {
	httputil.Render(c, http.StatusOK, pages.UserForm(layoutFromCtx(c, "Nouvel utilisateur", "users"), nil))
}

func (h *UsersHandler) Create(c *gin.Context) {
	cur := middlewares.CurrentUser(c)
	roles := parseRoles(c)
	_, err := h.users.Create(c.Request.Context(), services.CreateUserInput{
		FirstName: strings.TrimSpace(c.PostForm("first_name")),
		LastName:  strings.TrimSpace(c.PostForm("last_name")),
		Email:     strings.TrimSpace(c.PostForm("email")),
		Phone:     strings.TrimSpace(c.PostForm("phone")),
		Password:  c.PostForm("password"),
		Roles:     roles,
	}, cur.FullName())
	if err != nil {
		c.Redirect(http.StatusFound, "/users/new?alert=Erreur+creation")
		return
	}
	c.Redirect(http.StatusFound, "/users?alert=Utilisateur+cree")
}

func (h *UsersHandler) EditPage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/users?alert=Utilisateur+introuvable")
		return
	}
	httputil.Render(c, http.StatusOK, pages.UserEditForm(layoutFromCtx(c, "Modifier utilisateur", "users"), u))
}

func (h *UsersHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	err := h.users.Update(c.Request.Context(), uint(id), services.UpdateUserInput{
		FirstName: strings.TrimSpace(c.PostForm("first_name")),
		LastName:  strings.TrimSpace(c.PostForm("last_name")),
		Email:     strings.TrimSpace(c.PostForm("email")),
		Phone:     strings.TrimSpace(c.PostForm("phone")),
		Roles:     parseRoles(c),
	}, cur.FullName())
	if err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/users/%d/edit?alert=Erreur+mise+a+jour", id))
		return
	}
	c.Redirect(http.StatusFound, "/users?alert=Utilisateur+mis+a+jour")
}

func (h *UsersHandler) PasswordPage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/users?alert=Utilisateur+introuvable")
		return
	}
	httputil.Render(c, http.StatusOK, pages.UserPasswordForm(layoutFromCtx(c, "Modifier mot de passe", "users"), u))
}

func (h *UsersHandler) UpdatePassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	password := c.PostForm("password")
	confirm := c.PostForm("password_confirm")
	if len(password) < 8 {
		c.Redirect(http.StatusFound, fmt.Sprintf("/users/%d/password?alert=Mot+de+passe+trop+court", id))
		return
	}
	if password != confirm {
		c.Redirect(http.StatusFound, fmt.Sprintf("/users/%d/password?alert=Mots+de+passe+non+identiques", id))
		return
	}
	if err := h.users.UpdatePassword(c.Request.Context(), uint(id), password, cur.FullName()); err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/users/%d/password?alert=Erreur+mise+a+jour", id))
		return
	}
	c.Redirect(http.StatusFound, "/users?alert=Mot+de+passe+mis+a+jour")
}

func parseRoles(c *gin.Context) []models.RoleKey {
	keys := c.PostFormArray("roles")
	out := make([]models.RoleKey, 0, len(keys))
	for _, k := range keys {
		out = append(out, models.RoleKey(k))
	}
	return out
}

func (h *UsersHandler) ToggleRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_ = h.users.ToggleRole(c.Request.Context(), uint(id), models.RoleKey(c.PostForm("role")))
	c.Redirect(http.StatusFound, "/users?alert=Role+mis+a+jour")
}

func (h *UsersHandler) ToggleActive(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_ = h.users.ToggleActive(c.Request.Context(), uint(id))
	c.Redirect(http.StatusFound, "/users?alert=Statut+mis+a+jour")
}

func (h *UsersHandler) ToggleCanDisburse(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	allowed := c.PostForm("allowed") == "1"
	_ = h.reqSvc.SetAccountantCanDisburse(c.Request.Context(), uint(id), allowed)
	c.Redirect(http.StatusFound, "/users?alert=Autorisation+decaissement+maj")
}

func (h *UsersHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	if cur != nil && cur.ID == uint(id) {
		c.Redirect(http.StatusFound, "/users?alert=Impossible+de+supprimer+votre+compte")
		return
	}
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	if err := h.users.Delete(c.Request.Context(), uint(id), by); err != nil {
		c.Redirect(http.StatusFound, "/users?alert=Erreur+suppression")
		return
	}
	c.Redirect(http.StatusFound, "/users?alert=Utilisateur+supprime")
}
