// Package middleware implements the middleware chain for request processing.
package middleware

import (
	"net/http"
)

// chain implements the Chain interface for middleware composition.
type chain struct {
	middlewares []Middleware
}

// NewChain creates a new middleware chain.
func NewChain(middlewares ...Middleware) Chain {
	return &chain{
		middlewares: append([]Middleware{}, middlewares...),
	}
}

// Use adds middleware to the chain.
func (c *chain) Use(middleware Middleware) Chain {
	return &chain{
		middlewares: append(c.middlewares, middleware),
	}
}

// Then applies the middleware chain to a handler.
func (c *chain) Then(handler http.Handler) http.Handler {
	if handler == nil {
		handler = http.DefaultServeMux
	}

	// Apply middleware in reverse order (last added, first executed)
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i].Wrap(handler)
	}

	return handler
}

// ThenFunc applies the middleware chain to a handler function.
func (c *chain) ThenFunc(handlerFunc http.HandlerFunc) http.Handler {
	return c.Then(handlerFunc)
}

// Append extends the chain with additional middleware.
func (c *chain) Append(middlewares ...Middleware) Chain {
	newMiddlewares := make([]Middleware, len(c.middlewares)+len(middlewares))
	copy(newMiddlewares, c.middlewares)
	copy(newMiddlewares[len(c.middlewares):], middlewares)

	return &chain{
		middlewares: newMiddlewares,
	}
}

// Extend creates a new chain by extending this one with additional middleware.
func (c *chain) Extend(middlewares ...Middleware) Chain {
	return c.Append(middlewares...)
}
