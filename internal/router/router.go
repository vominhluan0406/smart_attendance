package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/smart-attendance/internal/config"
	"github.com/smart-attendance/smart-attendance/internal/handler"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
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
	ReportService     *service.ReportService
	DashboardService  *service.DashboardService
	PermissionService *service.PermissionService
	WebAuthnService   *service.WebAuthnService
	LeaveService      *service.LeaveService
	FraudAlertService *service.FraudAlertService
	Config              *config.Config
	RateLimitPerMin     int
	UserRateLimitPerMin int
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
	home := handler.NewHomeHandler(deps.Render, deps.BranchService, deps.WebAuthnService, deps.UserService)
	oauth := handler.NewOAuthHandler(deps.AuthService, deps.Config)
	auth := handler.NewAuthHandler(deps.AuthService, deps.Render, oauth.IsEnabled())
	users := handler.NewUserHandler(deps.UserService, deps.AuthService, deps.BranchService, deps.WebAuthnService, deps.Render)
	branches := handler.NewBranchHandler(deps.BranchService, deps.Render)
	attendance := handler.NewAttendanceHandler(deps.AttendanceService, deps.BranchService, deps.TOTPService, deps.UserService, deps.AuthService, deps.WebAuthnService, deps.Render)
	reports := handler.NewReportHandler(deps.ReportService, deps.BranchService, deps.Render)
	dashboard := handler.NewDashboardHandler(deps.DashboardService, deps.BranchService, deps.Render)
	leave := handler.NewLeaveHandler(deps.LeaveService, deps.BranchService, deps.Render)
	fraudAlerts := handler.NewFraudAlertHandler(deps.FraudAlertService, deps.BranchService, deps.Render)

	// Permission-based middleware helpers
	ps := deps.PermissionService
	requirePerm := func(code string) func(http.Handler) http.Handler {
		return middleware.RequirePermission(ps, code)
	}

	// Custom error pages
	r.NotFound(home.NotFound)
	r.MethodNotAllowed(home.NotFound)

	// Auth pages (no JWT required)
	r.Route("/auth", func(ar chi.Router) {
		ar.Get("/login", auth.LoginPage)
		ar.Post("/login", auth.LoginForm)
		// Public registration disabled - only admin can create users
		// ar.Get("/register", auth.RegisterPage)
		// ar.Post("/register", auth.RegisterForm)
		ar.Get("/logout", auth.Logout)

		// Microsoft OAuth
		ar.Get("/oauth/microsoft", oauth.MicrosoftLogin)
		ar.Get("/oauth/microsoft/callback", oauth.MicrosoftCallback)
	})

	// Protected pages (JWT required)
	r.Group(func(pr chi.Router) {
		pr.Use(middleware.JWTAuth(deps.AuthService))

		pr.Get("/", home.Index)

		// Dashboard (requires dashboard.view permission)
		pr.Route("/dashboard", func(dr chi.Router) {
			dr.Use(requirePerm(models.PermDashboardView))
			dr.Get("/", dashboard.DashboardPage)
			dr.Get("/stats", dashboard.StatsPartial)
			dr.Get("/chart", dashboard.ChartPartial)
			dr.Get("/recent", dashboard.RecentPartial)
		})

		// Reports
		pr.Route("/reports", func(rr chi.Router) {
			// Admin report selection (requires report.view_all)
			rr.Group(func(admRr chi.Router) {
				admRr.Use(requirePerm(models.PermReportViewAll))
				admRr.Get("/", reports.AdminReportPage)
			})

			// Personal history (requires report.view_own)
			rr.Group(func(ownRr chi.Router) {
				ownRr.Use(requirePerm(models.PermReportViewOwn))
				ownRr.Get("/my-history", reports.UserHistoryPage)
				ownRr.Get("/my-history/partial", reports.UserHistoryPartial)
				ownRr.Get("/my-history/export", reports.ExportUserHistory)
			})

			// Branch reports (requires report.view_branch)
			rr.Group(func(brRr chi.Router) {
				brRr.Use(requirePerm(models.PermReportViewBranch))
				brRr.Get("/branch/{branchID}", reports.BranchReportPage)
				brRr.Get("/branch/{branchID}/partial", reports.BranchReportPartial)
				brRr.Get("/branch/{branchID}/export", reports.ExportBranchReport)
			})
		})

		// Attendance — time log + QR display
		pr.Route("/attendance", func(ar chi.Router) {
			// Check-in page + log (rate limited)
			ar.Group(func(ciRouter chi.Router) {
				ciRouter.Use(requirePerm(models.PermAttendanceCheckIn))
				ciRouter.Get("/", attendance.AttendancePage)
				ciRouter.With(middleware.RateLimit(deps.RateLimitPerMin), middleware.RateLimitByUser(deps.UserRateLimitPerMin)).Post("/log", attendance.LogTimeForm)

				// Fallback Password check-in
				ciRouter.Get("/password", attendance.PasswordCheckinPage)
				ciRouter.With(middleware.RateLimit(deps.RateLimitPerMin), middleware.RateLimitByUser(deps.UserRateLimitPerMin)).Post("/password", attendance.PasswordLogForm)

				// Combined WiFi + GPS check-in
				ciRouter.Get("/wifi-gps", attendance.WiFiGPSCheckinPage)
			})

			// Manager redirect
			ar.Get("/qr-manager", attendance.ManagerQRRedirect)

			// QR display (Manager/Admin) — no rate limit, auto-refreshes every 15s
			ar.Get("/qr/{branchID}", attendance.QRDisplayPage)
			ar.Get("/qr/{branchID}/partial", attendance.QRCodePartial)
			ar.Get("/qr/{branchID}/image", attendance.QRImage)
		})

		// Leave management
		pr.Route("/leave", func(lr chi.Router) {
			lr.Get("/my", leave.MyLeavePage)
			lr.Post("/my", leave.SubmitRequest)

			lr.Group(func(mgrLr chi.Router) {
				mgrLr.Use(middleware.RequireRoles(models.RoleManager))
				mgrLr.Get("/manage", leave.ManageLeavePage)
				mgrLr.Post("/manage/{id}/review", leave.ReviewAction)
			})
		})

		// Fraud alerts (manager + admin)
		pr.Route("/alerts", func(ar chi.Router) {
			ar.Use(requirePerm(models.PermFraudAlertView))
			ar.Get("/", fraudAlerts.AlertsPage)
			ar.Get("/partial", fraudAlerts.AlertsPartial)

			ar.Group(func(rr chi.Router) {
				rr.Use(requirePerm(models.PermFraudAlertReview))
				rr.Post("/{id}/review", fraudAlerts.ReviewAction)
			})
		})

		// Branch management (requires branch.manage)
		pr.Route("/branches", func(br chi.Router) {
			br.Use(requirePerm(models.PermBranchManage))
			br.Get("/", branches.ListPage)
			br.Get("/create", branches.CreatePage)
			br.Post("/create", branches.CreateForm)
			br.Get("/{id}/edit", branches.EditPage)
			br.Put("/{id}", branches.UpdateForm)
			br.Delete("/{id}", branches.DeleteAction)
		})

		// User management (requires user.manage)
		pr.Route("/users", func(ur chi.Router) {
			ur.Use(requirePerm(models.PermUserManage))
			ur.Get("/", users.ListPage)
			ur.Get("/create", users.CreatePage)
			ur.Post("/create", users.CreateForm)
			ur.Get("/{id}/edit", users.EditPage)
			ur.Put("/{id}", users.UpdateForm)
			ur.Delete("/{id}", users.DeleteAction)

			// Credentials management
			ur.Post("/{id}/credentials/{credID}/approve", users.ApproveCredential)
			ur.Delete("/{id}/credentials/{credID}", users.DeleteCredential)
		})

		// User Profile & Biometrics
		pr.Get("/profile", users.ProfilePage)
		pr.Route("/api/webauthn", func(wr chi.Router) {
			wr.Get("/register/begin", users.RegisterBiometricBegin)
			wr.Post("/register/finish", users.RegisterBiometricFinish)
			wr.Get("/login/begin", attendance.BiometricLoginBegin)
			wr.Post("/login/finish", attendance.BiometricLoginFinish)
		})
	})

	// API routes
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", home.Health)

		// Auth API (public)
		api.Post("/auth/login", auth.APILogin)
		// api.Post("/auth/register", auth.APIRegister)
		api.Post("/auth/refresh", auth.APIRefresh)

		// Protected API
		api.Group(func(pa chi.Router) {
			pa.Use(middleware.JWTAuth(deps.AuthService))

			// Dashboard API
			pa.Route("/dashboard", func(da chi.Router) {
				da.Use(requirePerm(models.PermDashboardView))
				da.Get("/stats", dashboard.APIStats)
				da.Get("/charts", dashboard.APICharts)
			})

			// Attendance API
			pa.Route("/attendance", func(aa chi.Router) {
				aa.Use(middleware.RateLimit(deps.RateLimitPerMin))
				aa.Use(middleware.RateLimitByUser(deps.UserRateLimitPerMin))
				aa.Use(requirePerm(models.PermAttendanceCheckIn))
				aa.Post("/log", attendance.APILogTime)
				aa.Get("/status", attendance.APIStatus)
			})

			// Branch API
			pa.Route("/branches", func(ba chi.Router) {
				ba.Use(requirePerm(models.PermBranchManage))
				ba.Get("/", branches.APIList)
				ba.Post("/", branches.APICreate)
				ba.Get("/{id}", branches.APIGet)
				ba.Put("/{id}", branches.APIUpdate)
				ba.Delete("/{id}", branches.APIDelete)
			})

			// User API
			pa.Route("/users", func(ua chi.Router) {
				ua.Use(requirePerm(models.PermUserManage))
				ua.Get("/", users.APIList)
				ua.Get("/{id}", users.APIGet)
				ua.Put("/{id}", users.APIUpdate)
				ua.Delete("/{id}", users.APIDelete)
			})
		})
	})

	return r
}
