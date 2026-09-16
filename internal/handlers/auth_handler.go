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
	httputil.Render(c, http.StatusOK, pages.Login(httputil.PopFlash(c), csrf))
}

func (h *AuthHandler) LoginPost(c *gin.Context) {
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	_, token, err := h.auth.Login(c.Request.Context(), email, password)
	if err != nil {
		httputil.RedirectFlash(c, "/login", "Identifiants invalides")
		return
	}
	jcfg, err := config.LoadJWTConfig()
	if err != nil {
		httputil.RedirectFlash(c, "/login", "Configuration indisponible")
		return
	}
	middlewares.SetAuthCookie(c, jcfg.CookieName, token, int(jcfg.TTL.Seconds()))
	c.Redirect(http.StatusFound, "/dashboard")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	jcfg, _ := config.LoadJWTConfig()
	middlewares.SetAuthCookie(c, jcfg.CookieName, "", -1)
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
		httputil.RedirectFlash(c, "/profile/password", "Mot de passe actuel incorrect")
		return
	}
	if err := services.ValidatePassword(password); err != nil {
		httputil.RedirectFlash(c, "/profile/password", "Mot de passe trop court (8 caractères minimum)")
		return
	}
	if password != confirm {
		httputil.RedirectFlash(c, "/profile/password", "Mots de passe non identiques")
		return
	}
	if err := h.users.UpdatePassword(c.Request.Context(), user.ID, password, user.FullName()); err != nil {
		httputil.RedirectFlash(c, "/profile/password", "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/profile/password", "Mot de passe mis à jour")
}

type DashboardHandler struct {
	dash *services.DashboardService
}

func NewDashboardHandler(dash *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dash: dash}
}

func (h *DashboardHandler) Index(c *gin.Context) {
	user := middlewares.CurrentUser(c)
	m, _ := h.dash.Metrics(c.Request.Context(), user)
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
	roles, err := parseRoles(c)
	if err != nil {
		httputil.RedirectFlash(c, "/users/new", "Rôle invalide")
		return
	}
	_, err = h.users.Create(c.Request.Context(), services.CreateUserInput{
		FirstName: strings.TrimSpace(c.PostForm("first_name")),
		LastName:  strings.TrimSpace(c.PostForm("last_name")),
		Email:     strings.TrimSpace(c.PostForm("email")),
		Phone:     strings.TrimSpace(c.PostForm("phone")),
		Password:  c.PostForm("password"),
		Roles:     roles,
	}, cur.FullName())
	if err != nil {
		httputil.RedirectFlash(c, "/users/new", "Erreur lors de la création")
		return
	}
	httputil.RedirectFlash(c, "/users", "Utilisateur créé")
}

func (h *UsersHandler) EditPage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		httputil.RedirectFlash(c, "/users", "Utilisateur introuvable")
		return
	}
	httputil.Render(c, http.StatusOK, pages.UserEditForm(layoutFromCtx(c, "Modifier utilisateur", "users"), u))
}

func (h *UsersHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	roles, err := parseRoles(c)
	if err != nil {
		httputil.RedirectFlash(c, fmt.Sprintf("/users/%d/edit", id), "Rôle invalide")
		return
	}
	err = h.users.Update(c.Request.Context(), uint(id), services.UpdateUserInput{
		FirstName: strings.TrimSpace(c.PostForm("first_name")),
		LastName:  strings.TrimSpace(c.PostForm("last_name")),
		Email:     strings.TrimSpace(c.PostForm("email")),
		Phone:     strings.TrimSpace(c.PostForm("phone")),
		Roles:     roles,
	}, cur.FullName())
	if err != nil {
		httputil.RedirectFlash(c, fmt.Sprintf("/users/%d/edit", id), "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/users", "Utilisateur mis à jour")
}

func (h *UsersHandler) PasswordPage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	u, err := h.users.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		httputil.RedirectFlash(c, "/users", "Utilisateur introuvable")
		return
	}
	httputil.Render(c, http.StatusOK, pages.UserPasswordForm(layoutFromCtx(c, "Modifier mot de passe", "users"), u))
}

func (h *UsersHandler) UpdatePassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	password := c.PostForm("password")
	confirm := c.PostForm("password_confirm")
	if err := services.ValidatePassword(password); err != nil {
		httputil.RedirectFlash(c, fmt.Sprintf("/users/%d/password", id), "Mot de passe trop court (8 caractères minimum)")
		return
	}
	if password != confirm {
		httputil.RedirectFlash(c, fmt.Sprintf("/users/%d/password", id), "Mots de passe non identiques")
		return
	}
	if err := h.users.UpdatePassword(c.Request.Context(), uint(id), password, cur.FullName()); err != nil {
		httputil.RedirectFlash(c, fmt.Sprintf("/users/%d/password", id), "Erreur lors de la mise à jour")
		return
	}
	httputil.RedirectFlash(c, "/users", "Mot de passe mis à jour")
}

func parseRoles(c *gin.Context) ([]models.RoleKey, error) {
	keys := c.PostFormArray("roles")
	out := make([]models.RoleKey, 0, len(keys))
	for _, k := range keys {
		rk := models.RoleKey(k)
		if !rk.Valid() {
			return nil, fmt.Errorf("rôle invalide: %s", k)
		}
		out = append(out, rk)
	}
	return out, nil
}

func (h *UsersHandler) ToggleRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	role := models.RoleKey(c.PostForm("role"))
	if !role.Valid() {
		httputil.RedirectFlash(c, "/users", "Rôle invalide")
		return
	}
	_ = h.users.ToggleRole(c.Request.Context(), uint(id), role)
	httputil.RedirectFlash(c, "/users", "Rôle mis à jour")
}

func (h *UsersHandler) ToggleActive(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	_ = h.users.ToggleActive(c.Request.Context(), uint(id))
	httputil.RedirectFlash(c, "/users", "Statut mis à jour")
}

func (h *UsersHandler) ToggleCanDisburse(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	allowed := c.PostForm("allowed") == "1"
	_ = h.reqSvc.SetAccountantCanDisburse(c.Request.Context(), uint(id), allowed)
	httputil.RedirectFlash(c, "/users", "Autorisation décaissement mise à jour")
}

func (h *UsersHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	cur := middlewares.CurrentUser(c)
	if cur != nil && cur.ID == uint(id) {
		httputil.RedirectFlash(c, "/users", "Impossible de supprimer votre propre compte")
		return
	}
	by := ""
	if cur != nil {
		by = cur.FullName()
	}
	if err := h.users.Delete(c.Request.Context(), uint(id), by); err != nil {
		httputil.RedirectFlash(c, "/users", "Erreur lors de la suppression")
		return
	}
	httputil.RedirectFlash(c, "/users", "Utilisateur supprimé")
}
