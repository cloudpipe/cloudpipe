package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("I exist")
}

// ContextHandler is an HTTP HandlerFunc that accepts an additional parameter containing the
// server context.
type ContextHandler func(c *Context, w http.ResponseWriter, r *http.Request)

// HandleWith returns an http.HandlerFunc that binds a ContextHandler to a specific Context.
func (ch ContextHandler) HandleWith(c *Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { ch(c, w, r) }
}
