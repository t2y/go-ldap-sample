package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/caarlos0/env"
	"github.com/go-ldap/ldap/v3"
)

type Config struct {
	URL       string `env:"LDAP_URL,required"`
	User      string `env:"LDAP_USER,required"`
	Password  string `env:"LDAP_PASSWORD,required"`
	DN_SUFFIX string `env:"DN_SUFFIX,required"`
}

func connect(c *Config) (*ldap.Conn, error) {
	conn, err := ldap.DialTLS(
		"tcp",
		c.URL,
		&tls.Config{
			InsecureSkipVerify: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if err := conn.Bind(c.User, c.Password); err != nil {
		return nil, err
	}
	//conn.Debug = true
	return conn, nil
}

func search(conn *ldap.Conn, req *ldap.SearchRequest) error {
	r, err := conn.Search(req)
	if err != nil {
		return err
	}
	for _, entry := range r.Entries {
		fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("cn"))
	}
	log.Printf("entries length: %d", len(r.Entries))
	return nil
}

func searchAsync(conn *ldap.Conn, req *ldap.SearchRequest) error {
	i := 0
	cancelNum := 200
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := conn.SearchAsync(ctx, req, 0)
	for r.Next() {
		entry := r.Entry()
		fmt.Printf("%d: %s, %v\n", i, entry.DN, entry.GetAttributeValue("cn"))
		i++
		if i == cancelNum {
			cancel()
			time.Sleep(1 * time.Second)
		}
	}
	if err := r.Err(); err != nil {
		return err
	}
	return nil
}

const (
	sizeLimit = 0
	timeLimit = 0
)

func main() {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(fmt.Errorf("failed to set: %w", err))
	}
	conn, err := connect(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	req := ldap.NewSearchRequest(
		cfg.DN_SUFFIX,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		sizeLimit,
		timeLimit,
		false,
		"(objectclass=*)",
		[]string{},
		nil,
	)
	if err := searchAsync(conn, req); err != nil {
		log.Fatal(err)
	}
	log.Println("normal end")
}
