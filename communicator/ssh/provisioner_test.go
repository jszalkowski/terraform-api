package ssh

import (
	"testing"

	"github.com/xanzy/terraform-api/terraform"
)

func TestProvisioner_connInfo(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "root",
				"password":    "supersecret",
				"private_key": "someprivatekeycontents",
				"host":        "127.0.0.1",
				"port":        "22",
				"timeout":     "30s",

				"bastion_host": "127.0.1.1",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.User != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Password != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.PrivateKey != "someprivatekeycontents" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Host != "127.0.0.1" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Port != 22 {
		t.Fatalf("bad: %v", conf)
	}
	if conf.Timeout != "30s" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.ScriptPath != DefaultScriptPath {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionHost != "127.0.1.1" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPort != 22 {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionUser != "root" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPassword != "supersecret" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPrivateKey != "someprivatekeycontents" {
		t.Fatalf("bad: %v", conf)
	}
}

func TestProvisioner_connInfoLegacy(t *testing.T) {
	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":         "ssh",
				"key_file":     "/my/key/file.pem",
				"bastion_host": "127.0.1.1",
			},
		},
	}

	conf, err := parseConnectionInfo(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if conf.PrivateKey != "/my/key/file.pem" {
		t.Fatalf("bad: %v", conf)
	}
	if conf.BastionPrivateKey != "/my/key/file.pem" {
		t.Fatalf("bad: %v", conf)
	}
}
