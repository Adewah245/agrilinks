package handlers

import (
	"backend/internal/middleware"
	"backend/internal/services"
	"log"
	"net/http"
)

type ProfileEdit struct{ auth *services.AuthService }

func NewProfileEditHandler(auth *services.AuthService) *ProfileEdit { return &ProfileEdit{auth: auth} }

func (h *ProfileEdit) Handler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r)
	if !ok {
		http.Error(w, "Please sign in", 401)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Could not read the profile form", 400)
		return
	}
	defer r.MultipartForm.RemoveAll()
	photo, err := saveUploadedFile(r, "passport", "users")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	err = h.auth.UpdateProfile(user.ID, r.FormValue("full_name"), r.FormValue("phone"), r.FormValue("email"), r.FormValue("location"), r.FormValue("state"), r.FormValue("lga"), r.FormValue("current_password"), photo)
	if err != nil {
		log.Printf("profile update: %v", err)
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
