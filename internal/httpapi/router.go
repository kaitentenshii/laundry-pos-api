package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type RouterOptions struct {
	Auth           AuthService
	AllowedOrigins []string
	Accounts       AccountService
	Customers      CustomerService
	Services       LaundryServiceManager
	Orders         OrderManager
}

func NewRouter(logger *slog.Logger, readinessChecker ReadinessChecker, options ...RouterOptions) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(recoverer(logger))
	if len(options) > 0 {
		router.Use(cors(options[0].AllowedOrigins))
		if options[0].Auth != nil {
			registerAuthRoutes(router, options[0].Auth, logger)
			if options[0].Orders != nil {
				registerOrderRoutes(router, options[0].Auth, options[0].Orders, logger)
			}
			if options[0].Services != nil {
				registerLaundryServiceRoutes(router, options[0].Auth, options[0].Services, logger)
			}
			if options[0].Customers != nil {
				registerCustomerRoutes(router, options[0].Auth, options[0].Customers, logger)
			}
			if options[0].Accounts != nil {
				registerAccountRoutes(router, options[0].Auth, options[0].Accounts, logger)
			}
		}
	}

	router.Get("/health", health)
	router.Get("/ready", readiness(readinessChecker))

	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "The requested resource was not found.")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "The request method is not allowed for this resource.")
	})

	return router
}

func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "error", recovered, "request_id", middleware.GetReqID(r.Context()))
					writeError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
