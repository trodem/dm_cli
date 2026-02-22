package agent

import (
	"testing"
)

func TestParseBuilderJSON_Valid(t *testing.T) {
	raw := `{"function_name":"sys_ping","function_code":"function sys_ping { ping localhost }","target_file":"3_System_Toolkit.ps1","is_new_toolkit":false,"new_prefix":"","explanation":"pings localhost"}`
	result, err := parseBuilderJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.FunctionName != "sys_ping" {
		t.Fatalf("expected function_name='sys_ping', got %q", result.FunctionName)
	}
	if result.TargetFile != "3_System_Toolkit.ps1" {
		t.Fatalf("expected target_file='3_System_Toolkit.ps1', got %q", result.TargetFile)
	}
	if result.IsNewToolkit {
		t.Fatal("expected is_new_toolkit=false")
	}
}

func TestParseBuilderJSON_NewToolkit(t *testing.T) {
	raw := `{"function_name":"net_scan","function_code":"function net_scan {}","target_file":"Net_Toolkit.ps1","is_new_toolkit":true,"new_prefix":"net","explanation":"network scanner"}`
	result, err := parseBuilderJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsNewToolkit {
		t.Fatal("expected is_new_toolkit=true")
	}
	if result.NewPrefix != "net" {
		t.Fatalf("expected new_prefix='net', got %q", result.NewPrefix)
	}
}

func TestParseBuilderJSON_MarkdownFence(t *testing.T) {
	raw := "```json\n{\"function_name\":\"test_fn\",\"function_code\":\"function test_fn {}\",\"target_file\":\"Test.ps1\"}\n```"
	result, err := parseBuilderJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.FunctionName != "test_fn" {
		t.Fatalf("expected function_name='test_fn', got %q", result.FunctionName)
	}
}

func TestParseBuilderJSON_Empty(t *testing.T) {
	_, err := parseBuilderJSON("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseBuilderJSON_NoJSON(t *testing.T) {
	_, err := parseBuilderJSON("no json here at all")
	if err == nil {
		t.Fatal("expected error when no JSON object found")
	}
}

func TestParseBuilderJSON_InvalidJSON(t *testing.T) {
	_, err := parseBuilderJSON("{invalid json content here")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseBuilderJSON_EscapedNewlines(t *testing.T) {
	raw := `{"function_name":"fn","function_code":"line1\\nline2","target_file":"f.ps1"}`
	result, err := parseBuilderJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.FunctionCode != "line1\nline2" {
		t.Fatalf("expected escaped newlines to be replaced, got %q", result.FunctionCode)
	}
}
