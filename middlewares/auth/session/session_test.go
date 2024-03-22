// SPDX-FileCopyrightText: 2015-2024 caixw
//
// SPDX-License-Identifier: MIT

package session

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/issue9/assert/v4"
	"github.com/issue9/web"
	"github.com/issue9/web/server"
	"github.com/issue9/web/server/servertest"

	"github.com/issue9/webuse/v7/middlewares/auth"
)

var _ auth.Auth = &Session[int]{}

type data struct {
	Count int `query:"count"`
}

func TestSession(t *testing.T) {
	a := assert.New(t, false)
	srv, err := server.New("test", "1.0.0", &server.Options{
		HTTPServer: &http.Server{Addr: ":8080"},
		Mimetypes:  server.JSONMimetypes(),
		Logs:       &server.Logs{Handler: server.NewTermHandler(os.Stdout, nil), Created: server.NanoLayout},
	})
	a.NotError(err).NotNil(srv)

	store := NewCacheStore[*data](srv.Cache(), 500*time.Microsecond)
	a.NotNil(store)

	session := New(srv, store, 60, "sesson_id", "/", "localhost", false, false)
	a.NotNil(session)

	srv.Routers().Use(session)
	r := srv.Routers().New("default", nil)

	r.Get("/get1", func(ctx *web.Context) web.Responser {
		//a.TB().Helper()

		want := &data{}
		if resp := ctx.QueryObject(true, want, web.ProblemInternalServerError); resp != nil {
			return resp
		}

		v, err := GetValue[*data](ctx)
		a.NotError(err).Equal(v, want)

		v.Count++
		a.NotError(session.Save(ctx, v))

		return web.OK(nil)
	})

	r.Delete("/get1", func(ctx *web.Context) web.Responser {
		if err := session.Logout(ctx); err != nil {
			return ctx.Error(err, web.ProblemInternalServerError)
		}
		return web.NoContent()
	})

	defer servertest.Run(a, srv)()
	defer srv.Close(0)

	// 第一次验证，初始化 cookie
	resp := servertest.Get(a, "http://localhost:8080/get1").
		Do(nil).
		Status(http.StatusOK).
		Resp()

	// 第二次，带上 cookie
	cookie := resp.Cookies()[0]
	resp = servertest.Get(a, "http://localhost:8080/get1?count=1&id=").
		Cookie(cookie).
		Do(nil).
		Status(http.StatusOK).
		Resp()

	// 第三次访问
	cookie = resp.Cookies()[0]
	resp = servertest.Get(a, "http://localhost:8080/get1?count=2&id=").
		Cookie(cookie).
		Do(nil).
		Status(http.StatusOK).
		Resp()

	// 删除 cookie
	cookie = resp.Cookies()[0]
	resp = servertest.Delete(a, "http://localhost:8080/get1").
		Cookie(cookie).
		Do(nil).
		Status(http.StatusNoContent).
		Resp()

	// cookie 已经被删除
	servertest.Get(a, "http://localhost:8080/get1?count=0&id=").
		Cookie(cookie).
		Do(nil).
		Status(http.StatusOK)
}
