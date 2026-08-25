package cmd

import (
	"strings"
	"testing"
)

func TestSplitDSNSecretKeepsThePasswordOutOfArgv(t *testing.T) {
	const raw = "postgres://apiary:s3cr3t-pw@db.internal:5432/apiary?sslmode=require&application_name=apiary"

	sanitized, password, err := splitDSNSecret(raw)
	if err != nil {
		t.Fatalf("splitDSNSecret: %v", err)
	}
	if password != "s3cr3t-pw" {
		t.Fatalf("password = %q, want the DSN's password", password)
	}
	if strings.Contains(sanitized, "s3cr3t-pw") {
		t.Fatalf("the sanitized DSN still carries the password: %q", sanitized)
	}

	for _, want := range []string{"apiary@db.internal:5432", "/apiary", "sslmode=require", "application_name=apiary"} {
		if !strings.Contains(sanitized, want) {
			t.Fatalf("sanitized DSN lost %q: %q", want, sanitized)
		}
	}
}

func TestSplitDSNSecretDecodesAPercentEncodedPassword(t *testing.T) {
	sanitized, password, err := splitDSNSecret("postgresql://user:p%40ss%3Aword@localhost:5432/db")
	if err != nil {
		t.Fatalf("splitDSNSecret: %v", err)
	}
	if password != "p@ss:word" {
		t.Fatalf("password = %q, want %q", password, "p@ss:word")
	}
	if strings.Contains(sanitized, "p%40ss") || strings.Contains(sanitized, "p@ss") {
		t.Fatalf("sanitized DSN still carries the password: %q", sanitized)
	}
}

func TestSplitDSNSecretHandlesADSNWithNoPassword(t *testing.T) {
	sanitized, password, err := splitDSNSecret("postgres://user@localhost:5432/db?sslmode=disable")
	if err != nil {
		t.Fatalf("splitDSNSecret: %v", err)
	}
	if password != "" {
		t.Fatalf("invented a password: %q", password)
	}
	if !strings.Contains(sanitized, "user@localhost:5432/db") {
		t.Fatalf("sanitized DSN mangled: %q", sanitized)
	}
}

func TestSplitDSNSecretRefusesShapesItCannotInspect(t *testing.T) {
	for _, raw := range []string{
		"host=db.internal user=apiary password=s3cr3t dbname=apiary",
		"mysql://user:pw@localhost/db",
		"db.internal:5432/apiary",
		"",
	} {
		sanitized, password, err := splitDSNSecret(raw)
		if err == nil {
			t.Fatalf("accepted %q (sanitized=%q password=%q)", raw, sanitized, password)
		}
		if sanitized != "" || password != "" {
			t.Fatalf("refused %q but still returned data", raw)
		}
	}
}

func TestSplitDSNSecretTrimsSurroundingWhitespace(t *testing.T) {
	sanitized, password, err := splitDSNSecret("  postgres://u:pw@h:5432/d\n")
	if err != nil {
		t.Fatalf("splitDSNSecret: %v", err)
	}
	if password != "pw" {
		t.Fatalf("password = %q", password)
	}
	if strings.ContainsAny(sanitized, " \n\t") {
		t.Fatalf("sanitized DSN carries whitespace: %q", sanitized)
	}
}
