# go-ldap-sample

go-ldap sample code

## How to run

Prepare environment variables to connect your LDAP server.

```bash
$ export LDAP_URL="localhost:636"
$ export LDAP_USER="cn=Manager,dc=example,dc=com"
$ export LDAP_PASSWORD="ldappassword"
$ export DN_SUFFIX="dc=example,dc=com"
```

Run search query.

```bash
$ go run main.go
```
