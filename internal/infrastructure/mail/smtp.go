package mail

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

type SMTP struct {
	Host, Port, From, User, Password, Mode string
}

type errorEnvio struct {
	etapa string
	causa error
}

func (e errorEnvio) Error() string {
	var respuesta *textproto.Error
	if errors.As(e.causa, &respuesta) {
		return fmt.Sprintf("%s: SMTP %d", e.etapa, respuesta.Code)
	}
	var red net.Error
	if errors.As(e.causa, &red) && red.Timeout() {
		return e.etapa + ": tiempo de espera agotado"
	}
	return fmt.Sprintf("%s: %T", e.etapa, e.causa)
}

func (e errorEnvio) Unwrap() error { return e.causa }

func (s SMTP) EnviarCodigo(destino, codigo string) error {
	return s.enviar(destino, "Codigo de recuperacion de Cotizaciones",
		fmt.Sprintf("Tu código de recuperación es: %s\r\n\r\nCaduca en 10 minutos. Si no lo solicitaste, puedes ignorar este correo.", codigo))
}

func (s SMTP) EnviarCodigoEliminacion(destino, codigo string) error {
	return s.enviar(destino, "Codigo para eliminar tu cuenta de Cotizaciones",
		fmt.Sprintf("Tu código para confirmar la eliminación de tu cuenta es: %s\r\n\r\nCaduca en 10 minutos. Si no lo solicitaste, puedes ignorar este correo.", codigo))
}

func (s SMTP) enviar(destino, asunto, cuerpo string) error {
	de, err := mail.ParseAddress(s.From)
	if err != nil {
		return errorEnvio{"formato del remitente", err}
	}
	para, err := mail.ParseAddress(destino)
	if err != nil || para.Address != destino {
		return fmt.Errorf("destinatario inválido")
	}
	dominio := de.Address[strings.LastIndexByte(de.Address, '@')+1:]
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return errorEnvio{"generar identificador del mensaje", err}
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	address := net.JoinHostPort(s.Host, s.Port)
	config := &tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	if s.Mode == "tls" {
		conn, err = tls.DialWithDialer(dialer, "tcp", address, config)
	} else {
		conn, err = dialer.Dial("tcp", address)
	}
	if err != nil {
		return errorEnvio{"conectar", err}
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return errorEnvio{"configurar tiempo de espera", err}
	}
	cliente, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return errorEnvio{"saludo SMTP", err}
	}
	defer cliente.Close()
	if s.Mode == "starttls" {
		if err := cliente.StartTLS(config); err != nil {
			return errorEnvio{"STARTTLS", err}
		}
	}
	if s.User != "" {
		if err := cliente.Auth(smtp.PlainAuth("", s.User, s.Password, s.Host)); err != nil {
			return errorEnvio{"autenticar", err}
		}
	}
	if err := cliente.Mail(de.Address); err != nil {
		return errorEnvio{"remitente", err}
	}
	if err := cliente.Rcpt(para.Address); err != nil {
		return errorEnvio{"destinatario", err}
	}
	mensaje, err := cliente.Data()
	if err != nil {
		return errorEnvio{"iniciar contenido", err}
	}
	if _, err := fmt.Fprintf(mensaje, "From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@%s>\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		de.String(), para.String(), asunto, time.Now().UTC().Format(time.RFC1123Z), hex.EncodeToString(id[:]), dominio); err != nil {
		return errorEnvio{"escribir contenido", err}
	}
	texto := quotedprintable.NewWriter(mensaje)
	if _, err := texto.Write([]byte(cuerpo + "\r\n")); err != nil {
		return errorEnvio{"escribir cuerpo", err}
	}
	if err := texto.Close(); err != nil {
		return errorEnvio{"terminar cuerpo", err}
	}
	if err := mensaje.Close(); err != nil {
		return errorEnvio{"aceptar mensaje", err}
	}
	return nil
}
