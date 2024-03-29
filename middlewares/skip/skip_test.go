// SPDX-FileCopyrightText: 2024 caixw
//
// SPDX-License-Identifier: MIT

package skip

import (
	"net/http"
	"testing"

	"github.com/issue9/assert/v4"
	"github.com/issue9/web"
	"github.com/issue9/web/server/servertest"

	"github.com/issue9/webuse/v7/internal/testserver"
)

func TestSkip(t *testing.T) {
	a := assert.New(t, false)
	s := testserver.New(a)

	next := func(ctx *web.Context) web.Responser {
		return web.Created(nil, "")
	}

	router := s.Routers().New("def", nil)
	router.Any("/test", New(func(ctx *web.Context) bool { return ctx.Request().Method != http.MethodHead }, web.ProblemBadRequest).Middleware(next))

	defer servertest.Run(a, s)()
	defer s.Close(0)

	servertest.Get(a, "http://localhost:8080/test").
		Header("X-Forwarded-For", "192.168.1.1").
		Do(nil).
		Status(http.StatusCreated)

	servertest.NewRequest(a, http.MethodHead, "http://localhost:8080/test").
		Header("X-Forwarded-For", "192.168.1.1").
		Do(nil).
		Status(http.StatusBadRequest)
}
