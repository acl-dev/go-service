package master

import "testing"

func TestConfig(t *testing.T) {
	confFile := "testdata/test.cf"
	myConf := new(Config)
	myConf.InitConfig(confFile)

	expectLogfile := "/opt/soft/acl-master/var/log/aio_echo.log"
	if myConf.Get("master_log") != expectLogfile {
		t.Fatalf("Got: %s, Expect: %s",
			myConf.Get("master_log"),
			expectLogfile,
		)
	}
}
