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

func searchAsyncWithCancel(conn *ldap.Conn, req *ldap.SearchRequest) error {
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

func searchAsync(conn *ldap.Conn, req *ldap.SearchRequest) error {
	i := 0
	ctx := context.Background()
	r := conn.SearchAsync(ctx, req, 0)
	for r.Next() {
		entry := r.Entry()
		fmt.Printf("%d: %s, %v\n", i, entry.DN, entry.GetAttributeValue("cn"))
		i++
		referral := r.Referral()
		if referral != "" {
			fmt.Println("🐙", referral)
		}
		controlls := r.Controls()
		if len(controlls) != 0 {
			fmt.Println("🦑", controlls)
		}
	}
	if err := r.Err(); err != nil {
		return err
	}
	return nil
}

func syncRepl(conn *ldap.Conn, req *ldap.SearchRequest) error {
	i := 0
	ctx := context.Background()
	//mode := ldap.SyncRequestModeRefreshOnly
	mode := ldap.SyncRequestModeRefreshAndPersist
	cookie := []byte("rid=000,csn=20230719002941.016654Z#000000#000#000000")
	r := conn.Syncrepl(ctx, req, 512, mode, cookie, false)
	for r.Next() {
		fmt.Printf("%d: retrieved a response\n", i)
		entry := r.Entry()
		if entry != nil {
			fmt.Printf("- entry: %s\n", entry.DN)
			for _, a := range entry.Attributes {
				fmt.Printf("  - %s: %s\n", a.Name, a.Values)
			}
		}
		fmt.Printf("- referral: %v\n", r.Referral())
		for _, c := range r.Controls() {
			fmt.Printf("- control: %s\n", c.String())
		}
		i++
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
		cfg.DN_SUFFIX,          // Base DN
		ldap.ScopeWholeSubtree, // Scope
		ldap.NeverDerefAliases, // Deref Aliases
		sizeLimit,              // Size Limit
		timeLimit,              // Time Limit
		false,                  // Types Only
		"(objectclass=*)",      // Filter
		nil,                    // Attributes
		nil,                    // Controls
	)
	if err := syncRepl(conn, req); err != nil {
		log.Fatal(err)
	}
	log.Println("normal end")
}
