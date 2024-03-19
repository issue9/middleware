// SPDX-FileCopyrightText: 2015-2024 caixw
//
// SPDX-License-Identifier: MIT

package jwt

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/issue9/web"
)

const (
	contextKey contextKeyType = 1

	prefix = "bearer "

	prefixLen = 7 // len(prefix)
)

type (
	// Verifier JWT 验证器
	//
	// 仅负责对令牌的验证，如果需要签发令牌，则需要 [Signer] 对象，
	// 同时需要保证 [Signer] 添加的证书数量和 ID 与当前对象是相同的。
	Verifier[T Claims] struct {
		blocker       Blocker[T]
		keyFunc       jwt.Keyfunc
		claimsBuilder BuildClaimsFunc[T]
		keys          []*key
	}

	BuildClaimsFunc[T Claims] func() T

	contextKeyType int
)

// NewVerifier 声明 [Verifier] 对象
//
// b 为处理丢弃令牌的对象，如果为空表示不会对任何令牌作特殊处理；
// f 为 [Claims] 对象的生成方法；
func NewVerifier[T Claims](b Blocker[T], f BuildClaimsFunc[T]) *Verifier[T] {
	j := &Verifier[T]{
		blocker:       b,
		claimsBuilder: f,
		keys:          make([]*key, 0, 10),
	}

	j.keyFunc = func(t *jwt.Token) (any, error) {
		if len(j.keys) == 0 || len(t.Header) == 0 {
			return nil, ErrSigningMethodNotFound()
		}

		if kid, found := t.Header["kid"]; found {
			if index := slices.IndexFunc(j.keys, func(e *key) bool { return e.id == kid }); index >= 0 {
				k := j.keys[index]
				t.Method = k.sign // 忽略由用户指定的 header['alg']，而是有 kid 指定。
				return k.key, nil
			}
		}

		return nil, ErrSigningMethodNotFound()
	}

	return j
}

func (j *Verifier[T]) Logout(ctx *web.Context) error {
	if c, found := j.GetValue(ctx); found {
		return j.blocker.BlockToken(GetToken(ctx), c.IsRefresh())
	}
	return nil
}

// VerifiyRefresh 验证刷新令牌
func (j *Verifier[T]) VerifiyRefresh(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) web.Responser { return j.resp(ctx, true, next) }
}

// Middleware 解码用户的 token 并写入 [web.Context]
func (j *Verifier[T]) Middleware(next web.HandlerFunc) web.HandlerFunc {
	return func(ctx *web.Context) web.Responser { return j.resp(ctx, false, next) }
}

func (j *Verifier[T]) resp(ctx *web.Context, refresh bool, next web.HandlerFunc) web.Responser {
	token := GetToken(ctx)
	if token == "" || j.blocker.TokenIsBlocked(token) {
		return ctx.Problem(web.ProblemUnauthorized)
	}

	t, err := jwt.ParseWithClaims(token, j.claimsBuilder(), j.keyFunc)
	if err != nil { // 都算验证错误
		ctx.Logs().ERROR().Error(err)
		return ctx.Problem(web.ProblemUnauthorized)
	}

	if !t.Valid {
		return ctx.Problem(web.ProblemUnauthorized)
	}

	claims := t.Claims.(T)

	if refresh != claims.IsRefresh() {
		return ctx.Problem(web.ProblemUnauthorized)
	}

	if j.blocker.ClaimsIsBlocked(claims) {
		return ctx.Problem(web.ProblemUnauthorized)
	}

	if refresh { // 刷新令牌是一次性的
		if err := j.blocker.BlockToken(token, true); err != nil {
			ctx.Logs().ERROR().Error(err)
		}
	}

	ctx.SetVar(contextKey, claims)

	return next(ctx)
}

// GetValue 返回解码后的 [Claims] 对象
func (j *Verifier[T]) GetValue(ctx *web.Context) (claims T, found bool) {
	if v, found := ctx.GetVar(contextKey); found {
		return v.(T), true
	}
	var vv T
	return vv, false
}

// GetToken 获取客户端提交的 token
func GetToken(ctx *web.Context) string {
	h := ctx.Request().Header.Get("Authorization")
	if len(h) > prefixLen && strings.ToLower(h[:prefixLen]) == prefix {
		h = h[prefixLen:]
	}
	return h
}

