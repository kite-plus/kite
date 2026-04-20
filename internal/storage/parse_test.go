package storage

import (
	"encoding/json"
	"testing"
)

func TestParseConfig_Local(t *testing.T) {
	raw := json.RawMessage(`{"base_path":"/data/uploads","base_url":"https://example.com/files"}`)
	cfg, err := ParseConfig(DriverLocal, raw)
	if err != nil {
		t.Fatalf("ParseConfig local: %v", err)
	}
	if cfg.Driver != DriverLocal {
		t.Fatalf("driver mismatch: %s", cfg.Driver)
	}
	if cfg.Local == nil {
		t.Fatal("Local config should be populated")
	}
	if cfg.Local.BasePath != "/data/uploads" {
		t.Fatalf("BasePath: %s", cfg.Local.BasePath)
	}
	if cfg.Local.BaseURL != "https://example.com/files" {
		t.Fatalf("BaseURL: %s", cfg.Local.BaseURL)
	}
	if cfg.S3 != nil || cfg.FTP != nil {
		t.Fatal("S3 and FTP should be nil for local driver")
	}
}

func TestParseConfig_S3(t *testing.T) {
	raw := json.RawMessage(`{
		"endpoint":"s3.amazonaws.com",
		"region":"us-east-1",
		"bucket":"my-bucket",
		"access_key_id":"AKIA",
		"secret_access_key":"SECRET",
		"base_url":"https://cdn.example.com",
		"force_path_style":true
	}`)
	cfg, err := ParseConfig(DriverS3, raw)
	if err != nil {
		t.Fatalf("ParseConfig s3: %v", err)
	}
	if cfg.S3 == nil {
		t.Fatal("S3 config should be populated")
	}
	if cfg.S3.Bucket != "my-bucket" {
		t.Fatalf("Bucket: %s", cfg.S3.Bucket)
	}
	if !cfg.S3.ForcePathStyle {
		t.Fatal("ForcePathStyle should be true")
	}
}

func TestParseConfig_OSS(t *testing.T) {
	raw := json.RawMessage(`{"endpoint":"oss-cn-hangzhou.aliyuncs.com","region":"cn-hangzhou","bucket":"oss-bucket","access_key_id":"k","secret_access_key":"s","base_url":""}`)
	cfg, err := ParseConfig(DriverOSS, raw)
	if err != nil {
		t.Fatalf("ParseConfig oss: %v", err)
	}
	if cfg.Driver != DriverOSS || cfg.S3 == nil {
		t.Fatalf("OSS should parse as S3-compatible, cfg.S3=%v", cfg.S3)
	}
}

func TestParseConfig_COS(t *testing.T) {
	raw := json.RawMessage(`{"endpoint":"cos.ap-beijing.myqcloud.com","region":"ap-beijing","bucket":"cos-bucket","access_key_id":"k","secret_access_key":"s","base_url":""}`)
	cfg, err := ParseConfig(DriverCOS, raw)
	if err != nil {
		t.Fatalf("ParseConfig cos: %v", err)
	}
	if cfg.Driver != DriverCOS || cfg.S3 == nil {
		t.Fatalf("COS should parse as S3-compatible")
	}
}

func TestParseConfig_FTP(t *testing.T) {
	raw := json.RawMessage(`{"host":"ftp.example.com","port":21,"username":"ftpuser","password":"ftppass","base_path":"/upload","base_url":"https://ftp.example.com"}`)
	cfg, err := ParseConfig(DriverFTP, raw)
	if err != nil {
		t.Fatalf("ParseConfig ftp: %v", err)
	}
	if cfg.FTP == nil {
		t.Fatal("FTP config should be populated")
	}
	if cfg.Driver != DriverFTP {
		t.Fatalf("driver mismatch: %s", cfg.Driver)
	}
}

func TestParseConfig_UnknownDriver(t *testing.T) {
	_, err := ParseConfig("rclone", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
}

func TestParseConfig_InvalidJSON(t *testing.T) {
	_, err := ParseConfig(DriverLocal, json.RawMessage(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestIsS3Compatible(t *testing.T) {
	cases := []struct {
		driver string
		want   bool
	}{
		{DriverS3, true},
		{DriverOSS, true},
		{DriverCOS, true},
		{DriverLocal, false},
		{DriverFTP, false},
		{"unknown", false},
	}
	for _, c := range cases {
		if got := IsS3Compatible(c.driver); got != c.want {
			t.Errorf("IsS3Compatible(%q) = %v, want %v", c.driver, got, c.want)
		}
	}
}

func TestDriverConstants(t *testing.T) {
	if DriverLocal != "local" || DriverS3 != "s3" || DriverOSS != "oss" || DriverCOS != "cos" || DriverFTP != "ftp" {
		t.Fatal("driver constants changed unexpectedly")
	}
}
