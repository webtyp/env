//go:build wasm

package tests

import (
	"syscall/js"
	"testing"

	"webtyp.com/env"
)

func setupContext(t *testing.T, vars map[string]string) {
	t.Helper()
	env := js.Global().Get("Object").New()
	for k, v := range vars {
		env.Set(k, v)
	}
	ctx := js.Global().Get("Object").New()
	ctx.Set("env", env)
	prev := js.Global().Get("context")
	js.Global().Set("context", ctx)
	t.Cleanup(func() {
		if prev.IsUndefined() {
			js.Global().Delete("context")
		} else {
			js.Global().Set("context", prev)
		}
	})
}

func TestWasm_Get_ReadsFromContext(t *testing.T) {
	setupContext(t, map[string]string{"WASM_KEY": "wasm-value"})
	if got := env.Get("WASM_KEY"); got != "wasm-value" {
		t.Errorf("got %q, want wasm-value", got)
	}
}

func TestWasm_Get_MissingReturnsEmpty(t *testing.T) {
	setupContext(t, nil)
	if got := env.Get("WASM_MISSING_123"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestWasm_Lookup_FoundAndMissing(t *testing.T) {
	setupContext(t, map[string]string{"WASM_LOOKUP": "found"})
	if v, ok := env.Lookup("WASM_LOOKUP"); !ok || v != "found" {
		t.Errorf("Lookup found=%v %q, want true found", ok, v)
	}
	if _, ok := env.Lookup("WASM_NOT_FOUND_123"); ok {
		t.Error("Lookup missing should be not ok")
	}
}

func TestWasm_GetOr_FallsBack(t *testing.T) {
	setupContext(t, nil)
	if got := env.GetOr("WASM_MISSING_123", "fallback"); got != "fallback" {
		t.Errorf("got %q, want fallback", got)
	}
	setupContext(t, map[string]string{"WASM_HAS": "primary"})
	if got := env.GetOr("WASM_HAS", "fallback"); got != "primary" {
		t.Errorf("got %q, want primary", got)
	}
}

func TestWasm_Require_FoundAndMissing(t *testing.T) {
	setupContext(t, map[string]string{"WASM_REQ": "present"})
	if v, err := env.Require("WASM_REQ"); err != nil || v != "present" {
		t.Errorf("Require err %v %q", err, v)
	}
	setupContext(t, nil)
	if _, err := env.Require("WASM_MISSING_123"); err == nil {
		t.Error("Require missing should error")
	}
}

func TestWasm_Lookup_NoContextReturnsMissing(t *testing.T) {
	prev := js.Global().Get("context")
	js.Global().Delete("context")
	t.Cleanup(func() {
		if !prev.IsUndefined() {
			js.Global().Set("context", prev)
		}
	})
	if _, ok := env.Lookup("ANY"); ok {
		t.Error("Lookup without context should be not ok")
	}
	if got := env.Get("ANY"); got != "" {
		t.Errorf("Get without context got %q, want empty", got)
	}
}
