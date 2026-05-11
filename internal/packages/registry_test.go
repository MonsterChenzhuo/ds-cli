package packages

import "testing"

func TestDolphinSchedulerSpec341(t *testing.T) {
	spec, err := DolphinSchedulerSpec("3.4.1")
	if err != nil {
		t.Fatal(err)
	}
	if spec.URL != "https://archive.apache.org/dist/dolphinscheduler/3.4.1/apache-dolphinscheduler-3.4.1-bin.tar.gz" {
		t.Fatalf("unexpected url: %s", spec.URL)
	}
	if spec.SHA512URL != spec.URL+".sha512" {
		t.Fatalf("unexpected sha url: %s", spec.SHA512URL)
	}
}

func TestMySQLDriverSpec(t *testing.T) {
	spec := MySQLDriverSpec("8.0.33")
	if spec.Filename != "mysql-connector-j-8.0.33.jar" {
		t.Fatalf("unexpected filename: %s", spec.Filename)
	}
}