func (j *Verifier[T]) addKey(id string, sign SigningMethod, keyData any) {
	if slices.IndexFunc(j.keys, func(e *key) bool { return e.id == id }) >= 0 {
		panic(fmt.Sprintf("存在同名的签名方法 %s", id))
	}

	j.keys = append(j.keys, &key{
		id:   id,
		sign: sign,
		key:  keyData,
	})
}

func (j *Verifier[T]) AddHMAC(id string, sign *jwt.SigningMethodHMAC, secret []byte) {
	j.addKey(id, sign, secret)
}

func (j *Verifier[T]) addRSA(id string, sign SigningMethod, public []byte) {
	pub, err := jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		panic(err)
	}
	j.addKey(id, sign, pub)
}

func (j *Verifier[T]) AddRSA(id string, sign *jwt.SigningMethodRSA, public []byte) {
	j.addRSA(id, sign, public)
}

func (j *Verifier[T]) AddRSAPSS(id string, sign *jwt.SigningMethodRSAPSS, public []byte) {
	j.addRSA(id, sign, public)
}

func (j *Verifier[T]) AddECDSA(id string, sign *jwt.SigningMethodECDSA, public []byte) {
	pub, err := jwt.ParseECPublicKeyFromPEM(public)
	if err != nil {
		panic(err)
	}
	j.addKey(id, sign, pub)
}

func (j *Verifier[T]) AddEd25519(id string, sign *jwt.SigningMethodEd25519, public []byte) {
	pub, err := jwt.ParseEdPublicKeyFromPEM(public)
	if err != nil {
		panic(err)
	}

	j.addKey(id, sign, pub)
}

func (j *Verifier[T]) AddRSAFromFS(id string, sign *jwt.SigningMethodRSA, fsys fs.FS, public string) {
	pub, err := fs.ReadFile(fsys, public)
	if err != nil {
		panic(err)
	}
	j.AddRSA(id, sign, pub)
}

func (j *Verifier[T]) AddRSAPSSFromFS(id string, sign *jwt.SigningMethodRSAPSS, fsys fs.FS, public string) {
	pub, err := fs.ReadFile(fsys, public)
	if err != nil {
		panic(err)
	}
	j.AddRSAPSS(id, sign, pub)
}

func (j *Verifier[T]) AddECDSAFromFS(id string, sign *jwt.SigningMethodECDSA, fsys fs.FS, public string) {
	pub, err := fs.ReadFile(fsys, public)
	if err != nil {
		panic(err)
	}
	j.AddECDSA(id, sign, pub)
}

func (j *Verifier[T]) AddEd25519FromFS(id string, sign *jwt.SigningMethodEd25519, fsys fs.FS, public string) {
	pub, err := fs.ReadFile(fsys, public)
	if err != nil {
		panic(err)
	}
	j.AddEd25519(id, sign, pub)
}

// Add 添加签名方法
func (j *Verifier[T]) Add(id string, sign SigningMethod, public []byte) {
	switch m := sign.(type) {
	case *jwt.SigningMethodHMAC:
		j.AddHMAC(id, m, public)
	case *jwt.SigningMethodRSA:
		j.AddRSA(id, m, public)
	case *jwt.SigningMethodRSAPSS:
		j.AddRSAPSS(id, m, public)
	case *jwt.SigningMethodECDSA:
		j.AddECDSA(id, m, public)
	case *jwt.SigningMethodEd25519:
		j.AddEd25519(id, m, public)
	default:
		panic(invalidSignForID(id))
	}
}

// AddFromFS 添加签名方法密钥从文件中加载
func (j *Verifier[T]) AddFromFS(id string, sign SigningMethod, fsys fs.FS, public string) {
	switch m := sign.(type) {
	case *jwt.SigningMethodRSA:
		j.AddRSAFromFS(id, m, fsys, public)
	case *jwt.SigningMethodRSAPSS:
		j.AddRSAPSSFromFS(id, m, fsys, public)
	case *jwt.SigningMethodECDSA:
		j.AddECDSAFromFS(id, m, fsys, public)
	case *jwt.SigningMethodEd25519:
		j.AddEd25519FromFS(id, m, fsys, public)
	default:
		panic(invalidSignForID(id))
	}
}

func invalidSignForID(id string) string { return fmt.Sprintf("%s 对应的签名方法无效", id) }
