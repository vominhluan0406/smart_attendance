package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/smart-attendance/internal/handler"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type Deps struct {
	Render            *renderer.Renderer
	AuthService       *service.AuthService
	UserService       *service.UserService
	BranchService     *service.BranchService
	AttendanceService *service.AttendanceService
	TOTPService       *service.TOTPService
	RateLimitPerMin   int
}

func New(deps Deps) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)
	r.Use(chimw.RealIP)
	r.Use(chimw.Compress(5))

	// Static files
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Handlers
	home := handler.NewHomeHandler(deps.Render)
	auth := handler.NewAuthHandler(deps.AuthService, deps.Render)
	users := handler.NewUserHandler(deps.UserService, deps.AuthService, deps.Render)
	branches := handler.NewBranchHandler(deps.BranchService, deps.Render)
	attendance := handler.NewAttendanceHandler(deps.AttendanceService, deps.BranchService, deps.TOTPService, deps.Render)

	// Public pages
	r.Get("/", home.Index)

	// Auth pages (no JWT required)
	r.Route("/auth", func(ar chi.Router) {
		ar.Get("/login", auth.LoginPage)
		ar.Post("/login", auth.LoginForm)
		ar.Get("/register", auth.RegisterPage)
		ar.Post("/register", auth.RegisterForm)
		ar.Get("/logout", auth.Logout)
	})

	// Protected pages (JWT required)
	r.Group(func(pr chi.Router) {
		pr.Use(middleware.JWTAuth(deps.AuthService))

		pr.Get("/dashboard", home.Index)
		pr.Get("/reports", home.Index) // Placeholder until reports is built

		// Attendance check-in/out
		pr.Route("/attendance", func(ar chi.Router) {
			ar.Use(middleware.RateLimit(deps.RateLimitPerMin))
			ar.Get("/", attendance.CheckInPage)
			ar.Post("/check-in", attendance.CheckInForm)
			ar.Post("/check-out", attendance.CheckOutForm)

			// QR display (Manager/Admin) — shows live QR for branch
			ar.Get("/qr/{branchID}", attendance.QRDisplayPage)
			ar.Get("/qr/{branchID}/partial", attendance.QRCodePartial)
			ar.Get("/qr/{branchID}/image", attendance.QRImage)
		})

		// Branch management (Admin only)
		pr.Route("/branches", func(br chi.Router) {
			br.Use(middleware.AdminOnly)
			br.Get("/", branches.ListPage)
			br.Get("/create", branches.CreatePage)
			br.Post("/create", branches.CreateForm)
			br.Get("/{id}/edit", branches.EditPage)
			br.Put("/{id}", branches.UpdateForm)
			br.Delete("/{id}", branches.DeleteAction)
		})

		// User management (Admin only)
		pr.Route("/users", func(ur chi.Router) {
			ur.Use(middleware.AdminOnly)
			ur.Get("/", users.ListPage)
			ur.Get("/create", users.CreatePage)
			ur.Post("/create", users.CreateForm)
			ur.Get("/{id}/edit", users.EditPage)
			ur.Put("/{id}", users.UpdateForm)
			ur.Delete("/{id}", users.DeleteAction)
		})
	})

	// API routes
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", home.Health)

		// Auth API (public)
		api.Post("/auth/login", auth.APILogin)
		api.Post("/auth/register", auth.APIRegister)
		api.Post("/auth/refresh", auth.APIRefresh)

		// Protected API
		api.Group(func(pa chi.Router) {
			pa.Use(middleware.JWTAuth(deps.AuthService))

			// Attendance API
			pa.Route("/attendance", func(aa chi.Router) {
				aa.Use(middleware.RateLimit(deps.RateLimitPerMin))
				aa.Post("/check-in", attendance.APICheckIn)
				aa.Post("/check-out", attendance.APICheckOut)
				aa.Get("/status", attendance.APIStatus)
			})

			// Branch API (Admin only)
			pa.Route("/branches", func(ba chi.Router) {
				ba.Use(middleware.AdminOnly)
				ba.Get("/", branches.APIList)
				ba.Post("/", branches.APICreate)
				ba.Get("/{id}", branches.APIGet)
				ba.Put("/{id}", branches.APIUpdate)
				ba.Delete("/{id}", branches.APIDelete)
			})

			// User API (Admin only)
			pa.Route("/users", func(ua chi.Router) {
				ua.Use(middleware.AdminOnly)
				ua.Get("/", users.APIList)
				ua.Get("/{id}", users.APIGet)
				ua.Put("/{id}", users.APIUpdate)
				ua.Delete("/{id}", users.APIDelete)
			})
		})
	})

	return r
}
