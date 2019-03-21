package core

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
)

func Shorten(s string, length int) string {
	if len(s) > length {
		s = s[:length] + "..."
	}
	return strings.Replace(s, "\n", "", -1)
}

func ProjectDir() string {
	return filepath.ToSlash(os.Getenv("GOPATH")) + "/src/github.com/jaysinco/Tools/core"
}

func SplitRobust(s string, sep string) []string {
	set := strings.Split(s, sep)
	j := 0
	for i := 0; i < len(set); i++ {
		s := strings.TrimSpace(set[i])
		if s == "" {
			continue
		}
		set[j] = s
		j++
	}
	return set[:j]
}

type Mail struct {
	From  string
	To    []string
	Token string
	Sub   string
	Body  string
}

func (m *Mail) SendBySMTP() error {
	Debug("email '%s'", m.Sub)
	domain := m.From[strings.Index(m.From, "@")+1:]
	pwd, err := base64.StdEncoding.DecodeString(m.Token)
	if err != nil {
		return err
	}
	auth := smtp.PlainAuth("", m.From, string(pwd), fmt.Sprintf("smtp.%s", domain))
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Subject: %s\r\n"+
		"\r\n%s\r\n", m.From, strings.Join(m.To, ";"), m.Sub, m.Body)
	return smtp.SendMail(fmt.Sprintf("smtp.%s:25", domain), auth,
		m.From, m.To, []byte(msg))
}
