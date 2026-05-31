package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Hhz0823/oiwest-core/O-ui/database"
	"github.com/Hhz0823/oiwest-core/O-ui/model"
	"github.com/Hhz0823/oiwest-core/O-ui/web/middleware"

	"golang.org/x/crypto/bcrypt"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func success(w http.ResponseWriter, data interface{}) {
	writeJSON(w, 200, model.APIResponse{Success: true, Data: data})
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, model.APIResponse{Success: false, Message: msg})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	var user model.User
	var hashedPwd string
	err := db.QueryRow("SELECT id, username, password, role FROM users WHERE username=?", req.Username).
		Scan(&user.ID, &user.Username, &hashedPwd, &user.Role)
	if err != nil {
		fail(w, 401, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.Password)); err != nil {
		fail(w, 401, "invalid credentials")
		return
	}
	db.Exec("UPDATE users SET last_login=? WHERE id=?", time.Now(), user.ID)
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		fail(w, 500, "token generation failed")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "o-ui-token", Value: token, Path: "/", HttpOnly: true, MaxAge: 86400,
	})
	success(w, model.LoginResponse{Token: token, Username: user.Username, Role: user.Role})
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		fail(w, 401, "unauthorized")
		return
	}
	success(w, map[string]interface{}{
		"user_id": claims.UserID, "username": claims.Username, "role": claims.Role,
	})
}

func ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		fail(w, 401, "unauthorized")
		return
	}
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, 400, "invalid request")
		return
	}
	db := database.GetDB()
	var hashedPwd string
	db.QueryRow("SELECT password FROM users WHERE id=?", claims.UserID).Scan(&hashedPwd)
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(req.OldPassword)); err != nil {
		fail(w, 401, "old password incorrect")
		return
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	db.Exec("UPDATE users SET password=? WHERE id=?", string(newHash), claims.UserID)
	success(w, nil)
}
