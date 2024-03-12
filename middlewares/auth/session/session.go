// SPDX-FileCopyrightText: 2015-2024 caixw
//
// SPDX-License-Identifier: MIT

// Package session SESSION 管理
package session

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/issue9/web"
)

const contextTypeKey contextType = 0

type contextType int

type context[T any] struct {
	id string // session id
	s  *Session[T]
}

// Session session 管理
//
// T 为每个 session 的数据类型，不能是指针类型。
type Session[T any] struct {
	store              Store[T]
	lifetime           int
	name, path, domain string
	secure, httpOnly   bool
}

func (c *context[T]) set(v *T) error { return c.s.store.Set(c.id, v) }

func (c *context[T]) get() (*T, error) { return c.s.store.Get(c.id) }

func (c *context[T]) del() error { return c.s.store.Delete(c.id) }

// New 声明 Session 中间件
//
// lifetime 为 session 的有效时间，单位为秒；其它参数为 cookie 的相关设置。
func New[T any](store Store[T], lifetime int, name, path, domain string, secure, httpOnly bool) *Session[T] {
	return &Session[T]{
		store:    store,
		lifetime: lifetime,
		name:     name,
		path:     path,
		domain:   domain,
		secure:   secure,
		httpOnly: httpOnly,
	}
}

// Logout 退出登录
func (s *Session[T]) Logout(sessionid string) error { return s.store.Delete(sessionid) }

func (s *Session[T]) Middleware(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) web.Responser {
		var id string

		c, err := ctx.Request().Cookie(s.name)
		if err != nil && !errors.Is(err, http.ErrNoCookie) { // 不退出，给定默认值。
			ctx.Logs().ERROR().Error(err)
		}

		if c == nil {
			c = &http.Cookie{
				Name:     s.name,
				Path:     s.path,
				Domain:   s.domain,
				Secure:   s.secure,
				HttpOnly: s.httpOnly,
			}
		}

		if c.Value == "" {
			id = ctx.Server().UniqueID()
			c.Value = url.QueryEscape(id)
		} else {
			if id, err = url.QueryUnescape(c.Value); err != nil {
				return ctx.Error(err, web.ProblemInternalServerError)
			}
		}

		c.MaxAge = s.lifetime
		c.Expires = ctx.Begin().Add(time.Second * time.Duration(s.lifetime)) // http 1.0 和 ie8 仅支持此属性
		ctx.SetCookies(c)

		if v, err := s.store.Get(id); err != nil {
			return ctx.Error(err, web.ProblemInternalServerError)
		} else if v == nil {
			var z T
			s.store.Set(id, &z)
			return ctx.Problem(web.ProblemUnauthorized)
		}

		ctx.SetVar(contextTypeKey, &context[T]{id: id, s: s})

		return next(ctx)
	}
}

// GetValue 获取当前对话关联的信息
func GetValue[T any](ctx *web.Context) (sessionid string, val *T, err error) {
	if c, found := ctx.GetVar(contextTypeKey); found {
		cc := c.(*context[T])
		val, err := cc.get()
		return cc.id, val, err
	}

	var v T
	return "", &v, web.NewLocaleError("not found the context session key")
}

// SetValue 更新 session 保存的值
func SetValue[T any](ctx *web.Context, val *T) error {
	if c, found := ctx.GetVar(contextTypeKey); found {
		return c.(*context[T]).set(val)
	}
	return web.NewLocaleError("not found the context session key")
}

// DelValue 删除 session 中保存的值
func DelValue[T any](ctx *web.Context) error {
	if c, found := ctx.GetVar(contextTypeKey); found {
		return c.(*context[T]).del()
	}
	return nil
}
