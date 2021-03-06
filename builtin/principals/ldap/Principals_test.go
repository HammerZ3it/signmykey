package ldap

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestPrincipals(t *testing.T) {
	// TODO: Add LDAP mocking
	t.Skip()

	ldap := &Principals{
		Address:         "127.0.0.1",
		Port:            636,
		BindUser:        "CN=fakebinduser,OU=Users,DC=test,DC=domain",
		BindPassword:    "fakebindpasswd",
		UserSearchBase:  "OU=Users,DC=test,DC=domain",
		UserSearchStr:   "(&(objectClass=organizationalPerson)(sAMAccountName=%s))",
		GroupSearchBase: "OU=Groups,DC=test,DC=domain",
		GroupSearchStr:  "(&(objectClass=group)(member=%s))",
		UseTLS:          true,
		TLSVerify:       true,
		Prefix:          "smk-",
	}

	_, principals, err := ldap.Get(context.Background(), []byte("{\"user\": \"fakeuser\"}"))
	if err != nil {
		t.Logf("%s", err)
		t.Fail()
		return
	} else if len(principals) == 0 {
		t.Logf("empty list of principals")
		t.Fail()
		return
	}
}

func TestGetCN(t *testing.T) {
	cases := []struct {
		list    []string
		expList []string
	}{
		{
			[]string{
				"CN=grouptest1,OU=Groups,DC=test,DC=domain",
				"CN=grouptest-2,OU=Groups,DC=test,DC=domain",
				"DN=group3,OU=Groups,DC=test,DC=domain",
				"CN=,OU=Groups,DC=test,DC=domain",
				"CN=group4_test,CN=Groups,DC=test,DC=domain",
			},
			[]string{"grouptest1", "grouptest-2", "group4_test"},
		},
	}

	for _, c := range cases {
		cnList := getCN(c.list)
		assert.Equal(t, c.expList, cnList)
	}
}

func TestFilterByPrefix(t *testing.T) {
	cases := []struct {
		prefix  string
		list    []string
		expList []string
	}{
		{"smk-",
			[]string{"group1", "smk-group2", "group3-smk", "smk-group4", ""},
			[]string{"group2", "group4"},
		},
		{"",
			[]string{"group1", "smk-group2", "group3-smk", "smk-group4", ""},
			[]string{"group1", "smk-group2", "group3-smk", "smk-group4", ""},
		},
		{"smk-",
			[]string{},
			[]string{},
		},
	}

	for _, c := range cases {
		filtList := filterByPrefix(c.prefix, c.list)
		assert.Equal(t, c.expList, filtList)
	}
}

func TestPrincipalsInit(t *testing.T) {
	cases := []struct {
		config []byte
		auth   Principals
		err    string
	}{
		{
			[]byte(""),
			Principals{},
			"Missing config entries (ldapAddr, ldapPort, ldapTLS, ldapTLSVerify, ldapBindUser, ldapBindPassword, ldapUserBase, ldapUserSearch, ldapGroupBase, ldapGroupSearch) for Principals",
		},
		{
			[]byte("ldapAddr: 127.0.0.1"),
			Principals{},
			"Missing config entries (ldapPort, ldapTLS, ldapTLSVerify, ldapBindUser, ldapBindPassword, ldapUserBase, ldapUserSearch, ldapGroupBase, ldapGroupSearch) for Principals",
		},
		{
			[]byte(`
ldapAddr: 127.0.0.1
ldapUserSearch: "(&(objectClass=organizationalPerson)(sAMAccountName=%s))"
`),
			Principals{},
			"Missing config entries (ldapPort, ldapTLS, ldapTLSVerify, ldapBindUser, ldapBindPassword, ldapUserBase, ldapGroupBase, ldapGroupSearch) for Principals",
		},
		{
			[]byte(`
ldapAddr: 127.0.0.1
ldapPort: 636
ldapTLS: True
ldapTLSVerify: True
ldapBindUser: binduser
ldapBindPassword: bindpassword
ldapUserBase: "DC=fake,DC=org"
ldapUserSearch: "(&(objectClass=organizationalPerson)(sAMAccountName=%s))"
ldapGroupBase: "DC=fake,DC=org"
ldapGroupSearch: "(&(objectClass=group)(member=%s))"
`),
			Principals{
				Address:         "127.0.0.1",
				Port:            636,
				UseTLS:          true,
				TLSVerify:       true,
				BindUser:        "binduser",
				BindPassword:    "bindpassword",
				UserSearchBase:  "DC=fake,DC=org",
				UserSearchStr:   "(&(objectClass=organizationalPerson)(sAMAccountName=%s))",
				GroupSearchBase: "DC=fake,DC=org",
				GroupSearchStr:  "(&(objectClass=group)(member=%s))",
				TransformCase:   "none",
			},
			"",
		},
		{
			[]byte(`
ldapAddr: myldapserver.local
ldapPort: 389
ldapTLS: False
ldapTLSVerify: False
ldapBindUser: binduser
ldapBindPassword: bindpassword
ldapUserBase: "DC=fake,DC=org"
ldapUserSearch: "(&(objectClass=organizationalPerson)(sAMAccountName=%s))"
ldapGroupBase: "DC=fake,DC=org"
ldapGroupSearch: "(&(objectClass=group)(member=%s))"
`),
			Principals{
				Address:         "myldapserver.local",
				Port:            389,
				UseTLS:          false,
				TLSVerify:       false,
				BindUser:        "binduser",
				BindPassword:    "bindpassword",
				UserSearchBase:  "DC=fake,DC=org",
				UserSearchStr:   "(&(objectClass=organizationalPerson)(sAMAccountName=%s))",
				GroupSearchBase: "DC=fake,DC=org",
				GroupSearchStr:  "(&(objectClass=group)(member=%s))",
				TransformCase:   "none",
			},
			"",
		},
		{
			[]byte(`
ldapAddr: myldapserver.local
ldapTLS: False
ldapTLSVerify: False
ldapBindUser: binduser
ldapUserBase: "DC=fake,DC=org"
ldapUserSearch: "(&(objectClass=organizationalPerson)(sAMAccountName=%s))"
ldapGroupBase: "DC=fake,DC=org"
ldapGroupSearch: "(&(objectClass=group)(member=%s))"
`),
			Principals{},
			"Missing config entries (ldapPort, ldapBindPassword) for Principals",
		},
	}

	for _, c := range cases {
		testConfig := viper.New()
		testConfig.SetConfigType("yaml")
		err := testConfig.ReadConfig(bytes.NewBuffer(c.config))
		if err != nil {
			t.Error(err)
		}

		auth := Principals{}
		err = auth.Init(testConfig)

		assert.EqualValues(t, c.auth, auth)
		if c.err == "" {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, c.err)
		}
	}

}
