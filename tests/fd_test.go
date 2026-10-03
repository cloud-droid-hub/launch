// Copyright 2026 Cloud Droid Hub.
package fdtest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type fdRun struct {
	adb    string
	serial string
	bin    string
}

type fdReply struct {
	Begin int     `json:"begin"`
	End   int     `json:"end"`
	Limit int     `json:"limit"`
	Trace []int   `json:"trace"`
	Times []int64 `json:"times"`
}

func prep(t *testing.T) fdRun {
	t.Helper()
	root := os.Getenv("CDH_PROJECT_DIR")
	src := os.Getenv("CDH_LAUNCH_DIR")
	out := os.Getenv("CDH_FD_OUT")
	r := fdRun{os.Getenv("CDH_ADB"), os.Getenv("CDH_ADB_SERIAL"), "/data/local/tmp/launch-fd-check"}
	if root == "" || src == "" || out == "" || r.adb == "" || r.serial == "" {
		t.Fatal("explicit project, source, output, adb and serial required")
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	uid, err := exec.Command(r.adb, "-s", r.serial, "shell", "id", "-u").CombinedOutput()
	if err != nil || strings.TrimSpace(string(uid)) != "2000" {
		t.Fatalf("shell identity: %q %v", uid, err)
	}
	bin := os.Getenv("CDH_FD_BIN")
	if bin == "" {
		tool := filepath.Join(root, "output/toolchains/darwin-arm64/ndk29/toolchains/llvm/prebuilt/darwin-x86_64/bin/aarch64-linux-android28-clang++")
		bin = filepath.Join(out, "launch-fd-check")
		args := []string{"-std=c++17", "-O2", "-static-libstdc++", "-I" + filepath.Join(src, "app/src/main/cpp"), filepath.Join(src, "tests/native/open_fd.cpp"), "-o", bin}
		if os.Getenv("CDH_LAUNCH_OLD") == "1" {
			args = append(args, "-DTEST_OLD")
		}
		cmd := exec.Command(tool, args...)
		cmd.Env = append(os.Environ(), "TMPDIR="+out, "TMP="+out, "TEMP="+out, "GOTMPDIR="+out)
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("native compile: %s %v", data, err)
		}
	}
	for _, args := range [][]string{{"push", bin, r.bin}, {"shell", "chmod", "0755", r.bin}} {
		cmd := exec.Command(r.adb, append([]string{"-s", r.serial}, args...)...)
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("native transfer: %s %v", data, err)
		}
	}
	return r
}

func check(t *testing.T, r fdRun, mode, calls, cap string) {
	t.Helper()
	data, err := exec.Command(r.adb, "-s", r.serial, "shell", r.bin, mode, calls, cap).CombinedOutput()
	t.Logf("mode=%s calls=%s cap=%s output=%s error=%v", mode, calls, cap, data, err)
	if err != nil {
		t.Fatalf("native behavior: %s %v", data, err)
	}
	var got fdReply
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want, err := strconv.Atoi(cap)
	if err != nil {
		t.Fatal(err)
	}
	if got.Begin < 3 || got.End != got.Begin || got.Limit != want || len(got.Trace) != 5 || len(got.Times) != 5 {
		t.Fatalf("descriptor state: %+v", got)
	}
	for i, count := range got.Trace {
		if count != got.Begin || (calls == "5000" && got.Times[i] <= 0) || (calls != "5000" && got.Times[i] != 0) {
			t.Fatalf("round %d state: %+v", i+1, got)
		}
	}
}

func TestOpenFds(t *testing.T) {
	r := prep(t)
	for _, mode := range []string{"raw", "lib"} {
		t.Run(mode, func(t *testing.T) { check(t, r, mode, "5000", "32768") })
	}
}

func TestOpenFail(t *testing.T) {
	r := prep(t)
	for _, mode := range []string{"raw", "lib"} {
		t.Run(mode, func(t *testing.T) { check(t, r, mode, "5000", "768") })
	}
}

func TestOpenArg(t *testing.T) {
	r := prep(t)
	for _, mode := range []string{"raw", "lib"} {
		for _, calls := range []string{"0", "-1"} {
			t.Run(mode+calls, func(t *testing.T) { check(t, r, mode, calls, "768") })
		}
	}
}
