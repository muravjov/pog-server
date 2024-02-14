package grpcproxy

import (
	"testing"
	"time"

	"git.catbo.net/muravjov/go2023/util"
	"github.com/stretchr/testify/require"
)

func TestPOG_AUTH_var(t *testing.T) {
	t.SkipNow()

	name := "root"
	pass := "password"
	// a half of a year
	timeToLive := time.Hour * 24 * 30 * 6

	hash := GenAuthItem(name, pass, timeToLive)
	ok := doPasswordsMatch(hash, pass)
	require.True(t, ok)
}

func TestPOG_AUTHParsing(t *testing.T) {
	t.SkipNow()

	lst, err := ParseAuthList(POGAuthEnvVarPrefix)
	require.NoError(t, err)
	util.DumpIndent(lst)
}
