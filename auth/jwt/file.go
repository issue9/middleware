// SPDX-License-Identifier: MIT

package jwt

import (
	"io/fs"

	"github.com/golang-jwt/jwt/v4"
)

func NewRSAFromFS[T jwt.Claims](b ClaimsBuilderFunc[T], sign *jwt.SigningMethodRSA, fsys fs.FS, private, public string) (*JWT[T], error) {
	pvt, pub, err := loadFS(fsys, private, public)
	if err != nil {
		return nil, err
	}
	return NewRSA(b, sign, pvt, pub)
}

func NewRSAPSSFromFS[T jwt.Claims](b ClaimsBuilderFunc[T], sign *jwt.SigningMethodRSAPSS, fsys fs.FS, private, public string) (*JWT[T], error) {
	pvt, pub, err := loadFS(fsys, private, public)
	if err != nil {
		return nil, err
	}
	return NewRSAPSS(b, sign, pvt, pub)
}

func NewECDSAFromFS[T jwt.Claims](b ClaimsBuilderFunc[T], sign *jwt.SigningMethodECDSA, fsys fs.FS, private, public string) (*JWT[T], error) {
	pvt, pub, err := loadFS(fsys, private, public)
	if err != nil {
		return nil, err
	}
	return NewECDSA(b, sign, pvt, pub)
}

func NewEd25519FromFS[T jwt.Claims](b ClaimsBuilderFunc[T], sign *jwt.SigningMethodEd25519, fsys fs.FS, private, public string) (*JWT[T], error) {
	pvt, pub, err := loadFS(fsys, private, public)
	if err != nil {
		return nil, err
	}
	return NewEd25519(b, sign, pvt, pub)
}

func loadFS(fsys fs.FS, private, public string) (pvt []byte, pub []byte, err error) {
	pvt, err = fs.ReadFile(fsys, private)
	if err != nil {
		return nil, nil, err
	}

	pub, err = fs.ReadFile(fsys, public)
	if err != nil {
		return nil, nil, err
	}

	return pvt, pub, nil
}
