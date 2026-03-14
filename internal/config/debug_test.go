package config

import (
	"reflect"
	"testing"
)

func TestDebugPointerIssue(t *testing.T) {
	type Inner struct {
		B bool
	}
	type Outer struct {
		A *Inner
	}

	innerTrue := &Inner{B: true}
	c := Outer{A: innerTrue}

	userInner := &Inner{B: false}
	userCfg := Outer{A: userInner}

	t.Logf("Before: c.A.B=%v", c.A.B)

	// This is similar to what happens with Enabled *bool
	Merge(&c, userCfg)

	t.Logf("After: c.A.B=%v", c.A.B)

	// What about merging the inner struct directly?
	innerDst := reflect.ValueOf(innerTrue).Elem()
	innerSrc := reflect.ValueOf(userInner).Elem()

	t.Logf("innerDst.B before: %v", innerDst.FieldByName("B").Bool())
	mergeValue(innerDst.FieldByName("B"), innerSrc.FieldByName("B"))
	t.Logf("innerDst.B after: %v", innerDst.FieldByName("B").Bool())
}
