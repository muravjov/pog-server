package grpcproxy

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"git.catbo.net/muravjov/go2023/util"
	"github.com/stretchr/testify/require"
)

func TestPOG_AUTH_var(t *testing.T) {
	name := "root"
	pass := "password"

	hash, _ := hashPassword(pass)
	ok := doPasswordsMatch(hash, pass)
	require.True(t, ok)

	// a half of a year
	timeToLive := time.Hour * 24 * 30 * 6
	expirationDate := time.Now().UTC().Add(timeToLive)

	b, _ := json.Marshal(AuthItem{
		Name:       name,
		Hash:       hash,
		ExpDateStr: expirationDate.Format(time.RFC3339),
	})
	fmt.Println(string(b))
}

func TestPOG_AUTHParsing(t *testing.T) {
	lst := ParseAuthList()
	util.DumpIndent(lst)
}
