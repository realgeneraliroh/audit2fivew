// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

package avc_test

import (
	"strings"
	"testing"

	"audit2fivew/internal/avc"
)

const sampleAVC = `type=AVC msg=audit(1609459200.123:456): avc:  denied  { read write } for  pid=1234 comm="httpd" name="index.html" dev="sda1" ino=78901 scontext=system_u:system_r:httpd_t:s0 tcontext=unconfined_u:object_r:user_home_t:s0 tclass=file permissive=0`

func TestParse(t *testing.T) {
	d, err := avc.Parse(sampleAVC)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if d.Serial != "456" {
		t.Errorf("Serial = %q, want %q", d.Serial, "456")
	}
	if len(d.Permissions) != 2 || d.Permissions[0] != "read" || d.Permissions[1] != "write" {
		t.Errorf("Permissions = %v, want [read write]", d.Permissions)
	}
	if d.TClass != "file" {
		t.Errorf("TClass = %q, want %q", d.TClass, "file")
	}
	if d.Comm != "httpd" {
		t.Errorf("Comm = %q, want %q", d.Comm, "httpd")
	}
	if d.PID != 1234 {
		t.Errorf("PID = %d, want %d", d.PID, 1234)
	}
	if d.Name != "index.html" {
		t.Errorf("Name = %q, want %q", d.Name, "index.html")
	}
	if d.SContext != "system_u:system_r:httpd_t:s0" {
		t.Errorf("SContext = %q, want system_u:system_r:httpd_t:s0", d.SContext)
	}
	if d.TContext != "unconfined_u:object_r:user_home_t:s0" {
		t.Errorf("TContext = %q, want unconfined_u:object_r:user_home_t:s0", d.TContext)
	}
	if d.SType() != "httpd_t" {
		t.Errorf("SType = %q, want %q", d.SType(), "httpd_t")
	}
	if d.TType() != "user_home_t" {
		t.Errorf("TType = %q, want %q", d.TType(), "user_home_t")
	}
	if d.Permissive {
		t.Error("Permissive = true, want false")
	}
	if d.ContentHash == "" {
		t.Error("ContentHash is empty")
	}
}

func TestParse_NotAVC(t *testing.T) {
	_, err := avc.Parse("type=SYSCALL msg=audit(1609459200.123:456): something else")
	if err == nil {
		t.Error("expected error for non-AVC line, got nil")
	}
}

func TestParse_Permissive(t *testing.T) {
	line := strings.Replace(sampleAVC, "permissive=0", "permissive=1", 1)
	d, err := avc.Parse(line)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !d.Permissive {
		t.Error("Permissive = false, want true")
	}
}

func TestParse_WithPath(t *testing.T) {
	line := `type=AVC msg=audit(1609459200.000:100): avc:  denied  { getattr } for  pid=999 comm="nginx" path="/var/www/html/index.html" dev="dm-0" ino=55555 scontext=system_u:system_r:nginx_t:s0 tcontext=system_u:object_r:var_t:s0 tclass=file permissive=0`
	d, err := avc.Parse(line)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if d.Path != "/var/www/html/index.html" {
		t.Errorf("Path = %q, want /var/www/html/index.html", d.Path)
	}
	if d.Comm != "nginx" {
		t.Errorf("Comm = %q, want nginx", d.Comm)
	}
}

func TestGenerateTE(t *testing.T) {
	d, _ := avc.Parse(sampleAVC)
	te := d.GenerateTE()

	if !strings.Contains(te, "module my_httpd_t_user_home_t_fix 1.0;") {
		t.Errorf("TE missing module declaration:\n%s", te)
	}
	if !strings.Contains(te, "type httpd_t;") {
		t.Errorf("TE missing source type:\n%s", te)
	}
	if !strings.Contains(te, "type user_home_t;") {
		t.Errorf("TE missing target type:\n%s", te)
	}
	if !strings.Contains(te, "allow httpd_t user_home_t:file { read write };") {
		t.Errorf("TE missing allow rule:\n%s", te)
	}
}

func TestGenerateCommands(t *testing.T) {
	d, _ := avc.Parse(sampleAVC)
	cmds := d.GenerateCommands()

	if !strings.Contains(cmds, "checkmodule") {
		t.Errorf("commands missing checkmodule:\n%s", cmds)
	}
	if !strings.Contains(cmds, "semodule_package") {
		t.Errorf("commands missing semodule_package:\n%s", cmds)
	}
	if !strings.Contains(cmds, "semodule -i") {
		t.Errorf("commands missing semodule -i:\n%s", cmds)
	}
}

func TestHashStability(t *testing.T) {
	d1, _ := avc.Parse(sampleAVC)
	d2, _ := avc.Parse(sampleAVC)
	if d1.ContentHash != d2.ContentHash {
		t.Errorf("identical lines produced different hashes: %q vs %q",
			d1.ContentHash, d2.ContentHash)
	}
}

