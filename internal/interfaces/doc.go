// Package interfaces defines all service interfaces for the ProxyND application.
// These interfaces enable dependency injection, improve testability, and provide
// clear contracts between different layers of the application.
//
// The interfaces are organized by domain:
//   - cache.go: Cache management and backend interfaces
//   - webhook.go: Webhook batch processing and sending interfaces
//   - auth.go: Authentication and JWT service interfaces
//   - health.go: Health monitoring and checking interfaces
//   - verification.go: Package verification interfaces
//
// Usage example:
//
//	    cache interfaces.CacheManager
//	    auth  interfaces.AuthService
//	}
//
//	    return &MyService{
//	        cache: cache,
//	        auth:  auth,
//	    }
//	}
package interfaces
