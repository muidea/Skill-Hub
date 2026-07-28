package net

import (
	"crypto/tls"
	"gopkg.in/gomail.v2"
)

const (
	Html = "html"
	Text = "text"
)

// emailServer: smtp.example.com   smtp.163.com
// emailPort: 25
// skipTls: 是否忽略Tls
type ServerInfo struct {
	Server  string
	Port    int
	SkipTls bool
}

// user : example@example.com login smtp server user
// password: xxxxx login smtp server password
type SenderInfo struct {
	User     string
	Password string
}

// sendTo, ccTo: example@example.com;example1@163.com;example2@sina.com.cn;...
// subject:The subject of mail
// content: The content of mail
// attachment: 附件
// mimeType: mail type html or text
type MailInfo struct {
	SendTo     []string
	CCTo       []string
	Subject    string
	Content    string
	Attachment []string
	MimeType   string
}

// SendMail 发送邮件
func SendMail(mailServer *ServerInfo, mailSender *SenderInfo, mailInfo *MailInfo) error {
	goMailMsg := gomail.NewMessage()
	goMailMsg.SetHeader("From", mailSender.User)
	goMailMsg.SetHeader("To", mailInfo.SendTo...)
	if len(mailInfo.CCTo) > 0 {
		goMailMsg.SetHeader("Cc", mailInfo.CCTo...)
	}
	goMailMsg.SetHeader("Subject", mailInfo.Subject)
	if mailInfo.MimeType == Html {
		goMailMsg.SetBody("text/html", mailInfo.Content)
	} else {
		goMailMsg.SetBody("text/plain", mailInfo.Content)
	}

	for _, val := range mailInfo.Attachment {
		goMailMsg.Attach(val)
	}

	goMailDialer := gomail.NewDialer(mailServer.Server, mailServer.Port, mailSender.User, mailSender.Password)
	if mailServer.SkipTls {
		goMailDialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return goMailDialer.DialAndSend(goMailMsg)
}
