package commands

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// A malformed number names the command and the argument, and still carries
// strconv's error, so errors.Is(err, strconv.ErrSyntax) holds from Dispatch
// as it did on v0.3.2.
func TestAMalformedNumberArgumentWrapsTheStrconvError(t *testing.T) {
	cases := []struct {
		command string
		args    []string
		message string
	}{
		{"shift", []string{"x", "abc"}, `shift: invalid periods "abc"`},
		{"diffn", []string{"x", "abc"}, `diffn: invalid periods "abc"`},
		{"pctchange", []string{"x", "abc"}, `pctchange: invalid periods "abc"`},
		{"rolling", []string{"x", "abc", "mean"}, `rolling: invalid window "abc"`},
		{"rolling", []string{"x", "2", "mean", "minobs", "abc"}, `rolling: invalid minobs "abc"`},
		{"expanding", []string{"x", "abc", "sum"}, `expanding: invalid minobs "abc"`},
		{"ewm", []string{"x", "alpha", "abc", "mean"}, `ewm: invalid alpha "abc"`},
		{"ewm", []string{"x", "alpha", "0.5", "mean", "minobs", "abc"}, `ewm: invalid minobs "abc"`},
		{"movavg", []string{"x", "abc"}, `movavg: invalid window "abc"`},
		{"expsmooth", []string{"x", "abc"}, `expsmooth: invalid alpha "abc"`},
	}
	for _, c := range cases {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 3.0, 4.0)
		err := Dispatch(ctx, c.command, c.args)
		if err == nil {
			t.Errorf("%s %v: no error", c.command, c.args)
			continue
		}
		if !errors.Is(err, strconv.ErrSyntax) {
			t.Errorf("%s %v: errors.Is(err, strconv.ErrSyntax) is false for %q", c.command, c.args, err)
		}
		if !strings.HasPrefix(err.Error(), c.message) {
			t.Errorf("%s %v: message %q does not start with %q", c.command, c.args, err, c.message)
		}
		if strings.Contains(err.Error(), "strconv.") {
			t.Errorf("%s %v: the message hands back strconv's text: %q", c.command, c.args, err)
		}
	}
}
