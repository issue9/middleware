// SPDX-FileCopyrightText: 2015-2024 caixw
//
// SPDX-License-Identifier: MIT

// Package locales 本地化数据
package locales

import "embed"

//go:embed *.yaml
var Locales embed.FS
