// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import "errors"

var (
	ErrInvalidArgs             = errors.New("invalid args")
	ErrMissingSubcommand       = errors.New("must specify a subcommand")
	ErrInvalidAddress          = errors.New("invalid address")
	ErrInvalidKeyType          = errors.New("invalid key type")
	ErrInvalidVerificationType = errors.New("invalid verify type")
)
